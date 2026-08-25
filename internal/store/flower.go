package store

import (
	"strconv"
	"strings"
	"time"
)

// 稀兽花种(实时地图页的花种图层,见 docs/map.md 8)。
//
// 一行 = 大世界上当前开着的一朵花。描述性列(位置/精灵/有效期)来自花种列表(0x0375),
// 检测列(state/glass/lv)来自逐朵查询(0x0338),两边各写各的:重新收到列表时只覆盖描述列,
// 已检测出的炫彩结果不受影响。
const (
	FlowerUndetected = 0 // 未检测:还没为这朵花收到过 0x0338
	FlowerPlain      = 1 // 普通:检测过,glass_type = GT_NULL
	FlowerGlassy     = 2 // 炫彩:检测过,glass_type != GT_NULL
)

// FlowerRow 是一朵花的完整状态(库内一行)。
// ObjID 是 uint64 且**超出 int64 范围**(实测 9279722995167894922 > 2^63-1),故按 TEXT 存。
type FlowerRow struct {
	ObjID     uint64 `json:"-"`
	CfgID     uint32 `json:"-"`
	Star      int32  `json:"star,omitempty"`
	PetBase   uint32 `json:"-"`
	ContentID uint32 `json:"-"`
	SpecID    uint32 `json:"-"`
	EndTS     int64  `json:"endTs,omitempty"`
	State     int    `json:"st"`
	Glass     string `json:"glass,omitempty"` // 炫彩外观描述(仅 State=FlowerGlassy)
}

// ReplaceFlowers 按花种列表全量刷新:列表里的花 upsert 描述列(检测结果保留),
// 不在列表里的行删除(那朵花已经刷掉了)。
func (sc *Scoped) ReplaceFlowers(rows []FlowerRow) error {
	tx, err := sc.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	keep := make([]any, 0, len(rows)+1)
	keep = append(keep, sc.account)
	ins, err := tx.Prepare(`INSERT INTO flower_seed(account, obj_id, cfg_id, star, petbase, content_id, spec_id, end_ts, state, glass, updated_at)
		VALUES(?,?,?,?,?,?,?,?,0,'',?)
		ON CONFLICT(account, obj_id) DO UPDATE SET
			cfg_id=excluded.cfg_id, star=excluded.star, petbase=excluded.petbase,
			content_id=excluded.content_id, spec_id=excluded.spec_id, end_ts=excluded.end_ts,
			updated_at=excluded.updated_at`)
	if err != nil {
		return err
	}
	defer ins.Close()
	now := time.Now().Unix()
	for _, r := range rows {
		id := strconv.FormatUint(r.ObjID, 10)
		if _, err = ins.Exec(sc.account, id, r.CfgID, r.Star, r.PetBase, r.ContentID, r.SpecID, r.EndTS, now); err != nil {
			return err
		}
		keep = append(keep, id)
	}
	del := `DELETE FROM flower_seed WHERE account=?`
	if len(rows) > 0 {
		del += ` AND obj_id NOT IN (?` + strings.Repeat(",?", len(rows)-1) + `)`
	}
	if _, err = tx.Exec(del, keep...); err != nil {
		return err
	}
	return tx.Commit()
}

// SetFlowerDetected 记录一朵花的检测结果(0x0338)。描述列一并写入:玩家可能在收到花种列表
// 之前就点开了某朵花,那时库里还没有这一行,单靠这条消息也能把它显示出来。
func (sc *Scoped) SetFlowerDetected(r FlowerRow) error {
	_, err := sc.db.Exec(`INSERT INTO flower_seed(account, obj_id, cfg_id, star, petbase, content_id, spec_id, end_ts, state, glass, updated_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(account, obj_id) DO UPDATE SET
			cfg_id=excluded.cfg_id, star=excluded.star, petbase=excluded.petbase,
			content_id=excluded.content_id, spec_id=excluded.spec_id, end_ts=excluded.end_ts,
			state=excluded.state, glass=excluded.glass, updated_at=excluded.updated_at`,
		sc.account, strconv.FormatUint(r.ObjID, 10), r.CfgID, r.Star, r.PetBase, r.ContentID,
		r.SpecID, r.EndTS, r.State, r.Glass, time.Now().Unix())
	return err
}

// ResetFlowerDetection 把一朵花打回未检测:战斗里捕捉/击败后**花本身不消失**(还能再挑战),
// 但里面那只精灵重投了——种族/等级/炫彩全部作废,检测得从头再来一次。
// 只清里面那只精灵的事实(petbase/glass/state);花自身的位置、血脉(cfg_id)、星级、有效期照旧,
// 那些属于花实体而不是精灵(等级由星级算出,故一并不动)。
// 万一血脉也跟着重投,下一次列表/详情会把它改回来。
func (sc *Scoped) ResetFlowerDetection(objID uint64) error {
	_, err := sc.db.Exec(`UPDATE flower_seed SET petbase=0, glass='', state=?, updated_at=?
		WHERE account=? AND obj_id=?`,
		FlowerUndetected, time.Now().Unix(), sc.account, strconv.FormatUint(objID, 10))
	return err
}

// Flowers 返回本账号在 now 时刻仍有效的花种。**过期行不返回也不显示**:过了 end_ts 说明
// 服务器早已重投,库里这行的位置/精灵/检测结果全部作废,显示它只会误导;真正的删除留给
// PurgeFlowers(在消费管线里做,读路径不写库)。now 传消息时刻,离线回放才对得上。
func (sc *Scoped) Flowers(now int64) []FlowerRow {
	rows, err := sc.rdb.Query(`SELECT obj_id, cfg_id, star, petbase, content_id, spec_id, end_ts, state, glass
		FROM flower_seed WHERE account=? AND (end_ts=0 OR end_ts>?) ORDER BY obj_id`, sc.account, now)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []FlowerRow
	for rows.Next() {
		var r FlowerRow
		var id string
		if rows.Scan(&id, &r.CfgID, &r.Star, &r.PetBase, &r.ContentID, &r.SpecID,
			&r.EndTS, &r.State, &r.Glass) != nil {
			continue
		}
		if v, err := strconv.ParseUint(id, 10, 64); err == nil {
			r.ObjID = v
			out = append(out, r)
		}
	}
	return out
}

// PurgeFlowers 删除已过有效期的行;返回是否真的删掉了什么(据此决定要不要重推前端)。
func (sc *Scoped) PurgeFlowers(now int64) bool {
	res, err := sc.db.Exec(`DELETE FROM flower_seed WHERE account=? AND end_ts>0 AND end_ts<=?`,
		sc.account, now)
	if err != nil {
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

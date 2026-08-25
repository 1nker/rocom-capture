package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/whoisnian/rocom-capture/internal/store"
)

// 稀兽花种图层(见 docs/map.md 8)。
//
// 与固定点位的 POI 不同,花种每天(命定每两周)在候选点里重投一遍,当前开着哪几朵只有流量
// 知道,故点位存在库里(flower_seed 表)而不是 names.json;这里把库里的行投影成地图标记。
// 检测状态(未检测/普通/炫彩)同样是逐朵攒出来的,见 pipeline/flowers.go。

// visitFlowerSet 是「正在参观的那个世界」的花种:主人 uin + 那套花。
// 参观期间地图上画的是**好友的**花(方便提醒他去捉),而不是自己那套;离开即清空。
type visitFlowerSet struct {
	Owner uint32
	Rows  []store.FlowerRow
}

// SetVisitFlowers 记录/清空参观中那个世界的花种(owner=0 表示回到自己的世界)。
func (s *Server) SetVisitFlowers(account string, owner uint32, rows []store.FlowerRow) {
	if account == "" {
		return
	}
	s.posMu.Lock()
	defer s.posMu.Unlock()
	if owner == 0 {
		delete(s.visitFlowers, account)
		return
	}
	s.visitFlowers[account] = visitFlowerSet{Owner: owner, Rows: rows}
}

// visitFlowersOf 取当前账号正在参观的那套花(不在参观时 ok=false)。
func (s *Server) visitFlowersOf(account string) (visitFlowerSet, bool) {
	s.posMu.Lock()
	defer s.posMu.Unlock()
	v, ok := s.visitFlowers[account]
	return v, ok
}

// flowerMark 是推给前端的一朵花:底图归一化坐标(与玩家位置同一投影)+ 图标 + 检测结果。
type flowerMark struct {
	ID   string  `json:"id"`              // npc_obj_id(uint64 超出 JS 安全整数,用字符串)
	Res  int32   `json:"res"`             // 所属 scene_res_cfg_id(前端按当前场景筛)
	U    float64 `json:"u"`               // 底图归一化坐标
	V    float64 `json:"v"`               //
	Name string  `json:"n"`               // 花里那只精灵的形态名(火神…);查不到时为空
	Icon string  `json:"icon"`            // 血脉花图相对路径 flower/<原名>.webp
	Lv   int32   `json:"lv,omitempty"`    // 由星级查表算出(gamedata.FlowerLevel),列表一到就有
	St   int     `json:"st"`              // 0 未检测 / 1 普通 / 2 炫彩(store.Flower*)
	Glas string  `json:"glass,omitempty"` // 炫彩外观描述(仅 st=2)
}

// flowerMarks 把花种投影成地图标记(无底图场景/未收录候选点的行跳过)。
// 正在参观别人的世界时画的是**那个世界**的花(内存里的一套),否则是自己库里的。
// 第二个返回值是参观中那个世界的主人 uin(0 = 在自己的世界)。
func (s *Server) flowerMarks(acc string, now time.Time) ([]flowerMark, uint32) {
	rows := s.store.For(acc).Flowers(now.Unix())
	var owner uint32
	if v, ok := s.visitFlowersOf(acc); ok {
		rows, owner = v.Rows, v.Owner
	}
	out := []flowerMark{}
	for _, r := range rows {
		spot, ok := s.db.FlowerSpot(r.ContentID)
		if !ok {
			continue
		}
		u, v, ok := s.db.Project(uint32(spot.Res), spot.X, spot.Y)
		if !ok {
			continue
		}
		m := flowerMark{
			ID: strconv.FormatUint(r.ObjID, 10), Res: spot.Res, U: u, V: v,
			Icon: s.db.FlowerIcon(r.CfgID), Lv: s.db.FlowerLevel(r.Star, r.SpecID),
			St: r.State, Glas: r.Glass,
		}
		if info, ok := s.db.PetBase(r.PetBase); ok {
			m.Name = info.Name
		}
		out = append(out, m)
	}
	return out, owner
}

// PushFlowers 广播当前账号的花种图层(消费管线在列表/检测/战斗结算变动后调用)。
// now 取消息时刻而非挂钟:离线回放的包时间是几小时前的,用挂钟一比花全过期了。
func (s *Server) PushFlowers(account string, now time.Time) {
	if account == "" {
		return
	}
	marks, owner := s.flowerMarks(account, now)
	s.Hub().Broadcast("flowers", account, map[string]any{
		"account": account, "flowers": marks, "visit": owner != 0,
	})
}

// handleFlowers 返回当前账号的花种图层(实时地图页加载时的初值;之后由 SSE flowers 推增量)。
// 花种的有效期是服务器给的绝对时刻,这里按挂钟筛——页面是给「现在」看的。
func (s *Server) handleFlowers(w http.ResponseWriter, r *http.Request) {
	marks, owner := s.flowerMarks(s.acct(r), time.Now())
	writeJSON(w, map[string]any{"flowers": marks, "visit": owner != 0})
}

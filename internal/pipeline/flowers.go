package pipeline

import (
	"time"

	"github.com/whoisnian/rocom-capture/internal/capture"
	"github.com/whoisnian/rocom-capture/internal/gamedata"
	"github.com/whoisnian/rocom-capture/internal/scene"
	"github.com/whoisnian/rocom-capture/internal/store"
)

// ---- 实时地图的稀兽花种图层(见 docs/map.md 8)----
//
// 大世界同时开着二十来朵花种,每朵关着一只混血精灵。两条消息各给一半:
//
//	0x0375 花种列表  → 全集(位置/精灵/星级/有效期),**不含炫彩**;客户端登录后自动请求一次,
//	                   玩家开花种面板再请求一次。等级不在下发字段里,由星级查表算出
//	                   (gamedata.FlowerLevel),故列表一到就有等级。
//	0x0338 单朵详情  → 那一朵的炫彩;玩家在地图上点中某朵花、或走到花前时客户端才请求。
//
// 故炫彩只能**逐朵攒**:列表里的花先记「未检测」,玩家点过哪朵就把哪朵的结果补上。
// 本项目是被动抓包,不能代替客户端去查,没点过的花永远是未检测——前端因此要留三态,
// 绝不能把「没检测过」显示成「普通」。
//
// **参观好友世界时看到的是好友的花**:传送去好友的世界再打开花种面板,服务器给的是**那个世界**
// 的花(obj_id 与自己的完全不同、里面的精灵也不是自己的)。这些花照样画在地图上——正好用来
// 提醒好友去捉——但**绝不能记进自己的库**(否则自己那 23 朵会被整片换掉)。故分两套:
//   - 自己的:落 flower_seed 表,跨会话保留;
//   - 参观中的:只放在连接状态里(connState.visitFlowers),离开那个世界即弃。
// 「现在在谁的世界里」由 0x039d(ZONE_SCENE_PLAYER_VISIT_INFO_SYNC_NOTIFY)的 online_visit_owner
// 给出(传送通知之前就到);回包本身带 visit_flower_seed_boss_datas 则说明这份讲的是别人的花。
//
// **刷新即作废**,两条路子:
//   ①**整朵重投**:每天凌晨 4 点(命定花种每两周周五凌晨 4 点)。不必自己算这些时刻——
//     服务器把每朵花的有效期终点直接写在 end_timestamp 里(实测普通花种 = 次日 04:00:00、
//     命定花种 = 周五 03:59:59),过期的行连同检测结果一并丢弃(store.PurgeFlowers);
//     新的一批在下一次列表里以新的 npc_obj_id 出现,state 自然回到未检测。
//   ②**只重投里面的精灵**:打完一场(捕捉/击败)后花**不会消失**、还能再挑战,但里面换了
//     另一只精灵。此时只把该行打回未检测(onFlowerHarvested),花的位置/血脉/有效期照旧。

// onVisitInfo 处理 0x039d:记下现在站在谁的世界里。换了世界就把参观中的花种清空并重推
// (回自己世界时重推的就是库里自己的那套)。
//
// **`online_visit_owner` 是「这个世界的主人」,不是「有没有人在串门」**:A 加入 B 的世界后,
// 客串的 A 与做东的 B **都会**收到这条通知,只是 A 收到的是 B 的 uin、B 收到的是自己的。
// 只判「非 0」会把世界主也当成访客——他明明看的是自己的花,却被标成好友的(用户实测)。
// 故要跟本账号的 uin 比一下:等于自己 ⇒ 就在自己家里,照常走库里那套。
// 口径始终是:地图上画的是**脚下这个世界**的花。
func (p *Pipeline) onVisitInfo(conn, acc string, body []byte, now time.Time) {
	owner := scene.ParseVisitOwner(body)
	if self, ok := uidFromAcc(acc); ok && owner == self {
		owner = 0 // 世界主收到的是自己的 uin:这是自己家,不是在参观
	}
	cs := p.conn(conn)
	if cs.visitOwner == owner {
		return
	}
	cs.visitOwner, cs.visitFlowers = owner, nil
	p.pushFlowers(conn, acc, now)
}

// onFlowerList 收下花种列表(0x0375):全量替换那一套花种行,已检测出的结果保留
// (同一朵花的 obj_id 不变,检测列不在替换范围内)。
// 列表里同时有世界首领/传说精灵那两支,ParseFlowerList 只取花种那一支;这里再按
// NPC 表确认一次是花种(万一某天花种列表里混进别的 NPC,宁可不画也不画错)。
func (p *Pipeline) onFlowerList(m capture.Message, acc string) {
	list, ok := scene.ParseFlowerList(m.AppBody)
	if !ok {
		return // 服务器拒绝或没有花种那一支:原样保留现有的,别拿空列表覆盖
	}
	rows := make([]store.FlowerRow, 0, len(list.Seeds))
	for _, f := range list.Seeds {
		if _, ok := p.db.FlowerNpc(f.CfgID); !ok {
			continue
		}
		rows = append(rows, store.FlowerRow{
			ObjID: f.ObjID, CfgID: f.CfgID, Star: f.Star, PetBase: f.PetBase,
			ContentID: f.ContentID, SpecID: f.SpecID, EndTS: f.EndTS,
		})
	}
	cs := p.conn(m.Session)
	if list.Visiting {
		if cs.visitOwner == 0 {
			return // 还没收到 0x039d 就来了参观列表:不知道是谁的世界,宁可不画
		}
		cs.visitFlowers = make(map[uint64]store.FlowerRow, len(rows))
		for _, r := range rows {
			cs.visitFlowers[r.ObjID] = r
		}
		p.pushFlowers(m.Session, acc, m.Time)
		return
	}
	// 自己的列表到了 ⇒ 已经回到自己的世界(离开时的 0x039d 万一没来,这里兜底)
	cs.visitOwner, cs.visitFlowers = 0, nil
	if p.st.For(acc).ReplaceFlowers(rows) == nil {
		p.pushFlowers(m.Session, acc, m.Time)
	}
}

// onFlowerBattleInfo 收下单朵花的战斗信息(0x0338)——**炫彩检测的唯一来源**。
//
// 同一条消息也用于世界首领/传说精灵的挑战面板,故先按 NPC 表确认是花种。
// 判炫彩只看 battle_npc_glass_info.glass_type:同级的 randed_battle_npc_glass 不是炫彩标志
// (非炫彩的命定花种同样是 true),三份 pcap 实测,详见 docs/map.md 8。
func (p *Pipeline) onFlowerBattleInfo(m capture.Message, acc string) {
	b, ok := scene.ParseFlowerBattle(m.AppBody)
	if !ok {
		return
	}
	if _, ok := p.db.FlowerNpc(b.CfgID); !ok {
		return
	}
	row := store.FlowerRow{
		ObjID: b.ObjID, CfgID: b.CfgID, Star: b.Star, PetBase: b.PetBase,
		ContentID: b.ContentID, SpecID: b.SpecID, EndTS: b.EndTS,
		State: store.FlowerPlain,
	}
	if b.GlassType != gamedata.GlassNull {
		row.State = store.FlowerGlassy
		row.Glass = p.db.GlassDesc(b.GlassType, b.GlassValue)
		if row.Glass == "" { // 配置里查不到(新赛季款/新色号)时至少标出是炫彩
			row.Glass = "炫彩"
		}
	}
	cs := p.conn(m.Session)
	if b.Visiting { // 参观中点开的是好友那朵:只更新内存里那套,不碰自己的库
		if cs.visitOwner == 0 || cs.visitFlowers == nil {
			return
		}
		cs.visitFlowers[row.ObjID] = row
		p.pushFlowers(m.Session, acc, m.Time)
		return
	}
	if p.st.For(acc).SetFlowerDetected(row) == nil {
		p.pushFlowers(m.Session, acc, m.Time)
	}
}

// onFlowerHarvested 在战斗结算后把那朵花打回未检测。
//
// **花本身不会消失**(用户实测:捕捉之后花还在原地、可以再挑战一次,再捉到的是另一只精灵),
// 所以这里不是撤行而是**重置检测**:里面那只精灵重投了,种族/等级/炫彩全部作废,
// 玩家下次点开它时才会有新的 0x0338 补上。花的位置、血脉、有效期属于花实体,照旧。
//
// 结算里给的 npc_obj_id 正是花实体的对象 id(实测 0x132c 的 interact_npc_id 与
// monster_info.npc_obj_id 同为花的 obj_id),与野生宠撤标记复用同一份解析:它只给
// 被捕捉/击败的,逃跑/战败不在此列(那一战什么都没变,检测结果照留)。
func (p *Pipeline) onFlowerHarvested(conn, acc string, gone []uint64, now time.Time) {
	if cs := p.conns[conn]; cs != nil && cs.visitOwner != 0 {
		return // 参观中打的是好友那朵:那套只在内存里,等下一次列表刷新即可
	}
	sc := p.st.For(acc)
	known := map[uint64]store.FlowerRow{}
	for _, r := range sc.Flowers(now.Unix()) {
		known[r.ObjID] = r
	}
	changed := false
	for _, id := range gone {
		r, ok := known[id]
		// 不是花种,或这一行已经什么都不知道了(未检测且连种族都没有):无事可做。
		// **种族也要清**——它来自花种列表,精灵重投后同样是旧的,与检测过没检测过无关。
		if !ok || (r.State == store.FlowerUndetected && r.PetBase == 0) {
			continue
		}
		if sc.ResetFlowerDetection(id) == nil {
			changed = true
		}
	}
	if changed {
		p.pushFlowers(conn, acc, now)
	}
}

// pushFlowers 广播花种图层:参观别人世界时推那套内存里的,否则清掉过期行后推库里自己的。
func (p *Pipeline) pushFlowers(conn, acc string, now time.Time) {
	cs := p.conns[conn]
	if cs != nil && cs.visitOwner != 0 {
		rows := make([]store.FlowerRow, 0, len(cs.visitFlowers))
		for _, r := range cs.visitFlowers {
			rows = append(rows, r)
		}
		p.srv.SetVisitFlowers(acc, cs.visitOwner, rows)
	} else {
		p.srv.SetVisitFlowers(acc, 0, nil)
		p.st.For(acc).PurgeFlowers(now.Unix())
	}
	p.srv.PushFlowers(acc, now)
}

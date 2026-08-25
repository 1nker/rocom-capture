package scene

import (
	"bytes"

	"google.golang.org/protobuf/encoding/protowire"
)

// 稀兽花种(实时地图页的花种图层,见 docs/map.md 8)。
//
// 大世界同时开着二十来朵花种,每朵关着一只混血精灵,打赢可捕捉。两条消息各给一半信息:
//
//	0x0375 花种列表(客户端登录后自动请求一次,玩家开面板再请求)
//	  → 每朵花的位置(content_cfg_id 查候选点)、里面是哪只精灵、什么时候刷新;**不含炫彩**。
//	0x0338 单朵花的战斗信息(玩家在地图上点中某朵花 / 走到花前时请求)
//	  → 该朵花的炫彩(battle_npc_glass_info)与等级;一次只给一朵。
//
// 故「哪几朵是炫彩」只能逐朵攒:列表给全集,0x0338 给某一朵的检测结果。
// **判炫彩只能看 battle_npc_glass_info.glass_type**;同级的 randed_battle_npc_glass 不是炫彩标志,
// 三份 pcap 实测:炫彩的命定魔力猫 true、**非炫彩的命定火神也是 true**、非炫彩的普通铠甲虫 false
// ——第二例即反证,拿它当标志会把每一朵命定花种都判成炫彩。
const (
	OpQueryBossNpcInfoRsp    = 0x0375 // ZONE_SCENE_QUERY_BOSS_NPC_INFO_RSP,s2c:花种/世界首领/传说精灵列表
	OpTeamBattleInfoQueryRsp = 0x0338 // ZONE_SCENE_TEAM_BATTLE_INFO_QUERY_RSP,s2c:单个可挑战 NPC 的战斗信息
	OpPlayerVisitInfoNotify  = 0x039d // ZONE_SCENE_PLAYER_VISIT_INFO_SYNC_NOTIFY,s2c:正在参观谁的世界
)

// ParseVisitOwner 从 s2c ZoneScenePlayerVisitInfoSyncNotify(0x039d)取 online_visit_owner(1):
// 正在参观的世界主人 uin;0(字段缺省)= 回到自己的世界。
// 传送去好友世界时这条**先于**传送通知(0x015c)到达,故据它判「现在看到的花是谁的」。
// 实测(rocom-20260826-002049)进别人世界时下发两条,只有 first_enter_visiting(2)不同。
func ParseVisitOwner(body []byte) uint32 {
	var owner uint32
	scanFields(trimBody(body), func(num protowire.Number, typ protowire.Type, _ []byte, v uint64) {
		if num == 1 && typ == protowire.VarintType {
			owner = uint32(v)
		}
	})
	return owner
}

// FlowerSeed 是花种列表(0x0375 的 flower_npcs)里的一朵花。
// ObjID 是实体对象 id,花刷新(每日/每两周/被捕捉)后重新生成,故用它当「还是不是同一朵」的凭据;
// LogicID 反而不行——实测 npc_logic_id = content_cfg_id<<32 | 0xFFFE0001,是刷新行的纯函数,
// 换了一朵花只要落回同一个候选点就完全相同。
type FlowerSeed struct {
	ObjID     uint64 // npc_obj_id(7)
	CfgID     uint32 // npc_cfg_id(1),花种 NPC(属性 + 稀兽/命定,见 gamedata.FlowerNpc)
	Star      int32  // star(2),稀兽花种 5 星、命定花种 7 星
	PetBase   uint32 // battle_petbase_id(5),花里那只精灵的形态 id
	ContentID uint32 // content_cfg_id(8),刷新行 id → gamedata.FlowerSpot 查坐标
	EndTS     int64  // end_timestamp(10),本朵花的有效期终点(见 docs/map.md 8 的刷新规则)
	SpecID    uint32 // spec_flower_seed_id(11),命定花种才有(活动限定)
}

// FlowerBattle 是单朵花的战斗信息(0x0338 的 team_battle_info)里做炫彩检测所需的字段。
// 同一条消息也用于世界首领/传说精灵,故调用方须按 CfgID 确认是花种。
type FlowerBattle struct {
	ObjID      uint64 // npc_obj_id(7)
	CfgID      uint32 // npc_cfg_id(1)
	Star       int32  // star(2)
	PetBase    uint32 // battle_petbase_id(5)
	ContentID  uint32 // 由 npc_logic_id(6) 的高 32 位还原(本消息不直接给 content_cfg_id)
	EndTS      int64  // end_timestamp(27)
	SpecID     uint32 // spec_flower_seed_id(25)
	GlassType  int32  // battle_npc_glass_info(32).glass_type:0=非炫彩,1/2=炫彩
	GlassValue int32  // battle_npc_glass_info(32).glass_value:是哪一种炫彩(gamedata.GlassDesc)
	Visiting   bool   // 讲的是别人世界里那朵花(带 visit_flower_seed_boss_datas,字段 35)
}

// flowerLogicShift:npc_logic_id 的高 32 位即刷新行 id(content_cfg_id),低 32 位是常量。
const flowerLogicShift = 32

// visitFlowerField 是 BossNpcInfo(23)/TeamBattleInfo(35)里的 visit_flower_seed_boss_datas。
// **它出现即说明这份回包讲的是别人世界里的花种**:玩家传送去好友的世界后打开花种面板,客户端
// 会带上 friend_uin 再问一次,回包给的是**那个世界**的花(obj_id 与自己世界的完全不同,里面的
// 精灵也不是自己的),另附一份 visit_flower_seed_boss_datas 说明访客自己的种子。
// 实测(rocom-20260826-002049):自己的两次列表 23 条**一条都没有**这个字段,好友世界那次 23 条
// **条条都有**,故据此二分足够干净;拿它当判据还不必去解 c2s 请求里的 friend_uin
// (c2s 有 6 字节子头,单字段小消息的起点难以可靠锚定)。
// 解析出来只做**标记**:这些花照样有用(参观时画在地图上提醒好友去捉),只是不能记进自己的库。
const visitFlowerField = 23

// trimBody 去掉 tsf4g 尾之后的内容(s2c 消息 protobuf 从 0 起,尾部另有 trailer,
// 靠 ScanFields 遇到非法字段自行停下)。
func trimBody(body []byte) []byte {
	if head, _, ok := bytes.Cut(body, tsf4gMark); ok {
		return head
	}
	return body
}

// FlowerList 是一次花种列表回包。Visiting 表示这份讲的是**别人世界**里的花
// (带 visit_flower_seed_boss_datas,见 visitFlowerField):它们不能记进自己的库,
// 但参观期间照样画在地图上(见 docs/map.md 8)。
type FlowerList struct {
	Seeds    []FlowerSeed
	Visiting bool
}

// ParseFlowerList 从 s2c ZoneSceneQueryBossNpcInfoRsp(0x0375)取花种列表:
// flower_npcs(2,BossNpcInfos) → boss_npcs(1,重复 BossNpcInfo)。
// 只取 flower_npcs 那一支——world_leader_npcs(3)/legendary_npcs(4) 是世界首领与传说精灵,不是花种。
// ok=false 表示这份回包没法用:ret_code 非 0(服务器拒绝时这一支缺省,拿空列表会把库里的花全删掉)
// 或根本没有花种那一支。
func ParseFlowerList(body []byte) (FlowerList, bool) {
	var out FlowerList
	body = trimBody(body)
	if code, ok := retCode(body); ok && code != 0 {
		return out, false
	}
	infos := subMsg(body, 2)
	if infos == nil {
		return out, false
	}
	scanFields(infos, func(num protowire.Number, typ protowire.Type, val []byte, _ uint64) {
		if num != 1 || typ != protowire.BytesType { // boss_npcs
			return
		}
		var f FlowerSeed
		scanFields(val, func(n protowire.Number, t protowire.Type, _ []byte, v uint64) {
			if n == visitFlowerField && t == protowire.BytesType {
				out.Visiting = true
				return
			}
			if t != protowire.VarintType {
				return
			}
			switch n {
			case 1:
				f.CfgID = uint32(v)
			case 2:
				f.Star = int32(v)
			case 5:
				f.PetBase = uint32(v)
			case 7:
				f.ObjID = v
			case 8:
				f.ContentID = uint32(v)
			case 10:
				f.EndTS = int64(v)
			case 11:
				f.SpecID = uint32(v)
			}
		})
		if f.ObjID != 0 && f.CfgID != 0 {
			out.Seeds = append(out.Seeds, f)
		}
	})
	return out, true
}

// ParseFlowerBattle 从 s2c ZoneSceneTeamBattleInfoQueryRsp(0x0338)取单个可挑战 NPC 的战斗信息:
// team_battle_info(2,TeamBattleInfo)。ret_code 非 0 或没有 team_battle_info 时 ok=false;
// 讲的是别人世界里那朵花时(带 visit_flower_seed_boss_datas,字段号 35)置 Visiting。
func ParseFlowerBattle(body []byte) (FlowerBattle, bool) {
	var b FlowerBattle
	body = trimBody(body)
	if code, ok := retCode(body); ok && code != 0 {
		return b, false
	}
	info := subMsg(body, 2)
	if info == nil {
		return b, false
	}
	scanFields(info, func(num protowire.Number, typ protowire.Type, val []byte, v uint64) {
		if num == 35 && typ == protowire.BytesType { // visit_flower_seed_boss_datas:别人的世界
			b.Visiting = true
			return
		}
		if num == 32 && typ == protowire.BytesType { // battle_npc_glass_info(GlassInfo)
			scanFields(val, func(n protowire.Number, t protowire.Type, _ []byte, gv uint64) {
				if t != protowire.VarintType {
					return
				}
				switch n {
				case 1:
					b.GlassType = int32(gv)
				case 2:
					b.GlassValue = int32(gv)
				}
			})
			return
		}
		if typ != protowire.VarintType {
			return
		}
		switch num {
		case 1:
			b.CfgID = uint32(v)
		case 2:
			b.Star = int32(v)
		case 5:
			b.PetBase = uint32(v)
		case 6:
			b.ContentID = uint32(v >> flowerLogicShift)
		case 7:
			b.ObjID = v
		case 25:
			b.SpecID = uint32(v)
		case 27:
			b.EndTS = int64(v)
		}
	})
	return b, b.ObjID != 0 && b.CfgID != 0
}

// retCode 取 ret_info(1,RetInfo) → ret_code(1);无 ret_info 时 ok=false。
func retCode(body []byte) (int32, bool) {
	ret := subMsg(body, 1)
	if ret == nil {
		return 0, false
	}
	var code int32
	scanFields(ret, func(num protowire.Number, typ protowire.Type, _ []byte, v uint64) {
		if num == 1 && typ == protowire.VarintType {
			code = int32(v)
		}
	})
	return code, true
}

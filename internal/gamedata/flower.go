package gamedata

// 稀兽花种(实时地图页的花种图层,见 docs/map.md 8)。
//
// 花种不是固定点位:每朵花当前长在哪儿只有流量里的花种列表(0x0375)知道,列表给的是
// **刷新行 id**(content_cfg_id)。这里两张表把它换成能画在地图上的东西:
//   FlowerSpots: 刷新行 id -> 候选点世界坐标(全部候选点静态入库,1581 行)
//   flowerNpcs:  花种 NPC_CONF id -> 类别(稀兽/命定)与血脉花图
// 生成见 scripts/gen_gamedata.py 的「稀兽花种」段,图标见 scripts/gen_icons.py 的 flower 组。

// FlowerNpc 是一种花种 NPC:类别中文名与大地图图标。
// 同类别下按**血脉**分 18 个 NPC id(普通/草/火/水/…/幻),图标即该血脉的花图。
// 是血脉不是种族属性:花里孕育的是混血精灵(火血脉的花里可以是虫系的铠甲虫)。
type FlowerNpc struct {
	Name string `json:"n"`    // 稀兽花种 / 命定花种
	Icon string `json:"icon"` // 血脉花图的原始文件名;Go 侧拼 flower/<原名>.webp
}

// FlowerSpot 是一个花种候选点的世界坐标(厘米)。
type FlowerSpot struct {
	Res int32
	X   int32
	Y   int32
}

// FlowerNpc 返回某花种 NPC 的类别与图标;不是花种则 ok=false。
func (db *DB) FlowerNpc(cfgID uint32) (FlowerNpc, bool) {
	f, ok := db.flowerNpcs[key(cfgID)]
	return f, ok
}

// FlowerIcon 返回某花种 NPC 的图标相对路径(flower/<原名>.webp);无图标时为空串。
func (db *DB) FlowerIcon(cfgID uint32) string {
	f, ok := db.flowerNpcs[key(cfgID)]
	if !ok || f.Icon == "" {
		return ""
	}
	return "flower/" + f.Icon + ".webp"
}

// FlowerSpot 返回某刷新行(花种列表里的 content_cfg_id)的候选点坐标;未收录返回 ok=false。
func (db *DB) FlowerSpot(contentID uint32) (FlowerSpot, bool) {
	p, ok := db.flowerSpots[contentID]
	return p, ok
}

// FlowerLevel 返回花里那只精灵的等级。等级**不在任何下发字段里**,是按 (星级, 限定花种 id)
// 查表算出来的——复刻客户端 MagicManualUtils.GetFlowerLevel:命定花种(spec_flower_seed_id 非 0)
// 查 ACTIVITY_SPEC_FLOWER_SEED_CONF 的 activity_team_battle_star_level[星级],普通花种查
// PET_GLOBAL_CONFIG 的 team_battle_star_level_glass_<星级>。两个入参都在花种列表(0x0375)里,
// 故**列表一到就有等级**,不必等玩家点开某朵花(实测 5 星普通 55、7 星命定 60)。查不到返回 0。
func (db *DB) FlowerLevel(star int32, specID uint32) int32 {
	if star <= 0 {
		return 0
	}
	if specID != 0 {
		lv := db.flowerSpecLv[key(specID)]
		if int(star) <= len(lv) {
			return lv[star-1] // 配置是按星级 1..N 排的数组
		}
		return 0
	}
	return db.flowerStarLv[key(uint32(star))]
}

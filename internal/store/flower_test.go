package store

import (
	"path/filepath"
	"testing"
)

// 花种列表每次都是全量:列表里没有的行必须删掉(整朵重投后 obj_id 会全换,旧行留着就是幽灵点),
// 列表里有的行只覆盖描述列——已检测出的炫彩结果不能被一次列表刷新抹掉。
func TestReplaceFlowers(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer st.db.Close()
	sc := st.For("UID:1")

	old := []FlowerRow{
		{ObjID: 9279722995167894922, CfgID: 700002, Star: 7, PetBase: 3007, ContentID: 2606405, EndTS: 1787860799},
		{ObjID: 9279722995167901301, CfgID: 20143, Star: 5, PetBase: 3439, ContentID: 140309, EndTS: 1787688000},
	}
	if err := sc.ReplaceFlowers(old); err != nil {
		t.Fatal(err)
	}
	// 检测出一朵炫彩
	det := old[0]
	det.State, det.Glass = FlowerGlassy, "四角星·亮X暗 - 浅绿青"
	if err := sc.SetFlowerDetected(det); err != nil {
		t.Fatal(err)
	}
	// 同一批再来一次:检测结果必须保住
	if err := sc.ReplaceFlowers(old); err != nil {
		t.Fatal(err)
	}
	got := sc.Flowers(0)
	if len(got) != 2 {
		t.Fatalf("重复下发同一列表后行数 = %d, 期望 2", len(got))
	}
	for _, r := range got {
		if r.ObjID == det.ObjID && (r.State != FlowerGlassy || r.Glass == "") {
			t.Errorf("列表刷新不该抹掉已检测出的炫彩: %+v", r)
		}
	}

	// 整朵重投:obj_id 全换,旧行必须消失(实测 2026-08-26 零点前后 23 朵 obj_id 全变)
	fresh := []FlowerRow{
		{ObjID: 9281973836197157848, CfgID: 700002, Star: 7, PetBase: 3007, ContentID: 2606405, EndTS: 1787860799},
		{ObjID: 9281973836197161409, CfgID: 20143, Star: 5, PetBase: 3335, ContentID: 140309, EndTS: 1787688000},
	}
	if err := sc.ReplaceFlowers(fresh); err != nil {
		t.Fatal(err)
	}
	got = sc.Flowers(0)
	if len(got) != 2 {
		t.Fatalf("换代后行数 = %d, 期望 2(旧的两行应被删掉)", len(got))
	}
	for _, r := range got {
		if r.ObjID != fresh[0].ObjID && r.ObjID != fresh[1].ObjID {
			t.Errorf("旧行没被删掉: %+v", r)
		}
		if r.State != FlowerUndetected {
			t.Errorf("新一朵花应是未检测: %+v", r)
		}
	}
}

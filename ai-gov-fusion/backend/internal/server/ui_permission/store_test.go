package ui_permission

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestStore_RequiredActionValidation 验证 required_action 外键校验。
func TestStore_RequiredActionValidation(t *testing.T) {
	f := setupProjectorFixtures(t)

	// 尝试创建路由时引用不存在的 action_id
	nonExistentID := int64(99999)
	_, err := CreateRoute(f.db, CreateRouteRequest{
		RoutePath:        "/fake",
		MenuID:           &f.menuDashboard,
		RequiredActionID: &nonExistentID,
	})
	if err == nil {
		t.Error("引用不存在的 action_id 应返回错误")
	}

	// 尝试创建按钮绑定时引用不存在的 action_id
	_, err = CreateActionBinding(f.db, CreateActionBindingRequest{
		ButtonCode:       "btn-fake",
		ButtonLabel:      "假按钮",
		PageRoute:        "/fake",
		RequiredActionID: &nonExistentID,
	})
	if err == nil {
		t.Error("引用不存在的 action_id 应返回错误")
	}
}

// TestStore_CRUD 验证菜单、路由、按钮绑定的基本 CRUD 操作。
func TestStore_CRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	if err := db.AutoMigrate(&SysUIMenu{}, &SysUIRoute{}, &SysUIActionBinding{}, &sysActionCatalog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	// ── 菜单 CRUD ──
	m, err := CreateMenu(db, CreateMenuRequest{MenuCode: "test", Label: "测试", SortOrder: 5})
	if err != nil {
		t.Fatalf("CreateMenu 失败: %v", err)
	}
	if m.ID == 0 {
		t.Error("CreateMenu 应返回非零 ID")
	}

	got, err := GetMenu(db, m.ID)
	if err != nil {
		t.Fatalf("GetMenu 失败: %v", err)
	}
	if got.Label != "测试" {
		t.Errorf("期望 label=测试, 实际=%s", got.Label)
	}

	updated, err := UpdateMenu(db, m.ID, CreateMenuRequest{Label: "已更新", SortOrder: 10})
	if err != nil {
		t.Fatalf("UpdateMenu 失败: %v", err)
	}
	if updated.Label != "已更新" || updated.SortOrder != 10 {
		t.Error("UpdateMenu 未正确更新字段")
	}

	if err := DeleteMenu(db, m.ID); err != nil {
		t.Fatalf("DeleteMenu 失败: %v", err)
	}
	if _, err := GetMenu(db, m.ID); err == nil {
		t.Error("DeleteMenu 后 GetMenu 应返回错误")
	}

	// ── 路由 CRUD（需先建 action） ──
	db.Create(&sysActionCatalog{ID: 1, ActionCode: "test.read"})
	m2, _ := CreateMenu(db, CreateMenuRequest{MenuCode: "m2", Label: "菜单2"})

	r, err := CreateRoute(db, CreateRouteRequest{
		RoutePath:        "/test",
		MenuID:           &m2.ID,
		RequiredActionID: ptr(int64(1)),
	})
	if err != nil {
		t.Fatalf("CreateRoute 失败: %v", err)
	}
	if r.ID == 0 {
		t.Error("CreateRoute 应返回非零 ID")
	}

	routes, err := FindRoutesByMenu(db, m2.ID)
	if err != nil {
		t.Fatalf("FindRoutesByMenu 失败: %v", err)
	}
	if len(routes) != 1 {
		t.Errorf("期望 1 个路由，实际 %d", len(routes))
	}

	// ── 按钮绑定 CRUD ──
	b, err := CreateActionBinding(db, CreateActionBindingRequest{
		ButtonCode:       "btn-test",
		ButtonLabel:      "测试按钮",
		PageRoute:        "/test",
		RequiredActionID: ptr(int64(1)),
	})
	if err != nil {
		t.Fatalf("CreateActionBinding 失败: %v", err)
	}
	if b.ID == 0 {
		t.Error("CreateActionBinding 应返回非零 ID")
	}

	bindings, err := ListActionBindingsByPage(db, "/test")
	if err != nil {
		t.Fatalf("ListActionBindingsByPage 失败: %v", err)
	}
	if len(bindings) != 1 {
		t.Errorf("期望 1 个按钮绑定，实际 %d", len(bindings))
	}
}

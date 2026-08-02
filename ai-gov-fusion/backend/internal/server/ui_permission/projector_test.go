package ui_permission

import (
	"context"
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ── Mock ABAC 引擎 ─────────────────────────────────────────────────────────

// mockABAC 可配置的 ABAC 引擎——允许的动作集合在测试中注入。
// Evaluate 对 allowedActions 中的 action 返回 nil（通过），其余返回错误（拒绝）。
type mockABAC struct {
	allowedActions map[string]bool
}

func (m *mockABAC) Evaluate(_ context.Context, _ Subject, action string, _ Resource) error {
	if m.allowedActions[action] {
		return nil
	}
	return fmt.Errorf("拒绝: %s", action)
}

// 确保 mockABAC 实现了 ABACEngine 接口。
var _ ABACEngine = (*mockABAC)(nil)

// ── 共享测试夹具 ───────────────────────────────────────────────────────────

// projectorFixtures 在内存 SQLite 中建立菜单、路由、按钮绑定和操作目录的测试数据。
// 供 projector_test.go 和 store_test.go 共用。
type projectorFixtures struct {
	db        *gorm.DB
	projector *Projector
	// 预插入的测试记录 ID
	menuDashboard int64 // 仪表盘
	menuFinance   int64 // 资金管理
	menuAdmin     int64 // 系统管理
	menuModels    int64 // 模型管理
	actDashRead   string // dashboard.read
	actFundAlloc  string // fund.allocate
	actFundLedger string // fund.ledger.read
	actIamWrite   string // iam.settings.write
}

func setupProjectorFixtures(t *testing.T) *projectorFixtures {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}

	if err := db.AutoMigrate(&SysUIMenu{}, &SysUIRoute{}, &SysUIActionBinding{}, &sysActionCatalog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	f := &projectorFixtures{db: db}

	// ── 操作目录（sys_action_catalogs） ──
	acts := []struct {
		id         string
		actionCode string
	}{
		{"101", "dashboard.read"},
		{"201", "fund.allocate"},
		{"202", "fund.ledger.read"},
		{"301", "iam.settings.write"},
	}
	for _, a := range acts {
		db.Create(&sysActionCatalog{ID: a.id, ActionCode: a.actionCode})
	}
	f.actDashRead = "101"
	f.actFundAlloc = "201"
	f.actFundLedger = "202"
	f.actIamWrite = "301"

	// ── 菜单 ──
	menus := []CreateMenuRequest{
		{MenuCode: "dashboard", Label: "仪表盘", Icon: "dashboard", SortOrder: 1},
		{MenuCode: "finance", Label: "资金管理", Icon: "wallet", SortOrder: 2},
		{MenuCode: "admin", Label: "系统管理", Icon: "settings", SortOrder: 3},
		{MenuCode: "models", Label: "模型管理", Icon: "cube", SortOrder: 4},
	}
	for i, req := range menus {
		m, _ := CreateMenu(db, req)
		switch i {
		case 0:
			f.menuDashboard = m.ID
		case 1:
			f.menuFinance = m.ID
		case 2:
			f.menuAdmin = m.ID
		case 3:
			f.menuModels = m.ID
		}
	}

	// ── 路由 ──
	routes := []struct {
		path     string
		menuID   int64
		actionID *string
	}{
		{"/dashboard", f.menuDashboard, &f.actDashRead},
		{"/finance/allocate", f.menuFinance, &f.actFundAlloc},
		{"/finance/ledger", f.menuFinance, &f.actFundLedger},
		{"/admin/settings", f.menuAdmin, &f.actIamWrite},
		{"/models", f.menuModels, nil},
	}
	for _, rt := range routes {
		CreateRoute(db, CreateRouteRequest{
			RoutePath:        rt.path,
			MenuID:           &rt.menuID,
			RequiredActionID: rt.actionID,
		})
	}

	// ── 按钮绑定 ──
	bindings := []struct {
		code     string
		label    string
		page     string
		actionID *string
	}{
		{"btn-allocate", "划拨", "/finance/allocate", &f.actFundAlloc},
		{"btn-export", "导出", "/finance/ledger", &f.actFundLedger},
		{"btn-view-models", "查看模型", "/models", nil},
	}
	for _, bd := range bindings {
		CreateActionBinding(db, CreateActionBindingRequest{
			ButtonCode:       bd.code,
			ButtonLabel:      bd.label,
			PageRoute:        bd.page,
			RequiredActionID: bd.actionID,
		})
	}

	return f
}

// ptr 返回 T 类型值的指针。
func ptr[T any](v T) *T { return &v }

// ── 投影测试 ───────────────────────────────────────────────────────────────

func TestProjectMenus_AdminSeeAll(t *testing.T) {
	f := setupProjectorFixtures(t)
	ctx := context.Background()

	f.projector = NewProjector(f.db, &mockABAC{
		allowedActions: map[string]bool{
			"dashboard.read":    true,
			"fund.allocate":     true,
			"fund.ledger.read":  true,
			"iam.settings.write": true,
		},
	})

	menus, err := f.projector.ProjectMenus(ctx, Subject{Type: "user", ID: "admin-001"})
	if err != nil {
		t.Fatalf("ProjectMenus 失败: %v", err)
	}

	if len(menus) != 4 {
		t.Errorf("admin 应看到 4 个菜单，实际: %d", len(menus))
	}
	for _, m := range menus {
		if !m.Visible {
			t.Errorf("admin 的菜单 %s 应为 visible=true", m.MenuCode)
		}
	}
}

func TestProjectMenus_NormalUser(t *testing.T) {
	f := setupProjectorFixtures(t)
	ctx := context.Background()

	f.projector = NewProjector(f.db, &mockABAC{
		allowedActions: map[string]bool{
			"dashboard.read":   true,
			"fund.ledger.read": true,
		},
	})

	menus, err := f.projector.ProjectMenus(ctx, Subject{Type: "user", ID: "user-001"})
	if err != nil {
		t.Fatalf("ProjectMenus 失败: %v", err)
	}

	visibleCodes := make(map[string]bool)
	for _, m := range menus {
		visibleCodes[m.MenuCode] = true
	}

	if !visibleCodes["dashboard"] {
		t.Error("普通用户应看到仪表盘菜单")
	}
	if !visibleCodes["finance"] {
		t.Error("普通用户应看到资金管理菜单（有 fund.ledger.read 权限）")
	}
	if !visibleCodes["models"] {
		t.Error("普通用户应看到模型管理菜单（公开路由）")
	}
	if visibleCodes["admin"] {
		t.Error("普通用户不应看到系统管理菜单（无 iam.settings.write 权限）")
	}
}

func TestProjectMenus_NoAccess(t *testing.T) {
	f := setupProjectorFixtures(t)
	ctx := context.Background()

	f.projector = NewProjector(f.db, &mockABAC{
		allowedActions: map[string]bool{"dashboard.read": true},
	})

	menus, err := f.projector.ProjectMenus(ctx, Subject{Type: "user", ID: "user-002"})
	if err != nil {
		t.Fatalf("ProjectMenus 失败: %v", err)
	}

	visibleCodes := make(map[string]bool)
	for _, m := range menus {
		visibleCodes[m.MenuCode] = true
	}

	if !visibleCodes["dashboard"] {
		t.Error("应看到仪表盘菜单")
	}
	if !visibleCodes["models"] {
		t.Error("应看到模型管理菜单（公开路由）")
	}
	if visibleCodes["finance"] {
		t.Error("无 fund 权限不应看到资金管理菜单")
	}
	if visibleCodes["admin"] {
		t.Error("无 iam 权限不应看到系统管理菜单")
	}
}

func TestProjectActions_ButtonVisibility(t *testing.T) {
	f := setupProjectorFixtures(t)
	ctx := context.Background()

	f.projector = NewProjector(f.db, &mockABAC{
		allowedActions: map[string]bool{"fund.ledger.read": true},
	})

	subject := Subject{Type: "user", ID: "user-001"}

	// /finance/allocate 页面——btn-allocate 需要 fund.allocate，用户无此权限
	actions, err := f.projector.ProjectActions(ctx, subject, "/finance/allocate")
	if err != nil {
		t.Fatalf("ProjectActions 失败: %v", err)
	}
	if actions["btn-allocate"] {
		t.Error("btn-allocate 应为 false——用户无 fund.allocate 权限")
	}

	// /finance/ledger 页面——btn-export 需要 fund.ledger.read，用户有
	actions2, err := f.projector.ProjectActions(ctx, subject, "/finance/ledger")
	if err != nil {
		t.Fatalf("ProjectActions(/finance/ledger) 失败: %v", err)
	}
	if !actions2["btn-export"] {
		t.Error("btn-export 应为 true——用户有 fund.ledger.read 权限")
	}

	// /models 页面——公开按钮
	actions3, err := f.projector.ProjectActions(ctx, subject, "/models")
	if err != nil {
		t.Fatalf("ProjectActions(/models) 失败: %v", err)
	}
	if !actions3["btn-view-models"] {
		t.Error("btn-view-models 应为 true——公开按钮无需权限")
	}

	// 完全无权限的用户
	f2 := setupProjectorFixtures(t)
	f2.projector = NewProjector(f2.db, &mockABAC{allowedActions: map[string]bool{}})
	actions4, _ := f2.projector.ProjectActions(ctx, subject, "/finance/allocate")
	if actions4["btn-allocate"] {
		t.Error("无权限用户不应看到 btn-allocate")
	}
}

func TestProjectRoutes_NormalUser(t *testing.T) {
	f := setupProjectorFixtures(t)
	ctx := context.Background()

	f.projector = NewProjector(f.db, &mockABAC{
		allowedActions: map[string]bool{
			"dashboard.read":   true,
			"fund.ledger.read": true,
		},
	})

	routes, err := f.projector.ProjectRoutes(ctx, Subject{Type: "user", ID: "user-001"})
	if err != nil {
		t.Fatalf("ProjectRoutes 失败: %v", err)
	}

	accessible := make(map[string]bool)
	for _, r := range routes {
		accessible[r.RoutePath] = true
	}

	if !accessible["/dashboard"] {
		t.Error("应可访问 /dashboard")
	}
	if !accessible["/finance/ledger"] {
		t.Error("应可访问 /finance/ledger")
	}
	if !accessible["/models"] {
		t.Error("应可访问 /models（公开路由）")
	}
	if accessible["/finance/allocate"] {
		t.Error("不应可访问 /finance/allocate（无 fund.allocate）")
	}
	if accessible["/admin/settings"] {
		t.Error("不应可访问 /admin/settings（无 iam.settings.write）")
	}
}

func TestProjectMenus_ContainerPropagation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	if err := db.AutoMigrate(&SysUIMenu{}, &SysUIRoute{}, &SysUIActionBinding{}, &sysActionCatalog{}); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}

	db.Create(&sysActionCatalog{ID: "1", ActionCode: "reports.read"})

	// 二级菜单: settings（容器，无路由） → reports（叶子，有路由）
	settings, _ := CreateMenu(db, CreateMenuRequest{MenuCode: "settings", Label: "设置"})
	reports, _ := CreateMenu(db, CreateMenuRequest{MenuCode: "reports", Label: "报表", ParentID: &settings.ID})
	CreateRoute(db, CreateRouteRequest{
		RoutePath:        "/settings/reports",
		MenuID:           &reports.ID,
		RequiredActionID: ptr("1"),
	})

	ctx := context.Background()

	// 用户有 reports.read → 容器菜单也应可见
	proj := NewProjector(db, &mockABAC{allowedActions: map[string]bool{"reports.read": true}})
	menus, err := proj.ProjectMenus(ctx, Subject{Type: "user", ID: "u1"})
	if err != nil {
		t.Fatalf("ProjectMenus 失败: %v", err)
	}
	if len(menus) != 1 || menus[0].MenuCode != "settings" {
		t.Fatal("应返回 settings 菜单（子菜单可见，向上传播）")
	}
	if !menus[0].Visible || len(menus[0].Children) != 1 || menus[0].Children[0].MenuCode != "reports" {
		t.Error("settings 应 visible=true 且包含 reports 子菜单")
	}

	// 用户无任何权限 → 返回空
	proj2 := NewProjector(db, &mockABAC{allowedActions: map[string]bool{}})
	menus2, err := proj2.ProjectMenus(ctx, Subject{Type: "user", ID: "u2"})
	if err != nil {
		t.Fatalf("ProjectMenus 失败: %v", err)
	}
	if len(menus2) != 0 {
		t.Errorf("无权限时所有菜单应被过滤，实际返回 %d 个", len(menus2))
	}
}

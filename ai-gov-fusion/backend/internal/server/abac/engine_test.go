package abac

import (
	"context"
	"errors"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB 打开内存 SQLite 数据库并执行 ABAC 表迁移。
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开 SQLite 失败: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("迁移失败: %v", err)
	}
	return db
}

// newTestEngine 创建测试用的引擎实例。
func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	return NewEngine(newTestDB(t))
}

// registerAction 注册测试操作用于引擎评估。
func registerAction(t *testing.T, db *gorm.DB, code, name, axis, resourceType string) *SysActionCatalog {
	t.Helper()
	a := &SysActionCatalog{
		ID:           NewID(),
		ActionCode:   code,
		ActionName:   name,
		Axis:         axis,
		ResourceType: resourceType,
	}
	if err := db.Create(a).Error; err != nil {
		t.Fatalf("注册操作 %s 失败: %v", code, err)
	}
	return a
}

// ── TestEvaluate_Allow ──────────────────────────────────────────────────

// TestEvaluate_Allow 验证匹配 allow 策略时放行。
func TestEvaluate_Allow(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 注册操作。
	registerAction(t, db, "data.usage.read", "读取用量", AxisData, "usage")

	// 创建 allow 策略。
	p := &SysAccessPolicy{
		PolicyCode:     "P-ALLOW-DATA-READ",
		PolicyName:     "允许数据读取",
		Effect:         EffectAllow,
		ConditionsJSON: `{"axis":"data","resource_type":"usage"}`,
		Priority:       10,
	}
	if err := CreatePolicy(ctx, db, p); err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}

	// 绑定策略到主体。
	if err := BindPolicy(ctx, db, p.ID, SubjectTypeUser, "user-001"); err != nil {
		t.Fatalf("绑定策略失败: %v", err)
	}

	// 评估 —— 应放行。
	engine := NewEngine(db)
	subject := Subject{Type: SubjectTypeUser, ID: "user-001"}
	err := engine.Evaluate(ctx, subject, "data.usage.read", Resource{Type: "usage"})
	if err != nil {
		t.Errorf("期望允许，实际拒绝: %v", err)
	}
}

// ── TestEvaluate_Deny ──────────────────────────────────────────────────

// TestEvaluate_Deny 验证 deny 策略优先于 allow 策略。
// 当同一主体既有 allow 策略又有 deny 策略时，deny 必须胜出。
func TestEvaluate_Deny(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 注册操作。
	registerAction(t, db, "fund.allocate", "划拨", AxisFund, "account")

	// 创建 allow 策略（优先级低）。
	allowP := &SysAccessPolicy{
		PolicyCode:     "P-ALLOW-FUND",
		PolicyName:     "允许资金操作",
		Effect:         EffectAllow,
		ConditionsJSON: `{"axis":"fund"}`,
		Priority:       5,
	}
	if err := CreatePolicy(ctx, db, allowP); err != nil {
		t.Fatalf("创建 allow 策略失败: %v", err)
	}

	// 创建 deny 策略（优先级更高）。
	denyP := &SysAccessPolicy{
		PolicyCode:     "P-DENY-FUND-ALLOCATE",
		PolicyName:     "禁止划拨",
		Effect:         EffectDeny,
		ConditionsJSON: `{"axis":"fund"}`,
		Priority:       100,
	}
	if err := CreatePolicy(ctx, db, denyP); err != nil {
		t.Fatalf("创建 deny 策略失败: %v", err)
	}

	// 两条策略都绑定到同一主体。
	if err := BindPolicy(ctx, db, allowP.ID, SubjectTypeUser, "user-001"); err != nil {
		t.Fatalf("绑定 allow 策略失败: %v", err)
	}
	if err := BindPolicy(ctx, db, denyP.ID, SubjectTypeUser, "user-001"); err != nil {
		t.Fatalf("绑定 deny 策略失败: %v", err)
	}

	// 评估 —— deny 应胜出。
	engine := NewEngine(db)
	subject := Subject{Type: SubjectTypeUser, ID: "user-001"}
	err := engine.Evaluate(ctx, subject, "fund.allocate", Resource{Type: "account"})
	if err == nil {
		t.Error("期望拒绝（deny 应优先于 allow），实际允许")
	}
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("期望 ErrAccessDenied，实际: %v", err)
	}
}

// ── TestEvaluate_DefaultDeny ────────────────────────────────────────────

// TestEvaluate_DefaultDeny 验证无策略、无角色权限时默认拒绝。
func TestEvaluate_DefaultDeny(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 注册操作但不创建任何策略/角色。
	registerAction(t, db, "fund.allocate", "划拨", AxisFund, "account")

	// 评估 —— 默认拒绝。
	engine := NewEngine(db)
	subject := Subject{Type: SubjectTypeUser, ID: "user-001"}
	err := engine.Evaluate(ctx, subject, "fund.allocate", Resource{Type: "account"})
	if err == nil {
		t.Error("期望默认拒绝，实际允许")
	}
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("期望 ErrAccessDenied，实际: %v", err)
	}
}

// ── TestEvaluate_RoleBased ─────────────────────────────────────────────

// TestEvaluate_RoleBased 验证通过角色绑定获得权限。
func TestEvaluate_RoleBased(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 注册操作。
	action := registerAction(t, db, "data.usage.read", "读取用量", AxisData, "usage")

	// 创建角色。
	role := &SysRole{
		RoleCode: "data_viewer",
		RoleName: "数据查看者",
	}
	if err := CreateRole(ctx, db, role); err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}

	// 授予角色权限。
	if err := GrantPermission(ctx, db, role.ID, []string{action.ID}); err != nil {
		t.Fatalf("授予权限失败: %v", err)
	}

	// 将角色分配给主体。
	if err := AssignRole(ctx, db, SubjectTypeUser, "user-001", role.ID, nil, nil, nil); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}

	// 评估 —— 通过角色权限应放行。
	engine := NewEngine(db)
	subject := Subject{Type: SubjectTypeUser, ID: "user-001"}
	err := engine.Evaluate(ctx, subject, "data.usage.read", Resource{Type: "usage"})
	if err != nil {
		t.Errorf("期望通过角色权限允许，实际拒绝: %v", err)
	}
}

// ── TestEvaluate_SeparationOfDuty ──────────────────────────────────────

// TestEvaluate_SeparationOfDuty 验证职责分离——拥有 fund 轴权限的主体
// 被禁止执行 routing 轴操作。
func TestEvaluate_SeparationOfDuty(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 注册 fund 轴操作和 routing 轴操作。
	fundAction := registerAction(t, db, "fund.allocate", "划拨", AxisFund, "account")
	registerAction(t, db, "routing.price.write", "修改价目", AxisRouting, "price")

	// 创建 fund 管理员角色。
	fundAdminRole := &SysRole{
		RoleCode: "fund_admin",
		RoleName: "资金管理员",
	}
	if err := CreateRole(ctx, db, fundAdminRole); err != nil {
		t.Fatalf("创建 fund 管理员角色失败: %v", err)
	}

	// 授予 fund 操作权限。
	if err := GrantPermission(ctx, db, fundAdminRole.ID, []string{fundAction.ID}); err != nil {
		t.Fatalf("授予 fund 权限失败: %v", err)
	}

	// 将角色分配给主体。
	if err := AssignRole(ctx, db, SubjectTypeUser, "user-001", fundAdminRole.ID, nil, nil, nil); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}

	// 创建职责分离 deny 策略：fund_admin 不能操作 routing。
	sodPolicy := &SysAccessPolicy{
		PolicyCode:     "P-SOD-FUND",
		PolicyName:     "职责分离：资金管理员不可操作路由",
		Effect:         EffectDeny,
		ConditionsJSON: `{"axis":"routing"}`,
		Priority:       1000,
		IsSystem:       true,
	}
	if err := CreatePolicy(ctx, db, sodPolicy); err != nil {
		t.Fatalf("创建职责分离策略失败: %v", err)
	}

	// 将 SOD 策略绑定到 fund_admin 角色。
	if err := BindPolicy(ctx, db, sodPolicy.ID, SubjectTypeRole, fundAdminRole.ID); err != nil {
		t.Fatalf("绑定 SOD 策略失败: %v", err)
	}

	engine := NewEngine(db)
	subject := Subject{Type: SubjectTypeUser, ID: "user-001"}

	// 场景 1：fund.allocate 应放行（fund 角色有权限，且 SOD 策略不匹配）。
	err := engine.Evaluate(ctx, subject, "fund.allocate", Resource{Type: "account"})
	if err != nil {
		t.Errorf("期望 fund.allocate 允许，实际拒绝: %v", err)
	}

	// 场景 2：routing.price.write 应拒绝（SOD 策略阻止 fund 角色操作 routing）。
	err = engine.Evaluate(ctx, subject, "routing.price.write", Resource{Type: "price"})
	if err == nil {
		t.Error("期望 routing.price.write 被拒绝（职责分离），实际允许")
	}
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("期望 ErrAccessDenied，实际: %v", err)
	}
}

// ── TestGetPermissions ──────────────────────────────────────────────────

// TestGetPermissions 验证返回主体在指定资源类型上的全部允许动作列表。
func TestGetPermissions(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 注册多个操作。
	readAction := registerAction(t, db, "data.usage.read", "读取用量", AxisData, "usage")
	invokeAction := registerAction(t, db, "data.model.invoke", "调用模型", AxisData, "model")
	registerAction(t, db, "fund.allocate", "划拨", AxisFund, "account")

	// 创建角色。
	role := &SysRole{
		RoleCode: "data_user",
		RoleName: "数据使用者",
	}
	if err := CreateRole(ctx, db, role); err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}

	// 授予 data 相关的两个操作权限。
	if err := GrantPermission(ctx, db, role.ID, []string{readAction.ID, invokeAction.ID}); err != nil {
		t.Fatalf("授予权限失败: %v", err)
	}

	// 将角色分配给主体。
	if err := AssignRole(ctx, db, SubjectTypeUser, "user-001", role.ID, nil, nil, nil); err != nil {
		t.Fatalf("分配角色失败: %v", err)
	}

	// 获取权限列表。
	engine := NewEngine(db)
	subject := Subject{Type: SubjectTypeUser, ID: "user-001"}
	actions, err := engine.GetPermissions(ctx, subject, "usage")
	if err != nil {
		t.Fatalf("获取权限失败: %v", err)
	}

	// 验证包含预期的操作。
	found := make(map[string]bool)
	for _, a := range actions {
		found[a] = true
	}
	if !found["data.usage.read"] {
		t.Error("期望权限列表包含 data.usage.read")
	}
	if !found["data.model.invoke"] {
		t.Error("期望权限列表包含 data.model.invoke")
	}
	if found["fund.allocate"] {
		t.Error("权限列表不应包含 fund.allocate（未授予该角色）")
	}
	if len(actions) < 2 {
		t.Errorf("期望至少 2 条权限，实际 %d 条: %v", len(actions), actions)
	}
}

// ── TestEvaluate_ActionNotFound ────────────────────────────────────────

// TestEvaluate_ActionNotFound 验证未注册的操作返回明确错误。
func TestEvaluate_ActionNotFound(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	engine := NewEngine(db)
	subject := Subject{Type: SubjectTypeUser, ID: "user-001"}
	err := engine.Evaluate(ctx, subject, "nonexistent.action", Resource{Type: "party"})
	if err == nil {
		t.Error("期望未注册操作返回错误")
	}
	if !errors.Is(err, ErrActionNotFound) {
		t.Errorf("期望 ErrActionNotFound，实际: %v", err)
	}
}

// ── TestEvaluatePolicy_Simulation ──────────────────────────────────────

// TestEvaluatePolicy_Simulation 验证策略模拟评估功能。
func TestEvaluatePolicy_Simulation(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 注册操作。
	registerAction(t, db, "data.usage.read", "读取用量", AxisData, "usage")

	// 创建 deny 策略。
	denyP := &SysAccessPolicy{
		PolicyCode:     "P-DENY-TEST",
		PolicyName:     "测试拒绝策略",
		Effect:         EffectDeny,
		ConditionsJSON: `{"axis":"data"}`,
		Priority:       50,
	}
	if err := CreatePolicy(ctx, db, denyP); err != nil {
		t.Fatalf("创建 deny 策略失败: %v", err)
	}

	// 绑定到主体。
	if err := BindPolicy(ctx, db, denyP.ID, SubjectTypeUser, "user-001"); err != nil {
		t.Fatalf("绑定策略失败: %v", err)
	}

	// 模拟评估。
	result, err := EvaluatePolicy(ctx, db, Subject{Type: SubjectTypeUser, ID: "user-001"}, "data.usage.read", Resource{Type: "usage"})
	if err != nil {
		t.Fatalf("模拟评估失败: %v", err)
	}

	if result.Allowed {
		t.Error("期望模拟评估结果为拒绝")
	}
	if len(result.MatchedDenyPolicies) != 1 {
		t.Errorf("期望 1 条匹配的 deny 策略，实际 %d 条", len(result.MatchedDenyPolicies))
	}
	if result.MatchedDenyPolicies[0].PolicyCode != "P-DENY-TEST" {
		t.Errorf("期望匹配 P-DENY-TEST，实际: %s", result.MatchedDenyPolicies[0].PolicyCode)
	}
}


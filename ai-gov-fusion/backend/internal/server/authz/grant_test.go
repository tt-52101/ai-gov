package authz

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupDB 创建内存 SQLite 数据库并执行 grants 表迁移。
// 每个测试使用独立数据库，保证测试隔离性。
func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开内存 SQLite 失败: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("迁移 grants 表失败: %v", err)
	}
	return db
}

// newAllowGrant 创建一条默认的 allow 授权记录，用于测试辅助。
func newAllowGrant(id, principalType, principalID, axis, action string) *Grant {
	return &Grant{
		ID:            id,
		PrincipalType: principalType,
		PrincipalID:   principalID,
		Axis:          axis,
		Action:        action,
		Effect:        EffectAllow,
	}
}

// newDenyGrant 创建一条默认的 deny 授权记录，用于测试辅助。
func newDenyGrant(id, principalType, principalID, axis, action string) *Grant {
	return &Grant{
		ID:            id,
		PrincipalType: principalType,
		PrincipalID:   principalID,
		Axis:          axis,
		Action:        action,
		Effect:        EffectDeny,
	}
}

// ── CreateGrant 测试 ──────────────────────────────────────────────────────

// TestCreateGrant_Success 正常创建四轴授权——插入一条 allow 授权记录后，
// 可通过 GetGrant 查询到该记录且字段一致。
func TestCreateGrant_Success(t *testing.T) {
	db := setupDB(t)
	g := newAllowGrant("g-001", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)

	if err := CreateGrant(db, g); err != nil {
		t.Fatalf("创建授权记录失败: %v", err)
	}

	got, err := GetGrant(db, "g-001")
	if err != nil {
		t.Fatalf("查询授权记录失败: %v", err)
	}
	if got.PrincipalType != PrincipalUser {
		t.Errorf("principal_type = %q, want %q", got.PrincipalType, PrincipalUser)
	}
	if got.PrincipalID != "user-1" {
		t.Errorf("principal_id = %q, want %q", got.PrincipalID, "user-1")
	}
	if got.Axis != AxisFund {
		t.Errorf("axis = %q, want %q", got.Axis, AxisFund)
	}
	if got.Action != ActionBalanceRead {
		t.Errorf("action = %q, want %q", got.Action, ActionBalanceRead)
	}
	if got.Effect != EffectAllow {
		t.Errorf("effect = %q, want %q", got.Effect, EffectAllow)
	}
}

// TestCreateGrant_NilGrant 传入 nil 授权记录应返回错误。
func TestCreateGrant_NilGrant(t *testing.T) {
	db := setupDB(t)
	if err := CreateGrant(db, nil); err == nil {
		t.Fatal("预期传入 nil 授权记录返回错误，但成功")
	}
}

// TestCreateGrant_MissingRequired 缺少必填字段应返回错误。
func TestCreateGrant_MissingRequired(t *testing.T) {
	db := setupDB(t)
	tests := []struct {
		name  string
		grant Grant
	}{
		{"缺少 principal_type", Grant{ID: "g-e1", PrincipalID: "u1", Axis: AxisFund, Action: ActionBalanceRead}},
		{"缺少 principal_id", Grant{ID: "g-e2", PrincipalType: PrincipalUser, Axis: AxisFund, Action: ActionBalanceRead}},
		{"缺少 axis", Grant{ID: "g-e3", PrincipalType: PrincipalUser, PrincipalID: "u1", Action: ActionBalanceRead}},
		{"缺少 action", Grant{ID: "g-e4", PrincipalType: PrincipalUser, PrincipalID: "u1", Axis: AxisFund}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := CreateGrant(db, &tt.grant); err == nil {
				t.Errorf("预期 %s 返回错误，但成功", tt.name)
			}
		})
	}
}

// TestCreateGrant_InvalidEffect 非法 effect 值应返回错误。
func TestCreateGrant_InvalidEffect(t *testing.T) {
	db := setupDB(t)
	g := &Grant{
		ID:            "g-badeffect",
		PrincipalType: PrincipalUser,
		PrincipalID:   "user-1",
		Axis:          AxisFund,
		Action:        ActionBalanceRead,
		Effect:        "unknown",
	}
	if err := CreateGrant(db, g); err == nil {
		t.Fatal("预期非法 effect 返回错误，但成功")
	}
}

// TestCreateGrant_DefaultEffect 未填 effect 时默认设为 allow。
func TestCreateGrant_DefaultEffect(t *testing.T) {
	db := setupDB(t)
	g := &Grant{
		ID:            "g-default",
		PrincipalType: PrincipalUser,
		PrincipalID:   "user-1",
		Axis:          AxisData,
		Action:        ActionUsageRead,
	}
	if err := CreateGrant(db, g); err != nil {
		t.Fatalf("创建授权记录失败: %v", err)
	}
	got, err := GetGrant(db, "g-default")
	if err != nil {
		t.Fatalf("查询授权记录失败: %v", err)
	}
	if got.Effect != EffectAllow {
		t.Errorf("effect = %q, want %q", got.Effect, EffectAllow)
	}
}

// ── Evaluate 测试 ─────────────────────────────────────────────────────────

// TestEvaluateGrant_Allow allow 放行 —— 存在匹配的 allow 记录时 Evaluate 返回 true。
func TestEvaluateGrant_Allow(t *testing.T) {
	db := setupDB(t)
	if err := CreateGrant(db, newAllowGrant("g-allow", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建 allow 授权失败: %v", err)
	}

	allowed, err := Evaluate(db, PrincipalUser, "user-1", AxisFund, ActionBalanceRead)
	if err != nil {
		t.Fatalf("评估失败: %v", err)
	}
	if !allowed {
		t.Fatal("存在匹配的 allow 授权，预期放行但被拒绝")
	}
}

// TestEvaluateGrant_Deny deny 优先 —— 同时存在 allow 和 deny 时，deny 必须优先返回 false。
func TestEvaluateGrant_Deny(t *testing.T) {
	db := setupDB(t)
	if err := CreateGrant(db, newAllowGrant("g-allow2", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建 allow 授权失败: %v", err)
	}
	if err := CreateGrant(db, newDenyGrant("g-deny1", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建 deny 授权失败: %v", err)
	}

	allowed, err := Evaluate(db, PrincipalUser, "user-1", AxisFund, ActionBalanceRead)
	if err != nil {
		t.Fatalf("评估失败: %v", err)
	}
	if allowed {
		t.Fatal("同时存在 allow 和 deny 授权，deny 应优先，预期拒绝但放行")
	}
}

// TestEvaluateGrant_NoGrant 无授权默认拒绝 —— 没有任何匹配的授权记录时 Evaluate 返回 false。
func TestEvaluateGrant_NoGrant(t *testing.T) {
	db := setupDB(t)

	allowed, err := Evaluate(db, PrincipalParty, "party-x", AxisIAM, ActionKeyCreate)
	if err != nil {
		t.Fatalf("评估失败: %v", err)
	}
	if allowed {
		t.Fatal("无匹配授权记录，默认拒绝，预期拒绝但放行")
	}
}

// TestEvaluateGrant_DifferentAxis 不同轴的授权不应影响其他轴。
func TestEvaluateGrant_DifferentAxis(t *testing.T) {
	db := setupDB(t)
	// 在 fund 轴创建 allow 授权。
	if err := CreateGrant(db, newAllowGrant("g-fund", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建 fund 轴授权失败: %v", err)
	}

	// 查询 iam 轴——不应放行。
	allowed, err := Evaluate(db, PrincipalUser, "user-1", AxisIAM, ActionKeyCreate)
	if err != nil {
		t.Fatalf("评估失败: %v", err)
	}
	if allowed {
		t.Fatal("fund 轴的授权不应影响 iam 轴，预期拒绝但放行")
	}
}

// ── ListGrants 测试 ───────────────────────────────────────────────────────

// TestListGrants_ByAxis 按轴筛选——仅返回指定 axis 的授权记录。
func TestListGrants_ByAxis(t *testing.T) {
	db := setupDB(t)
	// 创建 fund 轴授权。
	if err := CreateGrant(db, newAllowGrant("g-f1", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建 fund 授权失败: %v", err)
	}
	if err := CreateGrant(db, newAllowGrant("g-f2", PrincipalUser, "user-2", AxisFund, ActionLedgerRead)); err != nil {
		t.Fatalf("创建 fund 授权失败: %v", err)
	}
	// 创建 data 轴授权。
	if err := CreateGrant(db, newAllowGrant("g-d1", PrincipalUser, "user-1", AxisData, ActionUsageRead)); err != nil {
		t.Fatalf("创建 data 授权失败: %v", err)
	}

	// 按 fund 轴筛选。
	grants, err := ListGrants(db, "", AxisFund)
	if err != nil {
		t.Fatalf("按轴筛选失败: %v", err)
	}
	if len(grants) != 2 {
		t.Fatalf("fund 轴应有 2 条授权，实际 %d 条", len(grants))
	}
	for _, g := range grants {
		if g.Axis != AxisFund {
			t.Errorf("预期 axis = %q，但出现 %q", AxisFund, g.Axis)
		}
	}
}

// TestListGrants_ByPrincipal 按主体类型筛选——仅返回指定 principal_type 的记录。
func TestListGrants_ByPrincipal(t *testing.T) {
	db := setupDB(t)
	if err := CreateGrant(db, newAllowGrant("g-p1", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建 user 授权失败: %v", err)
	}
	if err := CreateGrant(db, newAllowGrant("g-p2", PrincipalParty, "party-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建 party 授权失败: %v", err)
	}

	grants, err := ListGrants(db, PrincipalUser, "")
	if err != nil {
		t.Fatalf("按主体类型筛选失败: %v", err)
	}
	if len(grants) != 1 {
		t.Fatalf("user 类型应有 1 条授权，实际 %d 条", len(grants))
	}
	if grants[0].PrincipalType != PrincipalUser {
		t.Errorf("principal_type = %q, want %q", grants[0].PrincipalType, PrincipalUser)
	}
}

// TestListGrants_All 不传任何筛选参数时返回所有授权。
func TestListGrants_All(t *testing.T) {
	db := setupDB(t)
	if err := CreateGrant(db, newAllowGrant("g-a1", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建授权失败: %v", err)
	}
	if err := CreateGrant(db, newAllowGrant("g-a2", PrincipalParty, "party-1", AxisData, ActionUsageRead)); err != nil {
		t.Fatalf("创建授权失败: %v", err)
	}

	grants, err := ListGrants(db, "", "")
	if err != nil {
		t.Fatalf("查询所有授权失败: %v", err)
	}
	if len(grants) != 2 {
		t.Fatalf("应有 2 条授权，实际 %d 条", len(grants))
	}
}

// ── DeleteGrant 测试 ──────────────────────────────────────────────────────

// TestDeleteGrant_Success 正常删除授权记录。
func TestDeleteGrant_Success(t *testing.T) {
	db := setupDB(t)
	if err := CreateGrant(db, newAllowGrant("g-del", PrincipalUser, "user-1", AxisFund, ActionBalanceRead)); err != nil {
		t.Fatalf("创建授权记录失败: %v", err)
	}

	if err := DeleteGrant(db, "g-del"); err != nil {
		t.Fatalf("删除授权记录失败: %v", err)
	}

	// 确认已删除。
	_, err := GetGrant(db, "g-del")
	if err == nil {
		t.Fatal("预期查询已删除的记录返回错误，但成功")
	}
}

// TestDeleteGrant_NotFound 删除不存在的记录应返回错误。
func TestDeleteGrant_NotFound(t *testing.T) {
	db := setupDB(t)
	if err := DeleteGrant(db, "nonexistent"); err == nil {
		t.Fatal("预期删除不存在的记录返回错误，但成功")
	}
}

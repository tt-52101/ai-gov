package abac

import (
	"context"
	"errors"
	"testing"
)

// ── 策略 CRUD 测试 ─────────────────────────────────────────────────────

// TestPolicyCRUD 验证策略的基本 CRUD 操作。
func TestPolicyCRUD(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 创建策略。
	p := &SysAccessPolicy{
		PolicyCode:     "P-TEST-01",
		PolicyName:     "测试策略",
		Effect:         EffectAllow,
		ConditionsJSON: `{"axis":"data"}`,
		Priority:       10,
		Description:    "测试用策略",
	}
	if err := CreatePolicy(ctx, db, p); err != nil {
		t.Fatalf("创建策略失败: %v", err)
	}
	if p.ID == "" {
		t.Error("期望策略 ID 非空")
	}

	// 查询策略。
	retrieved, err := GetPolicy(ctx, db, p.ID)
	if err != nil {
		t.Fatalf("查询策略失败: %v", err)
	}
	if retrieved.PolicyCode != "P-TEST-01" {
		t.Errorf("期望 policy_code=P-TEST-01，实际 %s", retrieved.PolicyCode)
	}

	// 更新策略。
	retrieved.PolicyName = "更新后的测试策略"
	retrieved.Description = "更新后的描述"
	if err := UpdatePolicy(ctx, db, retrieved); err != nil {
		t.Fatalf("更新策略失败: %v", err)
	}

	// 删除策略。
	if err := DeletePolicy(ctx, db, p.ID); err != nil {
		t.Fatalf("删除策略失败: %v", err)
	}

	// 确认已删除。
	_, err = GetPolicy(ctx, db, p.ID)
	if err == nil {
		t.Error("期望策略已删除，但仍能查询到")
	}
}

// TestDeleteSystemPolicy_Denied 验证系统策略不可删除。
func TestDeleteSystemPolicy_Denied(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 种子内置策略。
	if _, err := SeedBuiltinPolicies(ctx, db); err != nil {
		t.Fatalf("种子策略失败: %v", err)
	}

	// 获取一条系统策略。
	policies, err := ListPolicies(ctx, db, "")
	if err != nil {
		t.Fatalf("查询策略失败: %v", err)
	}
	if len(policies) == 0 {
		t.Fatal("没有内置策略可供测试")
	}

	// 尝试删除系统策略 —— 应拒绝。
	err = DeletePolicy(ctx, db, policies[0].ID)
	if err == nil {
		t.Error("期望删除系统策略被拒绝")
	}
	if !errors.Is(err, ErrSystemPolicy) {
		t.Errorf("期望 ErrSystemPolicy，实际: %v", err)
	}
}

// TestBuiltinPolicies_Seed 验证内置策略的种子写入和幂等性。
// 种子操作创建 4 条策略、4 个 SOD 系统角色、4 条策略绑定——共 12 个实体。
func TestBuiltinPolicies_Seed(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 首次种子写入——应创建 12 个实体（策略 + 角色 + 绑定）。
	n, err := SeedBuiltinPolicies(ctx, db)
	if err != nil {
		t.Fatalf("首次种子失败: %v", err)
	}
	if n != 12 {
		t.Errorf("期望首次种子创建 12 个实体（4策略+4角色+4绑定），实际 %d", n)
	}

	// 再次种子写入 —— 幂等，不重复创建。
	n, err = SeedBuiltinPolicies(ctx, db)
	if err != nil {
		t.Fatalf("二次种子失败: %v", err)
	}
	if n != 0 {
		t.Errorf("期望二次种子创建 0 个实体，实际 %d", n)
	}

	// 验证策略数量。
	policies, err := ListPolicies(ctx, db, "")
	if err != nil {
		t.Fatalf("查询策略列表失败: %v", err)
	}
	if len(policies) != 4 {
		t.Errorf("期望 4 条内置策略，实际 %d", len(policies))
	}
	for _, p := range policies {
		if !p.IsSystem {
			t.Errorf("期望策略 %s 的 is_system=true，实际 %v", p.PolicyCode, p.IsSystem)
		}
	}

	// 验证 SOD 系统角色数量。
	roles, err := ListRoles(ctx, db)
	if err != nil {
		t.Fatalf("查询角色列表失败: %v", err)
	}
	if len(roles) != 4 {
		t.Errorf("期望 4 个 SOD 系统角色，实际 %d", len(roles))
	}
	for _, r := range roles {
		if !r.IsSystem {
			t.Errorf("期望角色 %s 的 is_system=true，实际 %v", r.RoleCode, r.IsSystem)
		}
	}

	// 验证策略绑定数量（每条策略绑定到对应的 SOD 角色）。
	var count int64
	db.WithContext(ctx).Model(&SysAccessPolicyBinding{}).
		Where("subject_type = ?", SubjectTypeRole).Count(&count)
	if count != 4 {
		t.Errorf("期望 4 条 SOD 策略绑定，实际 %d", count)
	}
}

// ── 角色 CRUD 测试 ─────────────────────────────────────────────────────

// TestRoleCRUD 验证角色的基本 CRUD 操作和权限绑定。
func TestRoleCRUD(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 创建角色。
	role := &SysRole{
		RoleCode: "test_role",
		RoleName: "测试角色",
	}
	if err := CreateRole(ctx, db, role); err != nil {
		t.Fatalf("创建角色失败: %v", err)
	}

	// 创建操作并授予权限。
	action1 := registerAction(t, db, "data.usage.read", "读取用量", AxisData, "usage")
	action2 := registerAction(t, db, "data.model.list", "列出模型", AxisData, "model")
	if err := GrantPermission(ctx, db, role.ID, []string{action1.ID, action2.ID}); err != nil {
		t.Fatalf("授予权限失败: %v", err)
	}

	// 查询角色权限。
	perms, err := GetRolePermissions(ctx, db, role.ID)
	if err != nil {
		t.Fatalf("查询角色权限失败: %v", err)
	}
	if len(perms) != 2 {
		t.Errorf("期望 2 条权限，实际 %d", len(perms))
	}

	// 撤销权限。
	if err := RevokePermission(ctx, db, role.ID, []string{action1.ID}); err != nil {
		t.Fatalf("撤销权限失败: %v", err)
	}

	// 验证仅剩 1 条权限。
	perms, err = GetRolePermissions(ctx, db, role.ID)
	if err != nil {
		t.Fatalf("查询角色权限失败: %v", err)
	}
	if len(perms) != 1 {
		t.Errorf("撤销后期望 1 条权限，实际 %d", len(perms))
	}
}

// TestDeleteSystemRole_Denied 验证系统角色不可删除。
func TestDeleteSystemRole_Denied(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()

	// 创建系统角色。
	role := &SysRole{
		RoleCode: "super_admin",
		RoleName: "超级管理员",
		IsSystem: true,
	}
	if err := CreateRole(ctx, db, role); err != nil {
		t.Fatalf("创建系统角色失败: %v", err)
	}

	// 尝试删除 —— 应拒绝。
	err := DeleteRole(ctx, db, role.ID)
	if err == nil {
		t.Error("期望删除系统角色被拒绝")
	}
	if !errors.Is(err, ErrSystemRole) {
		t.Errorf("期望 ErrSystemRole，实际: %v", err)
	}
}

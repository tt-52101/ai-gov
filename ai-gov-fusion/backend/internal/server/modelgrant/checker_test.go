package modelgrant

import (
	"context"
	"testing"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestDB 打开内存 SQLite 数据库，执行表迁移，返回 *gorm.DB。
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

// newTestChecker 创建绑定到测试数据库的 Checker。
func newTestChecker(t *testing.T) *Checker {
	t.Helper()
	return NewChecker(newTestDB(t))
}

// setupGrant 快速创建一条授权规则并返回其 ID。
func setupGrant(t *testing.T, db *gorm.DB, mg *ModelGrant) string {
	t.Helper()
	if err := CreateModelGrant(db, mg); err != nil {
		t.Fatalf("CreateModelGrant 失败: %v", err)
	}
	return mg.ID
}

// ── CheckAccess 测试 ────────────────────────────────────────────────────────

// TestCheckAccess_Allow 验证 ALLOW 规则放行。
func TestCheckAccess_Allow(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	principal := Principal{Type: PrincipalParty, ID: "party-1"}
	modelID := "gpt-4"

	setupGrant(t, c.DB, &ModelGrant{
		ID:            "mg-allow-1",
		PrincipalType: PrincipalParty,
		PrincipalID:   "party-1",
		ModelID:       strPtr(modelID),
		Effect:        EffectAllow,
	})

	if err := c.CheckAccess(ctx, principal, modelID); err != nil {
		t.Errorf("ALLOW 规则应放行，却返回错误: %v", err)
	}
}

// TestCheckAccess_Deny 验证 DENY 优先于 ALLOW。
func TestCheckAccess_Deny(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	principal := Principal{Type: PrincipalParty, ID: "party-1"}
	modelID := "gpt-4.5-preview"

	// ALLOW 规则。
	setupGrant(t, c.DB, &ModelGrant{
		ID:            "mg-allow-1",
		PrincipalType: PrincipalParty,
		PrincipalID:   "party-1",
		ModelID:       strPtr(modelID),
		Effect:        EffectAllow,
	})
	// DENY 规则。
	setupGrant(t, c.DB, &ModelGrant{
		ID:            "mg-deny-1",
		PrincipalType: PrincipalParty,
		PrincipalID:   "party-1",
		ModelID:       strPtr(modelID),
		Effect:        EffectDeny,
	})

	err := c.CheckAccess(ctx, principal, modelID)
	if err == nil {
		t.Error("DENY 规则应拒绝，却放行")
	}
	if err != ErrModelAccessDenied {
		t.Errorf("期望 ErrModelAccessDenied，得到 %v", err)
	}
}

// TestCheckAccess_KeyOverParty 验证 Key 级权限优先于 Party 级。
func TestCheckAccess_KeyOverParty(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	modelID := "claude-3"

	// Party 级 ALLOW。
	setupGrant(t, c.DB, &ModelGrant{
		ID:            "mg-party-allow",
		PrincipalType: PrincipalParty,
		PrincipalID:   "party-1",
		ModelID:       strPtr(modelID),
		Effect:        EffectAllow,
	})
	// Key 级 DENY——应覆盖 Party ALLOW。
	setupGrant(t, c.DB, &ModelGrant{
		ID:            "mg-key-deny",
		PrincipalType: PrincipalKey,
		PrincipalID:   "key-1",
		ModelID:       strPtr(modelID),
		Effect:        EffectDeny,
	})

	// 以 Key 主体查询——Key 级 DENY 应优先。
	err := c.CheckAccess(ctx, Principal{Type: PrincipalKey, ID: "key-1"}, modelID)
	if err == nil {
		t.Error("Key 级 DENY 应拒绝，却放行")
	}
	if err != ErrModelAccessDenied {
		t.Errorf("期望 ErrModelAccessDenied，得到 %v", err)
	}

	// 以 Party 主体查询——Party 级 ALLOW 应放行。
	if err := c.CheckAccess(ctx, Principal{Type: PrincipalParty, ID: "party-1"}, modelID); err != nil {
		t.Errorf("Party 级 ALLOW 应放行，却返回错误: %v", err)
	}
}

// TestCheckAccess_DefaultDeny 验证无规则时默认拒绝。
func TestCheckAccess_DefaultDeny(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	principal := Principal{Type: PrincipalParty, ID: "party-empty"}
	modelID := "deepseek-v3"

	// 该 party 上未配置任何规则。
	err := c.CheckAccess(ctx, principal, modelID)
	if err == nil {
		t.Error("无规则应默认拒绝，却放行")
	}
	if err != ErrModelAccessDenied {
		t.Errorf("期望 ErrModelAccessDenied，得到 %v", err)
	}
}

// ── CheckQuotaLimit 测试 ────────────────────────────────────────────────────

// TestCheckQuotaLimit_Exceeded 验证配额超限 → MODEL_BUDGET_EXCEEDED。
func TestCheckQuotaLimit_Exceeded(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	principal := Principal{Type: PrincipalParty, ID: "party-quota"}
	modelID := "gpt-4"
	limit := decimal.NewFromFloat(100.0)
	consumed := decimal.NewFromFloat(95.0)

	mg := &ModelGrant{
		ID:            "mg-quota-1",
		PrincipalType: PrincipalParty,
		PrincipalID:   "party-quota",
		ModelID:       strPtr(modelID),
		Effect:        EffectAllow,
		QuotaLimit:    &limit,
		QuotaConsumed: consumed,
	}
	setupGrant(t, c.DB, mg)

	// 预估 10 元 → 95 + 10 = 105 > 100 → 应超限。
	estimated := decimal.NewFromFloat(10.0)
	err := c.CheckQuotaLimit(ctx, principal, modelID, estimated)
	if err == nil {
		t.Error("配额超限应返回错误，却为 nil")
	}
	if err != ErrModelBudgetExceeded {
		t.Errorf("期望 ErrModelBudgetExceeded，得到 %v", err)
	}
}

// TestCheckQuotaLimit_Within 验证配额内通过。
func TestCheckQuotaLimit_Within(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	principal := Principal{Type: PrincipalParty, ID: "party-quota-ok"}
	modelID := "gpt-4"
	limit := decimal.NewFromFloat(100.0)
	consumed := decimal.NewFromFloat(50.0)

	mg := &ModelGrant{
		ID:            "mg-quota-2",
		PrincipalType: PrincipalParty,
		PrincipalID:   "party-quota-ok",
		ModelID:       strPtr(modelID),
		Effect:        EffectAllow,
		QuotaLimit:    &limit,
		QuotaConsumed: consumed,
	}
	setupGrant(t, c.DB, mg)

	// 预估 30 元 → 50 + 30 = 80 < 100 → 应通过。
	estimated := decimal.NewFromFloat(30.0)
	if err := c.CheckQuotaLimit(ctx, principal, modelID, estimated); err != nil {
		t.Errorf("配额内应通过，却返回错误: %v", err)
	}
}

// ── ConsumeQuota 测试 ───────────────────────────────────────────────────────

// TestConsumeQuota_Accumulates 验证 ConsumeQuota 正确累加消耗。
func TestConsumeQuota_Accumulates(t *testing.T) {
	c := newTestChecker(t)
	ctx := context.Background()

	principal := Principal{Type: PrincipalParty, ID: "party-cq"}
	modelID := "gpt-4"
	limit := decimal.NewFromFloat(100.0)

	mg := &ModelGrant{
		ID:            "mg-cq-1",
		PrincipalType: PrincipalParty,
		PrincipalID:   "party-cq",
		ModelID:       strPtr(modelID),
		Effect:        EffectAllow,
		QuotaLimit:    &limit,
		QuotaConsumed: decimal.NewFromFloat(20.0),
	}
	setupGrant(t, c.DB, mg)

	// 消耗 15 元 → 累计应 = 35。
	if err := c.ConsumeQuota(ctx, principal, modelID, decimal.NewFromFloat(15.0)); err != nil {
		t.Fatalf("ConsumeQuota 失败: %v", err)
	}

	// 重新查询验证。
	updated, err := GetModelGrant(c.DB, "mg-cq-1")
	if err != nil {
		t.Fatalf("GetModelGrant 失败: %v", err)
	}
	expected := decimal.NewFromFloat(35.0)
	if !updated.QuotaConsumed.Equal(expected) {
		t.Errorf("QuotaConsumed 期望 %s，实际 %s", expected.String(), updated.QuotaConsumed.String())
	}

	// 再消耗 70 元 → 累计应 = 105，超限。
	_ = c.ConsumeQuota(ctx, principal, modelID, decimal.NewFromFloat(70.0))
	err = c.CheckQuotaLimit(ctx, principal, modelID, decimal.NewFromFloat(1.0))
	if err != ErrModelBudgetExceeded {
		t.Errorf("累计 105 应超限，却得到: %v", err)
	}
}

// ── 工具函数 ────────────────────────────────────────────────────────────────

// strPtr 返回字符串指针，方便在测试中设置 ModelGrant 的可选字段。
func strPtr(s string) *string { return &s }

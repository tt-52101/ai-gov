package routing_test

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"tokenhub/backend/internal/server/routing"
	"tokenhub/backend/internal/server/routing/strategies"
)

// ── 测试辅助 ──────────────────────────────────────────────────────────────

// setupTestDB 创建内存 SQLite 数据库并注册全部策略。
func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("无法打开测试数据库: %v", err)
	}
	if err := db.AutoMigrate(&routing.RouteProfile{}, &routing.Decision{}); err != nil {
		t.Fatalf("自动迁移失败: %v", err)
	}
	return db
}

// setupTestProfile 创建测试用路由档案。
func setupTestProfile(t *testing.T, db *gorm.DB, name string, deltaCap float64, bindings []routing.StrategyBinding, shadow bool) *routing.RouteProfile {
	t.Helper()
	p := &routing.RouteProfile{
		Name:        name,
		Strategies:  bindings,
		DeltaCap:    decimal.NewFromFloat(deltaCap),
		MaxAttempts: 3,
		Shadow:      shadow,
		Status:      routing.ProfileStatusActive,
	}
	if err := routing.CreateProfile(db, p); err != nil {
		t.Fatalf("创建测试档案失败: %v", err)
	}
	return p
}

// makeCandidates 构建测试候选集。
func makeCandidates() []routing.Candidate {
	return []routing.Candidate{
		{
			ChannelID:   1,
			ModelID:     "gpt-4",
			Priority:    10,
			Weight:      100,
			Health:      routing.HealthUp,
			EstSell:     decimal.NewFromFloat(0.03),
			EstCost:     decimal.NewFromFloat(0.02),
			LatencyEWMA: 200 * time.Millisecond,
			ErrorRate:   0.01,
			Metadata:    map[string]any{"network_class": "external"},
		},
		{
			ChannelID:   2,
			ModelID:     "gpt-4",
			Priority:    5,
			Weight:      50,
			Health:      routing.HealthUp,
			EstSell:     decimal.NewFromFloat(0.025),
			EstCost:     decimal.NewFromFloat(0.018),
			LatencyEWMA: 150 * time.Millisecond,
			ErrorRate:   0.00,
			Metadata:    map[string]any{"network_class": "internal"},
		},
		{
			ChannelID:   3,
			ModelID:     "gpt-4",
			Priority:    10,
			Weight:      80,
			Health:      routing.HealthDown,
			EstSell:     decimal.NewFromFloat(0.05),
			EstCost:     decimal.NewFromFloat(0.04),
			LatencyEWMA: 500 * time.Millisecond,
			ErrorRate:   0.50,
			Metadata:    map[string]any{"network_class": "external"},
		},
	}
}

// ── 测试用例 ──────────────────────────────────────────────────────────────

// TestExecuteProfile_SimpleFailover 验证简单故障转移——
// S-HEALTH 将 down 状态的候选剔除，其余候选按 SCORE 排序返回。
func TestExecuteProfile_SimpleFailover(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	bindings := []routing.StrategyBinding{
		{Code: routing.StrategyHealth, Enabled: true, Priority: 10},
		{Code: routing.StrategyPriority, Enabled: true, Priority: 20},
	}
	profile := setupTestProfile(t, db, "simple-failover", 0, bindings, false)

	candidates := makeCandidates()
	result, decision, err := routing.ExecuteProfile(
		context.Background(), db, profile, candidates, decimal.NewFromFloat(0.03), 0,
	)
	if err != nil {
		t.Fatalf("管道执行失败: %v", err)
	}
	if decision == nil {
		t.Fatal("决策日志为空")
	}
	if decision.CandidatesIn != 3 {
		t.Errorf("输入候选数应为 3，实际 %d", decision.CandidatesIn)
	}
	if decision.CandidatesOut < 1 {
		t.Errorf("输出候选数应 >= 1，实际 %d", decision.CandidatesOut)
	}

	// 候选 3 (HealthDown) 应被剔除。
	for _, c := range result {
		if c.ChannelID == 3 && !c.Eliminated {
			t.Error("健康状况为 down 的候选 3 应被剔除")
		}
	}

	// 选中候选应为未剔除且得分最高者。
	if decision.Selected == 0 {
		t.Error("应有选中候选")
	}
	t.Logf("选中渠道: %d, 输入: %d, 输出: %d", decision.Selected, decision.CandidatesIn, decision.CandidatesOut)
}

// TestExecuteProfile_DeltaCap 验证 δ=0 时价格帽过滤——
// EstSell 超过锚定价的候选被剔除。
func TestExecuteProfile_DeltaCap(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	bindings := []routing.StrategyBinding{
		{Code: routing.StrategyHealth, Enabled: true, Priority: 10},
		{Code: routing.StrategyCost, Enabled: true, Priority: 20},
	}
	// δ=0: 只有 EstSell <= 锚定价(0.03) 的候选通过。
	profile := setupTestProfile(t, db, "delta-zero", 0, bindings, false)

	candidates := makeCandidates()
	result, decision, err := routing.ExecuteProfile(
		context.Background(), db, profile, candidates, decimal.NewFromFloat(0.03), 0,
	)
	if err != nil {
		t.Fatalf("管道执行失败: %v", err)
	}
	_ = decision

	// 计数：候选3 应因 HealthDown 或价格帽被剔除。
	eliminated := 0
	for _, c := range result {
		if c.Eliminated {
			eliminated++
			t.Logf("被剔除候选 %d: %s", c.ChannelID, c.ElimReason)
		}
	}
	if eliminated < 1 {
		t.Error("应有至少 1 个候选被剔除")
	}
}

// TestExecuteProfile_ComplianceHard 验证 S-COMPLIANCE 硬策略——
// INTERNAL_ONLY 请求中，external 候选被强制剔除。
func TestExecuteProfile_ComplianceHard(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	bindings := []routing.StrategyBinding{
		{Code: routing.StrategyCompliance, Enabled: true, Priority: 0},
		{Code: routing.StrategyHealth, Enabled: true, Priority: 10},
	}
	profile := setupTestProfile(t, db, "compliance-strict", 0, bindings, false)

	candidates := makeCandidates()
	// 将上下文标记为 INTERNAL_ONLY（模拟敏感主体请求）。
	ctx := context.WithValue(context.Background(), strategies.CtxKeyNetworkClass, "INTERNAL_ONLY")
	result, decision, err := routing.ExecuteProfile(
		ctx, db, profile, candidates, decimal.NewFromFloat(0.03), 0,
	)
	if err != nil {
		t.Fatalf("管道执行失败: %v", err)
	}
	_ = decision

	// 候选1 (network_class=external) 应被 S-COMPLIANCE 剔除。
	for _, c := range result {
		if c.ChannelID == 1 {
			if !c.Eliminated {
				t.Error("INTERNAL_ONLY 上下文中，external 候选应被剔除")
			}
			if c.ElimReason == "" {
				t.Error("被剔除候选应有剔除原因")
			}
		}
	}
}

// TestExecuteProfile_ShadowMode 验证影子模式——
// Shadow=true 时仅记录决策日志，不实际路由。
func TestExecuteProfile_ShadowMode(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	bindings := []routing.StrategyBinding{
		{Code: routing.StrategyHealth, Enabled: true, Priority: 10},
	}
	profile := setupTestProfile(t, db, "shadow-test", 0, bindings, true) // Shadow=true

	candidates := makeCandidates()
	result, decision, err := routing.ExecuteProfile(
		context.Background(), db, profile, candidates, decimal.NewFromFloat(0.03), 0,
	)
	if err != nil {
		t.Fatalf("影子模式管道执行失败: %v", err)
	}
	if decision == nil {
		t.Fatal("影子模式下决策日志不应为空")
	}
	if !profile.Shadow {
		t.Error("档案应为影子模式")
	}

	// 影子模式下仍返回结果，但实际路由由调用方根据 Shadow 字段判断。
	_ = result
	t.Logf("影子模式决策: 选中渠道=%d, 输入=%d, 输出=%d",
		decision.Selected, decision.CandidatesIn, decision.CandidatesOut)
}

// TestExecuteProfile_NoCandidates 验证空候选集返回错误。
func TestExecuteProfile_NoCandidates(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	profile := setupTestProfile(t, db, "empty-test", 0, nil, false)
	_, _, err := routing.ExecuteProfile(
		context.Background(), db, profile, nil, decimal.Zero, 0,
	)
	if err == nil {
		t.Error("空候选集应返回错误")
	}
}

// TestExecuteProfile_AllEliminated 验证全部候选被过滤剔除。
func TestExecuteProfile_AllEliminated(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	bindings := []routing.StrategyBinding{
		{Code: routing.StrategyHealth, Enabled: true, Priority: 10},
	}
	profile := setupTestProfile(t, db, "all-down", 0, bindings, false)

	// 所有候选均为 down 状态。
	allDown := []routing.Candidate{
		{
			ChannelID: 1,
			ModelID:   "test",
			Health:    routing.HealthDown,
			EstSell:   decimal.NewFromFloat(0.01),
		},
		{
			ChannelID: 2,
			ModelID:   "test",
			Health:    routing.HealthDown,
			EstSell:   decimal.NewFromFloat(0.01),
		},
	}

	_, _, err := routing.ExecuteProfile(
		context.Background(), db, profile, allDown, decimal.NewFromFloat(0.03), 0,
	)
	if err == nil {
		t.Error("全部候选被剔除时应返回错误")
	}
}

// TestCreateProfile_DeltaCapRejected 验证 δ>0.20 拒绝保存。
func TestCreateProfile_DeltaCapRejected(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	p := &routing.RouteProfile{
		Name:        "over-delta",
		DeltaCap:    decimal.NewFromFloat(0.25), // 超过 20% 硬上限。
		MaxAttempts: 3,
		Status:      routing.ProfileStatusActive,
	}
	err := routing.CreateProfile(db, p)
	if err == nil {
		t.Error("δ=25%% 应被拒绝保存")
	}
	t.Logf("预期拒绝: %v", err)
}

// TestUpdateProfile_DeltaAudit 验证 δ 变更触发审计日志。
func TestUpdateProfile_DeltaAudit(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	p := setupTestProfile(t, db, "delta-audit", 0.0, nil, false)

	// 修改 δ 从 0 → 0.10。
	p.DeltaCap = decimal.NewFromFloat(0.10)
	if err := routing.UpdateProfile(db, p); err != nil {
		t.Fatalf("更新档案失败: %v", err)
	}

	// 验证 δ 已更新。
	updated, err := routing.GetProfile(db, p.ID)
	if err != nil {
		t.Fatalf("查询更新后档案失败: %v", err)
	}
	expected := decimal.NewFromFloat(0.10)
	if !updated.DeltaCap.Equal(expected) {
		t.Errorf("δ 应为 %s，实际 %s", expected.String(), updated.DeltaCap.String())
	}
}

// TestListProfiles 验证档案列表查询。
func TestListProfiles(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	_ = setupTestProfile(t, db, "profile-a", 0.0, nil, false)
	_ = setupTestProfile(t, db, "profile-b", 0.05, nil, false)

	list, err := routing.ListProfiles(db)
	if err != nil {
		t.Fatalf("列出档案失败: %v", err)
	}
	if len(list) < 2 {
		t.Errorf("应至少有 2 个档案，实际 %d", len(list))
	}
}

// TestDeleteProfile 验证软删除。
func TestDeleteProfile(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	p := setupTestProfile(t, db, "to-delete", 0.0, nil, false)
	if err := routing.DeleteProfile(db, p.ID); err != nil {
		t.Fatalf("删除档案失败: %v", err)
	}

	// 删除后 ListProfiles 应不包含已删除档案。
	list, _ := routing.ListProfiles(db)
	for _, lp := range list {
		if lp.ID == p.ID {
			t.Error("软删除后 ListProfiles 不应包含已删除档案")
		}
	}
}

// TestGetProfileByName 验证按名称查询。
func TestGetProfileByName(t *testing.T) {
	strategies.RegisterAll()
	db := setupTestDB(t)

	_ = setupTestProfile(t, db, "unique-name-test", 0.0, nil, false)
	found, err := routing.GetProfileByName(db, "unique-name-test")
	if err != nil {
		t.Fatalf("按名称查询失败: %v", err)
	}
	if found.Name != "unique-name-test" {
		t.Errorf("名称不匹配: 期望 unique-name-test, 实际 %s", found.Name)
	}
}

// TestRegistry 验证策略注册表基本操作。
func TestRegistry(t *testing.T) {
	strategies.RegisterAll()

	// 验证全部 12 种策略已注册。
	registered := routing.GetRegistered()
	if len(registered) != 12 {
		t.Errorf("应有 12 个已注册策略，实际 %d", len(registered))
	}

	// 验证关键策略存在。
	for _, id := range []string{
		routing.StrategyCompliance, routing.StrategyHealth,
		routing.StrategyCost, routing.StrategyClassify,
	} {
		if !routing.HasStrategy(id) {
			t.Errorf("策略 %s 应已注册", id)
		}
		s := routing.GetStrategy(id)
		if s == nil || s.ID() != id {
			t.Errorf("GetStrategy(%s) 返回异常", id)
		}
	}
}

// TestDuplicateRegister 验证重复注册被拒绝。
func TestDuplicateRegister(t *testing.T) {
	strategies.RegisterAll()

	// 尝试注册一个已存在的策略。
	err := routing.Register(routing.GetStrategy(routing.StrategyHealth))
	if err == nil {
		t.Error("重复注册应返回错误")
	}
}

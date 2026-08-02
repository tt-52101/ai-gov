package strategies

import (
	"context"
	"testing"
	"time"

	"tokenhub/backend/internal/server/routing"

	"github.com/shopspring/decimal"
)

// ── 测试辅助函数 ────────────────────────────────────────────────────────────

// newCandidate 创建测试用的路由候选。
func newCandidate(id int64, opts ...func(*routing.Candidate)) routing.Candidate {
	c := routing.Candidate{
		ChannelID: id,
		ModelID:   "test-model",
		Priority:  0,
		Weight:    1.0,
		Health:    routing.HealthUp,
		EstSell:   decimal.Zero,
		ErrorRate: 0,
		Score:     0,
		Metadata:  make(map[string]any),
	}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// withPriority 设置候选的优先级。
func withPriority(p int) func(*routing.Candidate) {
	return func(c *routing.Candidate) { c.Priority = p }
}

// withWeight 设置候选的权重。
func withWeight(w float64) func(*routing.Candidate) {
	return func(c *routing.Candidate) { c.Weight = w }
}

// withHealth 设置候选的健康状态。
func withHealth(h string) func(*routing.Candidate) {
	return func(c *routing.Candidate) { c.Health = h }
}

// withEstSell 设置候选的预估内部价。
func withEstSell(s string) func(*routing.Candidate) {
	return func(c *routing.Candidate) {
		c.EstSell, _ = decimal.NewFromString(s)
	}
}

// withLatency 设置候选的 EWMA 延迟。
func withLatency(d time.Duration) func(*routing.Candidate) {
	return func(c *routing.Candidate) { c.LatencyEWMA = d }
}

// withErrorRate 设置候选的错误率。
func withErrorRate(r float64) func(*routing.Candidate) {
	return func(c *routing.Candidate) { c.ErrorRate = r }
}

// withMetadata 设置候选的元数据。
func withMetadata(key string, value any) func(*routing.Candidate) {
	return func(c *routing.Candidate) {
		if c.Metadata == nil {
			c.Metadata = make(map[string]any)
		}
		c.Metadata[key] = value
	}
}

// withEliminated 将候选标记为已剔除。
func withEliminated(reason string) func(*routing.Candidate) {
	return func(c *routing.Candidate) {
		c.Eliminated = true
		c.ElimReason = reason
	}
}

// ── PriorityStrategy 测试 ──────────────────────────────────────────────────

// TestPriorityStrategy_Filter 验证优先级策略的过滤行为。
func TestPriorityStrategy_Filter(t *testing.T) {
	s := &PriorityStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withPriority(3)),
		newCandidate(2, withPriority(1)),
		newCandidate(3, withPriority(2)),
		newCandidate(4, withPriority(3)),
	}

	result := s.Filter(context.Background(), candidates)

	// 优先级 3 为最高，应只保留优先级 3 的候选（1 和 4）。
	for _, c := range result {
		if c.ChannelID == 2 || c.ChannelID == 3 {
			if !c.Eliminated {
				t.Errorf("优先级 1 或 2 的候选 %d 应被剔除", c.ChannelID)
			}
		}
		if (c.ChannelID == 1 || c.ChannelID == 4) && c.Eliminated {
			t.Errorf("最高优先级候选 %d 不应被剔除", c.ChannelID)
		}
	}
}

// TestPriorityStrategy_Filter_AllEliminated 验证所有候选被剔除时不做过滤。
func TestPriorityStrategy_Filter_AllEliminated(t *testing.T) {
	s := &PriorityStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withPriority(3), withEliminated("已剔除")),
		newCandidate(2, withPriority(1), withEliminated("已剔除")),
	}

	result := s.Filter(context.Background(), candidates)
	if len(result) != 2 {
		t.Fatalf("期望 2 个候选，实际 %d", len(result))
	}
}

// TestPriorityStrategy_Filter_Empty 验证空候选列表不崩溃。
func TestPriorityStrategy_Filter_Empty(t *testing.T) {
	s := &PriorityStrategy{}
	result := s.Filter(context.Background(), []routing.Candidate{})
	if len(result) != 0 {
		t.Fatalf("期望空列表，实际 %d", len(result))
	}
}

// TestPriorityStrategy_Score 验证优先级策略的评分行为。
func TestPriorityStrategy_Score(t *testing.T) {
	s := &PriorityStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withPriority(3)),
		newCandidate(2, withPriority(1)),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 3 {
		t.Errorf("候选 1 期望分数 3，实际 %f", result[0].Score)
	}
	if result[1].Score != 1 {
		t.Errorf("候选 2 期望分数 1，实际 %f", result[1].Score)
	}
}

// ── HealthStrategy 测试 ────────────────────────────────────────────────────

// TestHealthStrategy_Filter 验证健康策略的过滤行为。
func TestHealthStrategy_Filter(t *testing.T) {
	s := &HealthStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withHealth(routing.HealthUp)),
		newCandidate(2, withHealth(routing.HealthDown)),
		newCandidate(3, withHealth(routing.HealthDegraded)),
	}

	result := s.Filter(context.Background(), candidates)

	// down 候选应被剔除。
	if !result[1].Eliminated {
		t.Errorf("down 候选应被剔除")
	}
	if result[1].ElimReason != "S-HEALTH: 渠道健康状态为 down，已剔除" {
		t.Errorf("剔除原因不匹配: %s", result[1].ElimReason)
	}

	// up 和 degraded 不应被剔除。
	if result[0].Eliminated {
		t.Errorf("up 候选不应被剔除")
	}
	if result[2].Eliminated {
		t.Errorf("degraded 候选不应被剔除")
	}
}

// TestHealthStrategy_Score 验证健康策略的评分行为。
func TestHealthStrategy_Score(t *testing.T) {
	s := &HealthStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withHealth(routing.HealthUp)),
		newCandidate(2, withHealth(routing.HealthDegraded)),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("up 候选期望分数 0，实际 %f", result[0].Score)
	}
	if result[1].Score != -5 {
		t.Errorf("degraded 候选期望分数 -5，实际 %f", result[1].Score)
	}
}

// TestHealthStrategy_Empty 验证空候选列表不崩溃。
func TestHealthStrategy_Empty(t *testing.T) {
	s := &HealthStrategy{}
	if result := s.Filter(context.Background(), []routing.Candidate{}); len(result) != 0 {
		t.Fatalf("期望空列表")
	}
	if result := s.Score(context.Background(), []routing.Candidate{}); len(result) != 0 {
		t.Fatalf("期望空列表")
	}
}

// ── WeightStrategy 测试 ────────────────────────────────────────────────────

// TestWeightStrategy_Score 验证权重策略的评分行为。
func TestWeightStrategy_Score(t *testing.T) {
	s := &WeightStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withWeight(10)),
		newCandidate(2, withWeight(30)),
	}

	result := s.Score(context.Background(), candidates)
	// 总权重 40，候选 1 分数 = 10/40 * 10 = 2.5，候选 2 分数 = 30/40 * 10 = 7.5
	if result[0].Score != 2.5 {
		t.Errorf("候选 1 期望分数 2.5，实际 %f", result[0].Score)
	}
	if result[1].Score != 7.5 {
		t.Errorf("候选 2 期望分数 7.5，实际 %f", result[1].Score)
	}
}

// TestWeightStrategy_Score_ZeroWeight 验证零权重不导致 NaN。
func TestWeightStrategy_Score_ZeroWeight(t *testing.T) {
	s := &WeightStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withWeight(0)),
		newCandidate(2, withWeight(0)),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("零权重候选期望分数 0，实际 %f", result[0].Score)
	}
}

// TestWeightStrategy_Score_AllEliminated 验证所有候选被剔除时评分不崩溃。
func TestWeightStrategy_Score_AllEliminated(t *testing.T) {
	s := &WeightStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withWeight(10), withEliminated("已剔除")),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("已剔除候选期望分数 0，实际 %f", result[0].Score)
	}
}

// ── AffinityStrategy 测试 ──────────────────────────────────────────────────

// TestAffinityStrategy_Score_Hit 验证会话亲和命中的加分行为。
func TestAffinityStrategy_Score_Hit(t *testing.T) {
	s := &AffinityStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("affinity_hit", true)),
		newCandidate(2),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 8 {
		t.Errorf("亲和命中候选期望分数 8，实际 %f", result[0].Score)
	}
	if result[1].Score != 0 {
		t.Errorf("非亲和命中候选期望分数 0，实际 %f", result[1].Score)
	}
}

// TestAffinityStrategy_Score_Eliminated 验证已剔除候选不加分。
func TestAffinityStrategy_Score_Eliminated(t *testing.T) {
	s := &AffinityStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("affinity_hit", true), withEliminated("已剔除")),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("已剔除候选期望分数 0，实际 %f", result[0].Score)
	}
}

// ── CostStrategy 测试 ──────────────────────────────────────────────────────

// TestCostStrategy_Score 验证成本策略的评分行为。
func TestCostStrategy_Score(t *testing.T) {
	s := &CostStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withEstSell("10")),
		newCandidate(2, withEstSell("5")), // 最低价
		newCandidate(3, withEstSell("20")),
	}

	result := s.Score(context.Background(), candidates)
	// 最低价 5，候选 2 得 10 分，候选 1 得 10*(5/10)=5 分，候选 3 得 10*(5/20)=2.5 分
	if result[1].Score != 10 {
		t.Errorf("最低价候选期望分数 10，实际 %f", result[1].Score)
	}
	if result[0].Score != 5 {
		t.Errorf("候选 1 期望分数 5，实际 %f", result[0].Score)
	}
	if result[2].Score != 2.5 {
		t.Errorf("候选 3 期望分数 2.5，实际 %f", result[2].Score)
	}
}

// TestCostStrategy_Score_AllZero 验证所有 EstSell 为零时均得满分。
func TestCostStrategy_Score_AllZero(t *testing.T) {
	s := &CostStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withEstSell("0")),
		newCandidate(2, withEstSell("0")),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 10 {
		t.Errorf("零价候选期望分数 10，实际 %f", result[0].Score)
	}
	if result[1].Score != 10 {
		t.Errorf("零价候选期望分数 10，实际 %f", result[1].Score)
	}
}

// TestCostStrategy_Empty 验证空候选列表不崩溃。
func TestCostStrategy_Empty(t *testing.T) {
	s := &CostStrategy{}
	result := s.Score(context.Background(), []routing.Candidate{})
	if len(result) != 0 {
		t.Fatalf("期望空列表")
	}
}

// ── LatencyStrategy 测试 ───────────────────────────────────────────────────

// TestLatencyStrategy_Score 验证延迟策略的评分行为。
func TestLatencyStrategy_Score(t *testing.T) {
	s := &LatencyStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withLatency(200*time.Millisecond)),
		newCandidate(2, withLatency(100*time.Millisecond)), // 最低延迟
		newCandidate(3, withLatency(400*time.Millisecond)),
	}

	result := s.Score(context.Background(), candidates)
	// 最低延迟 100ms，候选 2 得 10 分，候选 1 得 10*(100/200)=5 分，候选 3 得 10*(100/400)=2.5 分
	if result[1].Score != 10 {
		t.Errorf("最低延迟候选期望分数 10，实际 %f", result[1].Score)
	}
	if result[0].Score != 5 {
		t.Errorf("候选 1 期望分数 5，实际 %f", result[0].Score)
	}
	if result[2].Score != 2.5 {
		t.Errorf("候选 3 期望分数 2.5，实际 %f", result[2].Score)
	}
}

// TestLatencyStrategy_Score_NoData 验证无延迟数据时给中间分。
func TestLatencyStrategy_Score_NoData(t *testing.T) {
	s := &LatencyStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withLatency(0)), // 无数据
		newCandidate(2, withLatency(0)), // 无数据
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 5 {
		t.Errorf("无数据候选期望分数 5，实际 %f", result[0].Score)
	}
	if result[1].Score != 5 {
		t.Errorf("无数据候选期望分数 5，实际 %f", result[1].Score)
	}
}

// TestLatencyStrategy_Score_Mixed 验证混合数据场景。
func TestLatencyStrategy_Score_Mixed(t *testing.T) {
	s := &LatencyStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withLatency(100*time.Millisecond)),
		newCandidate(2, withLatency(0)), // 无数据
	}

	result := s.Score(context.Background(), candidates)
	// 候选 1 得 10 分（最低延迟），候选 2 得 5 分（中间分）
	if result[0].Score != 10 {
		t.Errorf("有数据候选期望分数 10，实际 %f", result[0].Score)
	}
	if result[1].Score != 5 {
		t.Errorf("无数据候选期望分数 5，实际 %f", result[1].Score)
	}
}

// ── ErrorStrategy 测试 ─────────────────────────────────────────────────────

// TestErrorStrategy_Filter 验证错误率策略的过滤行为。
func TestErrorStrategy_Filter(t *testing.T) {
	s := &ErrorStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withErrorRate(0.0)),
		newCandidate(2, withErrorRate(1.0)), // 100% 错误率
		newCandidate(3, withErrorRate(0.5)),
	}

	result := s.Filter(context.Background(), candidates)
	if !result[1].Eliminated {
		t.Errorf("100%% 错误率候选应被剔除")
	}
	if result[0].Eliminated || result[2].Eliminated {
		t.Errorf("非 100%% 错误率候选不应被剔除")
	}
}

// TestErrorStrategy_Score 验证错误率策略的评分行为。
func TestErrorStrategy_Score(t *testing.T) {
	s := &ErrorStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withErrorRate(0.0)),
		newCandidate(2, withErrorRate(0.5)),
		newCandidate(3, withErrorRate(1.0)),
	}

	result := s.Score(context.Background(), candidates)
	// 0% 错误率：10 分，50%：5 分，100%：0 分
	if result[0].Score != 10 {
		t.Errorf("0%% 错误率期望分数 10，实际 %f", result[0].Score)
	}
	if result[1].Score != 5 {
		t.Errorf("50%% 错误率期望分数 5，实际 %f", result[1].Score)
	}
	if result[2].Score != 0 {
		t.Errorf("100%% 错误率期望分数 0，实际 %f", result[2].Score)
	}
}

// ── RateStrategy 测试 ──────────────────────────────────────────────────────

// TestRateStrategy_Score_Limited 验证限流策略对限流候选的扣分。
func TestRateStrategy_Score_Limited(t *testing.T) {
	s := &RateStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("rate_limited", true)),
		newCandidate(2),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != -8 {
		t.Errorf("限流候选期望分数 -8，实际 %f", result[0].Score)
	}
	if result[1].Score != 0 {
		t.Errorf("非限流候选期望分数 0，实际 %f", result[1].Score)
	}
}

// TestRateStrategy_Score_Pressure 验证限流策略对压力候选的扣分。
func TestRateStrategy_Score_Pressure(t *testing.T) {
	s := &RateStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("rate_pressure", 0.5)),
		newCandidate(2),
	}

	result := s.Score(context.Background(), candidates)
	// 压力 0.5，扣分 0.5 * 5 = 2.5
	if result[0].Score != -2.5 {
		t.Errorf("压力候选期望分数 -2.5，实际 %f", result[0].Score)
	}
}

// TestRateStrategy_Score_OutOfRange 验证压力值超出范围时被截断。
func TestRateStrategy_Score_OutOfRange(t *testing.T) {
	s := &RateStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("rate_pressure", 1.5)), // 超过 1
		newCandidate(2, withMetadata("rate_pressure", -0.5)), // 小于 0
	}

	result := s.Score(context.Background(), candidates)
	// 1.5 截断为 1，扣 5 分；-0.5 截断为 0，不扣分
	if result[0].Score != -5 {
		t.Errorf("压力超限候选期望分数 -5，实际 %f", result[0].Score)
	}
	if result[1].Score != 0 {
		t.Errorf("负压力候选期望分数 0，实际 %f", result[1].Score)
	}
}

// TestRateStrategy_Eliminated 验证已剔除候选不参与评分。
func TestRateStrategy_Eliminated(t *testing.T) {
	s := &RateStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("rate_limited", true), withEliminated("已剔除")),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("已剔除候选期望分数 0，实际 %f", result[0].Score)
	}
}

// ── TagStrategy 测试 ───────────────────────────────────────────────────────

// TestTagStrategy_Filter_Match 验证标签策略的过滤——匹配时放行。
func TestTagStrategy_Filter_Match(t *testing.T) {
	s := &TagStrategy{}
	ctx := context.WithValue(context.Background(), CtxKeyBusinessTag, "premium")
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("tag", "premium")),
		newCandidate(2, withMetadata("tag", "standard")),
	}

	result := s.Filter(ctx, candidates)
	if result[0].Eliminated {
		t.Errorf("标签匹配候选不应被剔除")
	}
	if !result[1].Eliminated {
		t.Errorf("标签不匹配候选应被剔除")
	}
}

// TestTagStrategy_Filter_Wildcard 验证标签策略的过滤——通配符放行。
func TestTagStrategy_Filter_Wildcard(t *testing.T) {
	s := &TagStrategy{}
	ctx := context.WithValue(context.Background(), CtxKeyBusinessTag, "premium")
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("tag", "*")),
	}

	result := s.Filter(ctx, candidates)
	if result[0].Eliminated {
		t.Errorf("通配符标签候选不应被剔除")
	}
}

// TestTagStrategy_Filter_NoTag 验证无标签上下文时不做过滤。
func TestTagStrategy_Filter_NoTag(t *testing.T) {
	s := &TagStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("tag", "standard")),
	}

	result := s.Filter(context.Background(), candidates)
	if result[0].Eliminated {
		t.Errorf("无标签上下文时不应做过滤")
	}
}

// TestTagStrategy_Score 验证标签策略的评分。
func TestTagStrategy_Score(t *testing.T) {
	s := &TagStrategy{}
	ctx := context.WithValue(context.Background(), CtxKeyBusinessTag, "premium")
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("tag", "premium")),
		newCandidate(2, withMetadata("tag", "standard")),
	}

	result := s.Score(ctx, candidates)
	if result[0].Score != 7 {
		t.Errorf("标签匹配候选期望分数 7，实际 %f", result[0].Score)
	}
	if result[1].Score != 0 {
		t.Errorf("标签不匹配候选期望分数 0，实际 %f", result[1].Score)
	}
}

// ── ComplianceStrategy 测试 ────────────────────────────────────────────────

// TestComplianceStrategy_Filter_InternalOnly 验证合规策略阻断 INTERNAL_ONLY 的外网请求。
func TestComplianceStrategy_Filter_InternalOnly(t *testing.T) {
	s := &ComplianceStrategy{}
	ctx := context.WithValue(context.Background(), CtxKeyNetworkClass, "INTERNAL_ONLY")
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("network_class", "internal")),
		newCandidate(2, withMetadata("network_class", "external")),
	}

	result := s.Filter(ctx, candidates)
	if result[0].Eliminated {
		t.Errorf("内网候选不应被剔除")
	}
	if !result[1].Eliminated {
		t.Errorf("外网候选应被剔除")
	}
}

// TestComplianceStrategy_Filter_NonInternal 验证非 INTERNAL_ONLY 不做过滤。
func TestComplianceStrategy_Filter_NonInternal(t *testing.T) {
	s := &ComplianceStrategy{}
	ctx := context.WithValue(context.Background(), CtxKeyNetworkClass, "HYBRID_ALLOWED")
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("network_class", "external")),
	}

	result := s.Filter(ctx, candidates)
	if result[0].Eliminated {
		t.Errorf("非 INTERNAL_ONLY 时不应剔除外网候选")
	}
}

// TestComplianceStrategy_Filter_NoContext 验无上下文时不做过滤。
func TestComplianceStrategy_Filter_NoContext(t *testing.T) {
	s := &ComplianceStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("network_class", "external")),
	}

	result := s.Filter(context.Background(), candidates)
	if result[0].Eliminated {
		t.Errorf("无上下文时不应剔除候选")
	}
}

// TestComplianceStrategy_Score 验证合规策略不做打分。
func TestComplianceStrategy_Score(t *testing.T) {
	s := &ComplianceStrategy{}
	candidates := []routing.Candidate{newCandidate(1)}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("合规策略期望分数 0，实际 %f", result[0].Score)
	}
}

// ── CacheStrategy 测试 ─────────────────────────────────────────────────────

// TestCacheStrategy_Score 验证缓存策略的扣分行为。
func TestCacheStrategy_Score(t *testing.T) {
	s := &CacheStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("is_cache", true)),
		newCandidate(2),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != -10 {
		t.Errorf("缓存候选期望分数 -10，实际 %f", result[0].Score)
	}
	if result[1].Score != 0 {
		t.Errorf("非缓存候选期望分数 0，实际 %f", result[1].Score)
	}
}

// TestCacheStrategy_Score_Eliminated 验证已剔除缓存候选不扣分。
func TestCacheStrategy_Score_Eliminated(t *testing.T) {
	s := &CacheStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("is_cache", true), withEliminated("已剔除")),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("已剔除候选期望分数 0，实际 %f", result[0].Score)
	}
}

// ── ClassifyStrategy 测试 ──────────────────────────────────────────────────

// TestClassifyStrategy_Score 验证分类策略的评分行为。
func TestClassifyStrategy_Score(t *testing.T) {
	s := &ClassifyStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("tier", "simple")),
		newCandidate(2, withMetadata("tier", "complex")),
		newCandidate(3), // 未分类
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 10 {
		t.Errorf("simple 候选期望分数 10，实际 %f", result[0].Score)
	}
	if result[1].Score != -5 {
		t.Errorf("complex 候选期望分数 -5，实际 %f", result[1].Score)
	}
	if result[2].Score != 0 {
		t.Errorf("未分类候选期望分数 0，实际 %f", result[2].Score)
	}
}

// TestClassifyStrategy_Score_Eliminated 验证已剔除候选不参与评分。
func TestClassifyStrategy_Score_Eliminated(t *testing.T) {
	s := &ClassifyStrategy{}
	candidates := []routing.Candidate{
		newCandidate(1, withMetadata("tier", "simple"), withEliminated("已剔除")),
	}

	result := s.Score(context.Background(), candidates)
	if result[0].Score != 0 {
		t.Errorf("已剔除候选期望分数 0，实际 %f", result[0].Score)
	}
}

// ── RegisterAll 测试 ───────────────────────────────────────────────────────

// TestRegisterAll 验证 RegisterAll 不崩溃。
func TestRegisterAll(t *testing.T) {
	// 此测试仅验证注册函数不崩溃，实际注册由 routing 包的 registry 管理。
	RegisterAll()
}

// ── 空输入边界测试 ─────────────────────────────────────────────────────────

// TestEmptyCandidates 验证所有策略在空候选列表下不崩溃。
func TestEmptyCandidates(t *testing.T) {
	empty := []routing.Candidate{}
	ctx := context.Background()

	strategies := []struct {
		name string
		f    func(context.Context, []routing.Candidate) []routing.Candidate
		s    func(context.Context, []routing.Candidate) []routing.Candidate
	}{
		{"Priority", (&PriorityStrategy{}).Filter, (&PriorityStrategy{}).Score},
		{"Health", (&HealthStrategy{}).Filter, (&HealthStrategy{}).Score},
		{"Weight", (&WeightStrategy{}).Filter, (&WeightStrategy{}).Score},
		{"Affinity", (&AffinityStrategy{}).Filter, (&AffinityStrategy{}).Score},
		{"Cost", (&CostStrategy{}).Filter, (&CostStrategy{}).Score},
		{"Latency", (&LatencyStrategy{}).Filter, (&LatencyStrategy{}).Score},
		{"Error", (&ErrorStrategy{}).Filter, (&ErrorStrategy{}).Score},
		{"Rate", (&RateStrategy{}).Filter, (&RateStrategy{}).Score},
		{"Tag", (&TagStrategy{}).Filter, (&TagStrategy{}).Score},
		{"Compliance", (&ComplianceStrategy{}).Filter, (&ComplianceStrategy{}).Score},
		{"Cache", (&CacheStrategy{}).Filter, (&CacheStrategy{}).Score},
		{"Classify", (&ClassifyStrategy{}).Filter, (&ClassifyStrategy{}).Score},
	}

	for _, st := range strategies {
		t.Run(st.name+"_Filter", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Filter 空候选列表崩溃: %v", r)
				}
			}()
			st.f(ctx, empty)
		})
		t.Run(st.name+"_Score", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Score 空候选列表崩溃: %v", r)
				}
			}()
			st.s(ctx, empty)
		})
	}
}

// TestNilCandidates 验证所有策略在 nil 候选列表下不崩溃。
func TestNilCandidates(t *testing.T) {
	ctx := context.Background()

	strategies := []struct {
		name string
		f    func(context.Context, []routing.Candidate) []routing.Candidate
		s    func(context.Context, []routing.Candidate) []routing.Candidate
	}{
		{"Priority", (&PriorityStrategy{}).Filter, (&PriorityStrategy{}).Score},
		{"Health", (&HealthStrategy{}).Filter, (&HealthStrategy{}).Score},
		{"Weight", (&WeightStrategy{}).Filter, (&WeightStrategy{}).Score},
		{"Affinity", (&AffinityStrategy{}).Filter, (&AffinityStrategy{}).Score},
		{"Cost", (&CostStrategy{}).Filter, (&CostStrategy{}).Score},
		{"Latency", (&LatencyStrategy{}).Filter, (&LatencyStrategy{}).Score},
		{"Error", (&ErrorStrategy{}).Filter, (&ErrorStrategy{}).Score},
		{"Rate", (&RateStrategy{}).Filter, (&RateStrategy{}).Score},
		{"Tag", (&TagStrategy{}).Filter, (&TagStrategy{}).Score},
		{"Compliance", (&ComplianceStrategy{}).Filter, (&ComplianceStrategy{}).Score},
		{"Cache", (&CacheStrategy{}).Filter, (&CacheStrategy{}).Score},
		{"Classify", (&ClassifyStrategy{}).Filter, (&ClassifyStrategy{}).Score},
	}

	for _, st := range strategies {
		t.Run(st.name+"_Filter", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Filter nil 候选列表崩溃: %v", r)
				}
			}()
			st.f(ctx, nil)
		})
		t.Run(st.name+"_Score", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Score nil 候选列表崩溃: %v", r)
				}
			}()
			st.s(ctx, nil)
		})
	}
}
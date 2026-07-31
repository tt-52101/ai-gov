package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// ComplianceStrategy 实现 S-COMPLIANCE 合规网络策略。
//
// 此策略为硬策略（不可关闭），功能如下：
//   - Filter: 检查每个候选的 Metadata["network_class"] 字段，
//     若请求方标记为 INTERNAL_ONLY 而候选为 external，则剔除该候选。
//   - Score: 不做打分（返回原列表）。
//
// 规则依据：D-CON-02（数据不出境定理）——INTERNAL_ONLY 主体的请求
// 不得产生任何外网上游流量。
type ComplianceStrategy struct{}

// ID 返回策略代码 "S-COMPLIANCE"。
func (s *ComplianceStrategy) ID() string { return routing.StrategyCompliance }

// Filter 剔除不符合合规要求的候选。
//
// 从上下文中读取请求方的网络分类标签（key: "request_network_class"），
// 若为 "internal_only"，则剔除所有 Metadata["network_class"] 为 "external" 的候选。
// 若上下文中无网络分类标签，不做过滤（保守放行）。
func (s *ComplianceStrategy) Filter(ctx context.Context, candidates []routing.Candidate) []routing.Candidate {
	reqClass, _ := ctx.Value(CtxKeyNetworkClass).(string)
	if reqClass != "internal_only" {
		return candidates
	}

	result := make([]routing.Candidate, 0, len(candidates))
	for _, c := range candidates {
		nc, _ := c.Metadata["network_class"].(string)
		if nc == "external" {
			c.Eliminated = true
			c.ElimReason = "S-COMPLIANCE: INTERNAL_ONLY 请求不可路由到外网上游"
			result = append(result, c)
			continue
		}
		result = append(result, c)
	}
	return result
}

// Score 合规策略不做打分。
func (s *ComplianceStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// ── 上下文键常量 ──────────────────────────────────────────────────────────

// CtxKeyNetworkClass 用于在上下文中传递请求方的网络分类标签。
// 取值为 "internal_only" 时触发 S-COMPLIANCE 硬过滤。
const CtxKeyNetworkClass = ctxKey("network_class")

// CtxKeyBusinessTag 用于在上下文中传递请求的业务标签，供 S-TAG 使用。
const CtxKeyBusinessTag = ctxKey("business_tag")

// ctxKey 定义上下文键类型，避免与其他包的键冲突。
type ctxKey string

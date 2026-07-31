package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// TagStrategy 实现 S-TAG 业务标签策略。
//
// 按请求的业务标签定向路由到匹配的渠道：
//   - Filter: 若上下文中存在业务标签（key: "business_tag"），
//     剔除 Metadata["tag"] 不匹配的候选。
//   - Score: 对标签匹配的候选加 7 分。
//
// 业务标签仅用于归因和路由决策，不替代鉴权与扣费账户。
type TagStrategy struct{}

// ID 返回策略代码 "S-TAG"。
func (s *TagStrategy) ID() string { return routing.StrategyTag }

// Filter 按业务标签过滤——剔除标签不匹配的候选。
//
// 从上下文中读取请求的业务标签（key: "business_tag"）：
//   - 若上下文中无标签，不做过滤（保守放行）。
//   - 若候选的 Metadata["tag"] 与请求标签不匹配，则剔除。
func (s *TagStrategy) Filter(ctx context.Context, candidates []routing.Candidate) []routing.Candidate {
	reqTag, _ := ctx.Value(CtxKeyBusinessTag).(string)
	if reqTag == "" {
		return candidates
	}

	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		candTag, _ := candidates[i].Metadata["tag"].(string)
		if candTag != reqTag && candTag != "*" {
			candidates[i].Eliminated = true
			candidates[i].ElimReason = "S-TAG: 业务标签不匹配（请求=" + reqTag + "，候选=" + candTag + "）"
		}
	}
	return candidates
}

// Score 对标签匹配的候选加分。
//
// 若上下文中存在业务标签且候选的 Metadata["tag"] 匹配，加 7 分。
func (s *TagStrategy) Score(ctx context.Context, candidates []routing.Candidate) []routing.Candidate {
	reqTag, _ := ctx.Value(CtxKeyBusinessTag).(string)
	if reqTag == "" {
		return candidates
	}

	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		candTag, _ := candidates[i].Metadata["tag"].(string)
		if candTag == reqTag || candTag == "*" {
			candidates[i].Score += 7
		}
	}
	return candidates
}


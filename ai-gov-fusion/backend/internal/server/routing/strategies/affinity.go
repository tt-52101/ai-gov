package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// AffinityStrategy 实现 S-AFFINITY 会话亲和策略。
//
// 同一会话内的后续请求优先路由到之前使用过的渠道：
//   - Filter: 不做过滤。
//   - Score: 检查候选的 Metadata["affinity_hit"] 标记——
//     若为 true（该候选是当前会话此前使用的渠道），加 8 分。
//
// 会话标识通常由路由引擎在首次请求时写入候选元数据，后续请求通过
// 上下文中携带的会话 ID 匹配。
type AffinityStrategy struct{}

// ID 返回策略代码 "S-AFFINITY"。
func (s *AffinityStrategy) ID() string { return routing.StrategyAffinity }

// Filter 会话亲和策略不做过滤。
func (s *AffinityStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// Score 对会话亲和命中候选加分。
//
// 检查每个候选的 Metadata["affinity_hit"] 标记：
//   - true：该候选是当前会话此前使用的渠道，加 8 分以提升优先级。
//   - false 或不存在：不加分。
func (s *AffinityStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		if hit, ok := candidates[i].Metadata["affinity_hit"].(bool); ok && hit {
			candidates[i].Score += 8
		}
	}
	return candidates
}

package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// CacheStrategy 实现 S-CACHE 缓存兜底策略。
//
// 作为降级手段，仅在其他所有候选均被剔除时使用缓存候选：
//   - Filter: 不做过滤。
//   - Score: 标记为缓存渠道的候选扣分 -10——正常情况下不选用，
//     只有在其它候选全部被剔除时才会因唯一剩余而被选中。
//
// 此策略的设计意图是：当全部上游渠道不可用时，回退到本地缓存
// （如 Redis/内存缓存中存储的历史响应）提供兜底服务。
type CacheStrategy struct{}

// ID 返回策略代码 "S-CACHE"。
func (s *CacheStrategy) ID() string { return routing.StrategyCache }

// Filter 缓存策略不做硬过滤。
func (s *CacheStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// Score 对缓存候选大幅扣分——使其作为兜底选择。
//
// 检查每个候选的 Metadata["is_cache"] 标记：
//   - true：缓存候选，扣 10 分——正常情况下不被选中。
//   - false 或不存在：正常候选，不加分也不扣分。
//
// 只有当所有非缓存候选均被其他策略剔除后，缓存候选才会胜出。
func (s *CacheStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		if isCache, ok := candidates[i].Metadata["is_cache"].(bool); ok && isCache {
			candidates[i].Score -= 10
		}
	}
	return candidates
}

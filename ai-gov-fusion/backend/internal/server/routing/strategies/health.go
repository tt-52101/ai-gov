package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// HealthStrategy 实现 S-HEALTH 健康与熔断策略。
//
// 基于三态健康模型（up / degraded / down）进行路由决策：
//   - Filter: down 状态的候选直接剔除。
//   - Score: degraded 状态的候选扣分（-5 分），up 状态不加分。
//
// 熔断器三态：
//   - Closed（正常运行）→ Health="up"
//   - HalfOpen（探测恢复中）→ Health="degraded"
//   - Open（熔断器打开）→ Health="down"
type HealthStrategy struct{}

// ID 返回策略代码 "S-HEALTH"。
func (s *HealthStrategy) ID() string { return routing.StrategyHealth }

// Filter 剔除健康状态为 down 的候选。
//
// down 状态的候选已被熔断器标记为不可用，应从候选集中直接移除。
func (s *HealthStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	result := make([]routing.Candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.Health == routing.HealthDown {
			c.Eliminated = true
			c.ElimReason = "S-HEALTH: 渠道健康状态为 down，已剔除"
			result = append(result, c)
			continue
		}
		result = append(result, c)
	}
	return result
}

// Score 对降级候选进行扣分。
//
// up = 0 分（正常），degraded = -5 分（警告，但可用）。
func (s *HealthStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if candidates[i].Health == routing.HealthDegraded {
			candidates[i].Score -= 5
		}
	}
	return candidates
}

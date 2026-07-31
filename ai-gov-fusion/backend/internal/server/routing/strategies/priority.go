package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// PriorityStrategy 实现 S-PRI 优先级分组策略。
//
// 按候选的 Priority 字段分组，高优先级组未耗尽前不使用低优先级候选：
//   - Filter: 找出最高非空优先级组，剔除所有低于该优先级的候选。
//   - Score: 按 Priority 值直接加分（每个优先级等级 +1 分）。
type PriorityStrategy struct{}

// ID 返回策略代码 "S-PRI"。
func (s *PriorityStrategy) ID() string { return routing.StrategyPriority }

// Filter 按优先级分组——只保留最高优先级的未剔除候选。
//
// 高优先级组未耗尽时不使用低优先级候选：
// 找出所有未剔除候选中的最高 Priority 值，剔除所有低于该值的候选。
func (s *PriorityStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	// 找出未剔除候选中的最高优先级。
	maxPri := -1
	for _, c := range candidates {
		if !c.Eliminated && c.Priority > maxPri {
			maxPri = c.Priority
		}
	}

	if maxPri < 0 {
		return candidates // 无未剔除候选。
	}

	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		if candidates[i].Priority < maxPri {
			candidates[i].Eliminated = true
			candidates[i].ElimReason = "S-PRI: 优先级低于最高组，已被剔除"
		}
	}
	return candidates
}

// Score 按优先级加分——每个优先级等级 +1 分。
func (s *PriorityStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if !candidates[i].Eliminated {
			candidates[i].Score += float64(candidates[i].Priority)
		}
	}
	return candidates
}

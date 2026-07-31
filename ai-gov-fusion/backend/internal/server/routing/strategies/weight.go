package strategies

import (
	"context"
	"math"

	"tokenhub/backend/internal/server/routing"
)

// WeightStrategy 实现 S-WEIGHT 权重与负载策略。
//
// 按候选的 Weight 字段进行概率型打分，权重越高的候选得分越高：
//   - Filter: 不做过滤。
//   - Score: 将 Weight 归一化后作为分数（Weight / 总权重 × 10）。
//     权重越高的候选获得更高的概率型分数。
type WeightStrategy struct{}

// ID 返回策略代码 "S-WEIGHT"。
func (s *WeightStrategy) ID() string { return routing.StrategyWeight }

// Filter 权重策略不做过滤。
func (s *WeightStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// Score 按权重概率分配分数。
//
// 计算方式：将所有未剔除候选的 Weight 归一化到 0~10 区间后作为分数。
// 公式：score = (weight / totalWeight) × 10
func (s *WeightStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	totalWeight := 0.0
	eligible := make([]*routing.Candidate, 0)
	for i := range candidates {
		if !candidates[i].Eliminated {
			totalWeight += candidates[i].Weight
			eligible = append(eligible, &candidates[i])
		}
	}

	if totalWeight <= 0 || len(eligible) == 0 {
		return candidates
	}

	for _, c := range eligible {
		score := (c.Weight / totalWeight) * 10.0
		// 对极端小权重做下限保护。
		if math.IsNaN(score) {
			score = 0
		}
		c.Score += score
	}
	return candidates
}

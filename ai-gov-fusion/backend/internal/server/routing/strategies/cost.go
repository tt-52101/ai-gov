package strategies

import (
	"context"
	"sort"

	"tokenhub/backend/internal/server/routing"

	"github.com/shopspring/decimal"
)

// CostStrategy 实现 S-COST 成本感知策略。
//
// 按候选的 EstSell（预估内部结算价）评分，价格越低得分越高：
//   - Filter: 不做过滤。
//   - Score: 将 EstSell 映射为分数——最低价的候选得满分 10 分，
//     其余候选按与最低价的比例折算。
type CostStrategy struct{}

// ID 返回策略代码 "S-COST"。
func (s *CostStrategy) ID() string { return routing.StrategyCost }

// Filter 成本策略不做过滤（价格帽过滤在管道前置阶段完成）。
func (s *CostStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// Score 按预估内部价打分——价格越低分数越高。
//
// 计算方式：
//   - 找出未剔除候选中的最低 EstSell。
//   - 最低价候选 = +10 分。
//   - 其他候选 = +10 × (最低价 / 当前价)。
//   - 若所有 EstSell 为零，均得 10 分。
func (s *CostStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	eligible := filterEligible(candidates)
	if len(eligible) == 0 {
		return candidates
	}

	// 收集非零 EstSell 值并排序求最低价。
	prices := make([]decimal.Decimal, 0, len(eligible))
	for _, idx := range eligible {
		if candidates[idx].EstSell.IsPositive() {
			prices = append(prices, candidates[idx].EstSell)
		}
	}

	if len(prices) == 0 {
		// 所有 EstSell = 0，无法比较，均得满分。
		for _, idx := range eligible {
			candidates[idx].Score += 10
		}
		return candidates
	}

	sort.Slice(prices, func(i, j int) bool {
		return prices[i].LessThan(prices[j])
	})
	minPrice := prices[0]

	for _, idx := range eligible {
		if candidates[idx].EstSell.IsZero() || !candidates[idx].EstSell.IsPositive() {
			candidates[idx].Score += 10
			continue
		}
		ratio := minPrice.Div(candidates[idx].EstSell)
		score, _ := ratio.Float64()
		candidates[idx].Score += score * 10
	}
	return candidates
}

// filterEligible 返回未剔除候选的索引列表。
func filterEligible(candidates []routing.Candidate) []int {
	eligible := make([]int, 0, len(candidates))
	for i := range candidates {
		if !candidates[i].Eliminated {
			eligible = append(eligible, i)
		}
	}
	return eligible
}

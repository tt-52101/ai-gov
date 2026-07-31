package strategies

import (
	"context"
	"time"

	"tokenhub/backend/internal/server/routing"
)

// LatencyStrategy 实现 S-LATENCY 延迟感知策略。
//
// 基于候选的 EWMA（指数加权移动平均）延迟进行评分，延迟越低得分越高：
//   - Filter: 不做过滤。
//   - Score: 找出最低延迟候选得满分 10 分，其余按比例折算。
//     对零延迟候选（未采集到延迟数据）给予中位分数 5 分。
//
// 延迟数据通常由渠道探针（channel_probes）定期采集后写入候选的 LatencyEWMA 字段。
type LatencyStrategy struct{}

// ID 返回策略代码 "S-LATENCY"。
func (s *LatencyStrategy) ID() string { return routing.StrategyLatency }

// Filter 延迟策略不做过滤。
func (s *LatencyStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// Score 按 EWMA 延迟打分——延迟越低分数越高。
//
// 计算方式：
//   - 找出所有非零延迟中的最小值。
//   - 最低延迟候选 = +10 分。
//   - 其他候选 = +10 × (最低延迟 / 当前延迟)。
//   - 零延迟候选（无数据）= +5 分（中间值，不偏好也不惩罚）。
func (s *LatencyStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	eligible := filterEligible(candidates)
	if len(eligible) == 0 {
		return candidates
	}

	// 找出非零最小延迟。
	var minLatency time.Duration
	for _, idx := range eligible {
		d := candidates[idx].LatencyEWMA
		if d > 0 && (minLatency == 0 || d < minLatency) {
			minLatency = d
		}
	}

	if minLatency == 0 {
		// 所有候选均无延迟数据，全部给 5 分。
		for _, idx := range eligible {
			candidates[idx].Score += 5
		}
		return candidates
	}

	for _, idx := range eligible {
		d := candidates[idx].LatencyEWMA
		if d <= 0 {
			candidates[idx].Score += 5 // 无数据，中间分。
			continue
		}
		ratio := float64(minLatency) / float64(d)
		candidates[idx].Score += ratio * 10
	}
	return candidates
}

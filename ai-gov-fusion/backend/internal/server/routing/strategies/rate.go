package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// RateStrategy 实现 S-RATE 限流感知策略。
//
// 对最近触发 HTTP 429（限流）或接近 RPM/TPM 上限的候选降权：
//   - Filter: 不做过滤。
//   - Score: 检查候选的 Metadata["rate_limited"] 标记：
//     若标记为 true（近期触发过 429），扣分 -8 分；
//     若 Metadata["rate_pressure"] 存在（接近限流阈值），按压力比例扣分（最多 -5）。
//
// 限流数据通常由渠道探针或上游响应中提取后写入候选元数据。
type RateStrategy struct{}

// ID 返回策略代码 "S-RATE"。
func (s *RateStrategy) ID() string { return routing.StrategyRate }

// Filter 限流策略不做硬过滤。
func (s *RateStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// Score 对限流风险候选降权。
//
// 惩罚规则：
//   - Metadata["rate_limited"] == true：近期触发过 429，扣 8 分。
//   - Metadata["rate_pressure"] 存在：接近限流阈值，按比例扣分（0~5 分）。
//     "rate_pressure" 取值为 0.0~1.0（0=无压力，1=即将触发限流）。
func (s *RateStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}

		// 检查是否近期触发过限流。
		if limited, ok := candidates[i].Metadata["rate_limited"].(bool); ok && limited {
			candidates[i].Score -= 8
			continue
		}

		// 检查限流压力等级。
		pressure := ratePressure(candidates[i].Metadata)
		if pressure > 0 {
			candidates[i].Score -= pressure * 5
		}
	}
	return candidates
}

// ratePressure 从元数据中提取限流压力等级（0.0~1.0）。
func ratePressure(meta map[string]any) float64 {
	if meta == nil {
		return 0
	}
	switch v := meta["rate_pressure"].(type) {
	case float64:
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	case int:
		if v <= 0 {
			return 0
		}
		if v >= 1 {
			return 1
		}
		return float64(v)
	default:
		return 0
	}
}

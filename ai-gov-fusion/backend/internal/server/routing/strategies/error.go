package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// ErrorStrategy 实现 S-ERROR 错误率感知策略。
//
// 基于候选的近期错误率进行评分，错误率越低得分越高：
//   - Filter: 错误率 >= 1.0（100%）的候选直接剔除。
//   - Score: 错误率越低分数越高——0% 错误率得 10 分，
//     随错误率线性递减，100% 得 0 分。
type ErrorStrategy struct{}

// ID 返回策略代码 "S-ERROR"。
func (s *ErrorStrategy) ID() string { return routing.StrategyError }

// Filter 剔除错误率达 100% 的候选。
func (s *ErrorStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		if candidates[i].ErrorRate >= 1.0 {
			candidates[i].Eliminated = true
			candidates[i].ElimReason = "S-ERROR: 错误率 100%，已剔除"
		}
	}
	return candidates
}

// Score 按错误率打分——错误率越低分数越高。
//
// 公式：score = (1 - errorRate) × 10
//   - 0% 错误率 = 10 分。
//   - 50% 错误率 = 5 分。
//   - 100% 错误率 = 0 分。
func (s *ErrorStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		candidates[i].Score += (1.0 - candidates[i].ErrorRate) * 10
	}
	return candidates
}

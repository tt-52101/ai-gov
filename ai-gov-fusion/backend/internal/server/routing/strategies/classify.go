package strategies

import (
	"context"

	"tokenhub/backend/internal/server/routing"
)

// ClassifyStrategy 实现 S-CLASSIFY 智能分类策略。
//
// 使用轻量级模型预判任务复杂度，将简单任务自动路由到低成本模型：
//   - Filter: 不做过滤。
//   - Score: 根据候选的 Metadata["tier"] 分类标签打分：
//     * "simple" 意图候选 +10 分（偏好低价模型）。
//     * "complex" 意图候选 -5 分（不鼓励，但也不剔除）。
//
// 分类逻辑（阶段 C 可选）：
//   调用方在进入管道前使用轻量分类模型分析请求内容，判断任务复杂度
//   （如翻译、摘要为 simple；代码生成、推理为 complex），并将分类结果
//   写入候选的 Metadata["tier"] 字段。
//
// 性能考量：此策略本身不做模型推理，依赖调用方预填充的分类结果。
// 若 Metadata["tier"] 不存在，视为未分类（0 分，不影响排序）。
type ClassifyStrategy struct{}

// ID 返回策略代码 "S-CLASSIFY"。
func (s *ClassifyStrategy) ID() string { return routing.StrategyClassify }

// Filter 分类策略不做过滤。
func (s *ClassifyStrategy) Filter(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	return candidates
}

// Score 按任务复杂度分类打分。
//
// 打分规则：
//   - "simple"：任务简单，鼓励使用低成本模型，+10 分。
//   - "complex"：任务复杂，不鼓励降级到简单模型，-5 分。
//   - 未分类（"" 或不存在）：0 分，不影响排序。
func (s *ClassifyStrategy) Score(_ context.Context, candidates []routing.Candidate) []routing.Candidate {
	for i := range candidates {
		if candidates[i].Eliminated {
			continue
		}
		tier, _ := candidates[i].Metadata["tier"].(string)
		switch tier {
		case "simple":
			candidates[i].Score += 10
		case "complex":
			candidates[i].Score -= 5
		}
	}
	return candidates
}

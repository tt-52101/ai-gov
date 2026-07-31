// Package strategies 实现路由引擎的 12 种可插拔策略。
// 每种策略提供 Filter（剔除不合格候选）和 Score（打分排序）两个方法，
// 通过 init() 自动注册到 routing 包的全局策略注册表。
package strategies

import "tokenhub/backend/internal/server/routing"

// ── 策略注册入口 ──────────────────────────────────────────────────────────

// RegisterAll 向全局注册表注册全部 12 种策略。
// 应在应用启动时调用，确保所有策略在路由档案解析前就绪。
func RegisterAll() {
	routing.Register(&ComplianceStrategy{})
	routing.Register(&HealthStrategy{})
	routing.Register(&PriorityStrategy{})
	routing.Register(&WeightStrategy{})
	routing.Register(&CostStrategy{})
	routing.Register(&LatencyStrategy{})
	routing.Register(&ErrorStrategy{})
	routing.Register(&RateStrategy{})
	routing.Register(&AffinityStrategy{})
	routing.Register(&TagStrategy{})
	routing.Register(&CacheStrategy{})
	routing.Register(&ClassifyStrategy{})
}

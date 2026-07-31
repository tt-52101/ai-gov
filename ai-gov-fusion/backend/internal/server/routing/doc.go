// Package routing 实现模型请求的智能路由引擎，核心为 12 种可插拔策略。
//
// 策略管道：S-COMPLIANCE → δ 价格帽 → S-CLASSIFY → 其余策略 → 选取最优候选。
// 每种策略通过全局注册表管理，路由档案通过策略绑定组合多种策略。
package routing

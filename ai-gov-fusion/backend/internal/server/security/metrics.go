package security

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ── 安全指标命名空间 ────────────────────────────────────────────────────────

const (
	metricsNamespace = "tokenhub"
	metricsSubsystem = "security"
)

// SecurityMetrics 安全模块的 Prometheus 指标集合。
//
// 覆盖以下关键路径埋点：
//   - 出网管控：Egress 策略检查次数与阻断次数
//   - 内容安全：Hook 请求/响应检查次数与阻断次数
//   - 异常流量：限流器触发次数
//
// 所有指标通过 promauto 注册到默认 Registry，与 server.GatewayMetrics 隔离。
type SecurityMetrics struct {
	// EgressChecks 出网管控检查总次数（按策略、结果、模型分类）。
	EgressChecks *prometheus.CounterVec
	// EgressBlocked 出网管控阻断总次数（按策略、原因分类）。
	EgressBlocked *prometheus.CounterVec
	// HookChecks 安全钩子检查总次数（按阶段、结果分类）。
	HookChecks *prometheus.CounterVec
	// HookBlocked 安全钩子阻断总次数（按阶段、原因分类）。
	HookBlocked *prometheus.CounterVec
	// RateLimitTriggered 限流器触发总次数（按 key 来源分类）。
	RateLimitTriggered *prometheus.CounterVec
}

// NewSecurityMetrics 创建并注册安全模块的 Prometheus 指标。
func NewSecurityMetrics() *SecurityMetrics {
	return &SecurityMetrics{
		EgressChecks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Subsystem: metricsSubsystem,
				Name:      "egress_checks_total",
				Help:      "出网管控检查总次数，按策略、结果、模型分类",
			},
			[]string{"policy", "result", "model"},
		),
		EgressBlocked: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Subsystem: metricsSubsystem,
				Name:      "egress_blocked_total",
				Help:      "出网管控阻断总次数，按策略、原因分类",
			},
			[]string{"policy", "reason"},
		),
		HookChecks: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Subsystem: metricsSubsystem,
				Name:      "hook_checks_total",
				Help:      "安全钩子检查总次数，按阶段（request/response）、结果（pass/block）分类",
			},
			[]string{"phase", "result"},
		),
		HookBlocked: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Subsystem: metricsSubsystem,
				Name:      "hook_blocked_total",
				Help:      "安全钩子阻断总次数，按阶段、原因分类",
			},
			[]string{"phase", "reason"},
		),
		RateLimitTriggered: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: metricsNamespace,
				Subsystem: metricsSubsystem,
				Name:      "ratelimit_triggered_total",
				Help:      "限流器触发总次数，按来源分类",
			},
			[]string{"source"},
		),
	}
}

// ── 全局默认安全指标实例 ────────────────────────────────────────────────────

// DefaultSecurityMetrics 包级默认安全指标实例，供 CheckEgress 等函数直接使用。
// 若不需要指标上报，可在启动时调用 ResetSecurityMetrics(nil) 禁用。
var DefaultSecurityMetrics = NewSecurityMetrics()

// ResetSecurityMetrics 重置全局安全指标实例。
// 传入 nil 可禁用安全指标上报。
func ResetSecurityMetrics(m *SecurityMetrics) {
	if m == nil {
		DefaultSecurityMetrics = &SecurityMetrics{}
		return
	}
	DefaultSecurityMetrics = m
}

// ── 空指标安全保护 ──────────────────────────────────────────────────────────

// isMetricsEnabled 检查安全指标是否启用（非空实例）。
func isMetricsEnabled() bool {
	return DefaultSecurityMetrics != nil && DefaultSecurityMetrics.EgressChecks != nil
}
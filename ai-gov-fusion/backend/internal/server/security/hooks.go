package security

import (
	"context"
	"log/slog"
)

// ── 钩子上下文类型 ────────────────────────────────────────────────────────

// HookRequest 描述一次进入调度前的请求上下文，供安全钩子在 OnRequest 阶段检查。
//
// 字段稳定后，后续阶段可扩展以携带请求体摘要、内容分类结果、脱敏标记等。
type HookRequest struct {
	// RequestID 全链路唯一请求标识。
	RequestID string

	// UserID 发起调用的用户标识。
	UserID string

	// PartyID 发起调用所属的主体标识。
	PartyID string

	// AccountID 扣费账户标识。
	AccountID string

	// APIKeyID 使用的网关 Key 标识。
	APIKeyID string

	// ModelName 请求的逻辑模型名称。
	ModelName string

	// ClientIP 客户端 IP 地址。
	ClientIP string

	// UserAgent 客户端 User-Agent 请求头。
	UserAgent string

	// NetworkClass 请求的网络分类（internal / external）。
	NetworkClass string

	// DataClassification 请求涉及的数据分类（public / internal / confidential / restricted）。
	DataClassification string

	// RequestPayload 原始请求体（序列化后的 JSON 字符串）。
	// 后续阶段用于内容安全扫描与提示词注入检测。
	RequestPayload string
}

// HookResponse 描述一次上游响应返回后的上下文，供安全钩子在 OnResponse 阶段检查。
type HookResponse struct {
	// RequestID 与原始请求对应的全链路标识。
	RequestID string

	// StatusCode 上游返回的 HTTP 状态码。
	StatusCode int

	// ResponsePayload 上游响应体（序列化后的 JSON 字符串）。
	// 后续阶段用于敏感数据回扫与脱敏验证。
	ResponsePayload string

	// BlockReason 若被阻断，记录阻断原因。
	BlockReason string
}

// ── 钩子接口 ──────────────────────────────────────────────────────────────

// Hook 安全钩子接口——数据面管线中的可插拔安全扩展点（PRD SEC-05，架构级 P0）。
//
// 当前阶段提供空实现（NoopHook），所有请求放行。
// 后续阶段接入：
//   - 内容安全引擎（敏感词检测、有害内容识别）
//   - 提示词注入检测
//   - 敏感数据脱敏（手机号、身份证号、银行卡号等）
//   - 出网管控（INTERNAL_ONLY 强制拦截）
//
// 实现必须是并发安全的——多个 goroutine 可能同时调用 OnRequest 或 OnResponse。
type Hook interface {
	// OnRequest 请求进入调度前执行。若返回非 nil error，请求被阻断且 error
	// 内容作为阻断原因返回给调用方。返回 nil 表示放行。
	//
	// ctx 携带请求级上下文，可用于超时控制和日志串联。
	OnRequest(ctx context.Context, req *HookRequest) error

	// OnResponse 上游响应返回后执行。若返回非 nil error，响应被阻断且不返回给调用方。
	// 返回 nil 表示放行。
	//
	// 注意：对于流式响应，OnResponse 在首个 chunk 到达后即触发，
	// 后续 chunk 不受影响——流式场景的阻断应在 OnRequest 阶段完成。
	OnResponse(ctx context.Context, resp *HookResponse) error
}

// ── 空实现 ────────────────────────────────────────────────────────────────

// NoopHook 空实现的安全钩子——不执行任何检查，所有请求和响应直接放行。
//
// 这是阶段 B 的默认钩子。后续阶段可替换为真实的内容安全钩子或出网管控钩子，
// 通过 Chain.Add 注入到管线中。
type NoopHook struct{}

// OnRequest 空实现——始终返回 nil，放行所有请求。
func (n *NoopHook) OnRequest(_ context.Context, _ *HookRequest) error {
	return nil
}

// OnResponse 空实现——始终返回 nil，放行所有响应。
func (n *NoopHook) OnResponse(_ context.Context, _ *HookResponse) error {
	return nil
}

// 编译期断言 NoopHook 实现了 Hook 接口。
var _ Hook = (*NoopHook)(nil)

// ── 默认内容安全限制 ────────────────────────────────────────────────────────

const (
	// DefaultMaxRequestPayloadBytes 默认请求体最大长度（1MB）。
	DefaultMaxRequestPayloadBytes = 1 << 20 // 1MB
	// DefaultMaxResponsePayloadBytes 默认响应体最大长度（10MB）。
	DefaultMaxResponsePayloadBytes = 10 << 20 // 10MB
)

// ContentSafetyHook 内容安全钩子——提供请求/响应体长度限制检查与结构化日志记录。
//
// 这是 NoopHook 的替代实现，通过在 OnRequest/OnResponse 中执行以下检查
// 来满足内容安全需求（PRD SEC-03）：
//   - 请求体长度检查：超过 MaxRequestPayloadBytes 的请求被阻断。
//   - 响应体长度检查：超过 MaxResponsePayloadBytes 的响应被阻断。
//   - 结构化日志：记录每次请求/响应检查结果，含 request_id 等链路字段。
//
// 后续阶段可在此钩子中接入敏感词检测、提示词注入检测、敏感数据脱敏等能力。
// 实现线程安全——所有字段在构造时设置，运行期间不变。
type ContentSafetyHook struct {
	// MaxRequestPayloadBytes 请求体最大字节数。0 表示不限制。
	MaxRequestPayloadBytes int
	// MaxResponsePayloadBytes 响应体最大字节数。0 表示不限制。
	MaxResponsePayloadBytes int
}

// NewContentSafetyHook 创建内容安全钩子，使用默认长度限制。
func NewContentSafetyHook() *ContentSafetyHook {
	return &ContentSafetyHook{
		MaxRequestPayloadBytes:  DefaultMaxRequestPayloadBytes,
		MaxResponsePayloadBytes: DefaultMaxResponsePayloadBytes,
	}
}

// OnRequest 检查请求体长度是否超过限制。
//
// 检查项：
//   - 若 MaxRequestPayloadBytes > 0 且请求体超过该值，阻断并记录日志。
//   - 始终记录结构化日志（含 request_id、user_id、model 等字段）。
func (h *ContentSafetyHook) OnRequest(ctx context.Context, req *HookRequest) error {
	payloadLen := len(req.RequestPayload)

	// 检查请求体长度限制。
	if h.MaxRequestPayloadBytes > 0 && payloadLen > h.MaxRequestPayloadBytes {
		slog.WarnContext(ctx, "内容安全钩子阻断",
			"request_id", req.RequestID,
			"user_id", req.UserID,
			"model", req.ModelName,
			"reason", "请求体超过长度限制",
			"payload_bytes", payloadLen,
			"max_bytes", h.MaxRequestPayloadBytes,
		)
		return &HookBlockedError{
			HookIndex: 0,
			Reason:    "请求体超过内容安全长度限制",
		}
	}

	slog.DebugContext(ctx, "内容安全钩子放行",
		"request_id", req.RequestID,
		"user_id", req.UserID,
		"model", req.ModelName,
		"payload_bytes", payloadLen,
		"network_class", req.NetworkClass,
		"data_classification", req.DataClassification,
	)
	return nil
}

// OnResponse 检查响应体长度是否超过限制。
//
// 检查项：
//   - 若 MaxResponsePayloadBytes > 0 且响应体超过该值，阻断并记录日志。
//   - 始终记录结构化日志（含 request_id、status_code 等字段）。
func (h *ContentSafetyHook) OnResponse(ctx context.Context, resp *HookResponse) error {
	payloadLen := len(resp.ResponsePayload)

	// 检查响应体长度限制。
	if h.MaxResponsePayloadBytes > 0 && payloadLen > h.MaxResponsePayloadBytes {
		slog.WarnContext(ctx, "内容安全钩子阻断响应",
			"request_id", resp.RequestID,
			"status_code", resp.StatusCode,
			"reason", "响应体超过长度限制",
			"payload_bytes", payloadLen,
			"max_bytes", h.MaxResponsePayloadBytes,
		)
		return &HookBlockedError{
			HookIndex: 0,
			Reason:    "响应体超过内容安全长度限制",
		}
	}

	slog.DebugContext(ctx, "内容安全钩子放行响应",
		"request_id", resp.RequestID,
		"status_code", resp.StatusCode,
		"payload_bytes", payloadLen,
	)
	return nil
}

// 编译期断言 ContentSafetyHook 实现了 Hook 接口。
var _ Hook = (*ContentSafetyHook)(nil)

// ── 钩子链 ────────────────────────────────────────────────────────────────

// Chain 安全钩子链——按注册顺序依次执行多个 Hook。
//
// 执行语义：
//   - OnRequest 阶段：按 Add 顺序依次调用。首个返回 error 的钩子立即阻断，
//     后续钩子不再执行，error 原样返回给调用方。
//   - OnResponse 阶段：按 Add 顺序依次调用。首个返回 error 的钩子立即阻断。
//
// Chain 本身不是并发安全的——应在服务启动时完成注册，运行期间不动态修改。
// 各 Hook 内部必须自行处理并发安全。
type Chain struct {
	hooks []Hook
}

// Add 将钩子追加到链尾。
//
// 若传入 nil，调用将被静默忽略。
func (c *Chain) Add(h Hook) {
	if h != nil {
		c.hooks = append(c.hooks, h)
	}
}

// Execute 依次执行所有已注册钩子的 OnRequest 方法。
// 首个返回 error 的钩子立即终止链路，返回该 error。
// 全部通过则返回 nil。
//
// ctx 会原样传递给每个钩子，用于超时控制和日志串联。
func (c *Chain) Execute(ctx context.Context, req *HookRequest) error {
	for i, h := range c.hooks {
		if err := h.OnRequest(ctx, req); err != nil {
			// 返回包装后的错误，携带钩子索引便于定位。
			return &HookBlockedError{
				HookIndex: i,
				Reason:    err.Error(),
			}
		}
	}
	return nil
}

// ExecuteResponse 依次执行所有已注册钩子的 OnResponse 方法。
// 首个返回 error 的钩子立即终止链路。
func (c *Chain) ExecuteResponse(ctx context.Context, resp *HookResponse) error {
	for i, h := range c.hooks {
		if err := h.OnResponse(ctx, resp); err != nil {
			return &HookBlockedError{
				HookIndex: i,
				Reason:    err.Error(),
			}
		}
	}
	return nil
}

// ── 错误类型 ──────────────────────────────────────────────────────────────

// HookBlockedError 表示请求或响应被安全钩子阻断。
type HookBlockedError struct {
	// HookIndex 阻断钩子在 Chain 中的索引（0 起始）。
	HookIndex int

	// Reason 阻断原因（由钩子返回的错误信息）。
	Reason string
}

// Error 实现 error 接口，返回可读的阻断描述。
func (e *HookBlockedError) Error() string {
	return "security: 钩子 #" + itoa(e.HookIndex) + " 阻断: " + e.Reason
}

// itoa 简单整数转字符串（避免引入 strconv 包）。
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	neg := false
	if i < 0 {
		neg = true
		i = -i
	}
	for i > 0 {
		digits = append([]byte{byte('0' + i%10)}, digits...)
		i /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}

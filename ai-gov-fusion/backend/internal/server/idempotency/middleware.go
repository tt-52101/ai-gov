package idempotency

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Context key 类型——使用未导出类型防止跨包碰撞。
type contextKey struct{}

// ctxKey 是幂等键值的单例 context key。
var ctxKey contextKey

// IdempotencyKeyHeader HTTP 头名称，对齐 IETF 草案标准和 Stripe 约定。
const IdempotencyKeyHeader = "Idempotency-Key"

// MaxKeyLength 幂等键最大允许长度。
// UUID v4 为 36 个字符，但我们允许最多 255 个字符以兼容其他键格式（PRD §8.7）。
const MaxKeyLength = 255

// GetIdempotencyKey 从 context 中提取幂等键。
// 若 Middleware 注入了键则返回键字符串和 true，否则返回空字符串和 false。
func GetIdempotencyKey(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(ctxKey).(string)
	return key, ok
}

// WithIdempotencyKey 将幂等键注入 context。
// 由 Middleware 在验证后使用，也可在测试中使用。
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ctxKey, key)
}

// Middleware 从入站 HTTP 请求中提取 Idempotency-Key 头，若存在则验证其格式，
// 并将其注入请求 context。
//
// 验证规则（PRD §8.7）：
//   - 该头是可选的。不带该头的请求正常通过。
//   - 若存在，值必须 ≤ 255 个字符。
//   - 若存在，值应为 UUID v4 格式（RFC 4122）。非 UUID 值带警告被接受但不拒绝，
//     因为中间件层不拥有幂等语义——服务层拥有。
//   - 头名称按 HTTP/1.1 规范大小写不敏感。
//
// 此中间件不决定哪些端点需要幂等。该职责属于控制器/服务层，
// 后者调用 GetIdempotencyKey 检索值并决定缺少键是否为错误。
//
// 仅 POST、PUT 和 PATCH 请求检查该头。
// GET、DELETE、HEAD、OPTIONS 及其他安全/幂等方法不经检查直接通过。
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 仅检查变更性方法。安全方法（GET、HEAD、OPTIONS）
		// 和 DELETE（按 HTTP 语义是幂等的）不需要幂等键验证。
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			// 检查该头。
		default:
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get(IdempotencyKeyHeader)
		if key == "" {
			// 未提供键。由控制器决定这是否为错误。
			next.ServeHTTP(w, r)
			return
		}

		// 去除空白。HTTP 头值不应有前导/尾部空白，但有些客户端会添加。
		key = strings.TrimSpace(key)

		if err := ValidateKey(key); err != nil {
			// 键格式无效。立即以清晰错误拒绝。
			http.Error(w, fmt.Sprintf(`{"error":{"code":"IDEMPOTENCY_KEY_INVALID","message":%q}}`, err.Error()),
				http.StatusBadRequest)
			return
		}

		// 将已验证的键注入 context 供下游处理器使用。
		ctx := WithIdempotencyKey(r.Context(), key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateKey 检查幂等键是否符合所需格式。
//
// 规则：
//   - 不得为空。
//   - 不得超过 MaxKeyLength（255）个字符。
//   - 应为有效的 UUID v4（RFC 4122）。在此验证层拒绝非 UUID 键，
//     因为随机 UUID 对于确保最多执行一次保证有意义是必需的。
//     短或可预测的键增加了意外碰撞的风险。
//
// 若键有效则返回 nil，若验证失败则返回描述性错误。
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("幂等键为空")
	}

	if len(key) > MaxKeyLength {
		return fmt.Errorf("幂等键超过最大长度 %d 个字符", MaxKeyLength)
	}

	// 验证 UUID v4 格式：8-4-4-4-12 十六进制数字加连字符。
	// 格式：xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
	if !isValidUUIDv4(key) {
		return fmt.Errorf("幂等键必须是有效的 UUID v4（RFC 4122）")
	}

	return nil
}

// isValidUUIDv4 检查字符串是否为有效的 UUID v4（RFC 4122）。
// 期望格式为：xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
// 其中 x 为小写十六进制数字。
//
// 这是一个快速、无分配的检查，验证：
//   - 精确长度 36 个字符。
//   - 位置 8、13、18、23 处有连字符。
//   - 版本半字节为 4。
//   - 变体半字节为 8、9、a 或 b。
//   - 所有其他字符为有效十六进制数字。
func isValidUUIDv4(s string) bool {
	if len(s) != 36 {
		return false
	}

	// 检查连字符位置。
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}

	// 检查版本半字节（位置 14 必须为 '4'）。
	if s[14] != '4' {
		return false
	}

	// 检查变体半字节（位置 19 必须为 8、9、a 或 b）。
	switch s[19] {
	case '8', '9', 'a', 'b', 'A', 'B':
		// 有效变体。
	default:
		return false
	}

	// 验证所有剩余字符为十六进制数字。
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // 已检查过连字符
		}
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

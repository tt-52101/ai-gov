package idempotency

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Context key 类型——使用未导出类型防止跨包碰撞。
type contextKey struct{}

// ctxKey is the singleton context key for the idempotency key value.
var ctxKey contextKey

// IdempotencyKeyHeader HTTP 头名称，对齐 IETF 草案标准和 Stripe 约定。
const IdempotencyKeyHeader = "Idempotency-Key"

// MaxKeyLength 幂等键最大允许长度。
// UUID v4 is 36 characters, but we allow up to 255 to accommodate
// alternative key formats (PRD §8.7).
const MaxKeyLength = 255

// GetIdempotencyKey extracts the idempotency key from a context.
// Returns the key string and true if a key was injected by Middleware,
// or an empty string and false otherwise.
func GetIdempotencyKey(ctx context.Context) (string, bool) {
	key, ok := ctx.Value(ctxKey).(string)
	return key, ok
}

// WithIdempotencyKey injects an idempotency key into a context.
// Used by Middleware after validation, and by tests.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, ctxKey, key)
}

// Middleware extracts the Idempotency-Key header from incoming HTTP requests,
// validates its format if present, and injects it into the request context.
//
// Validation rules (PRD §8.7):
//   - The header is optional. Requests without it pass through unchanged.
//   - If present, the value must be ≤ 255 characters.
//   - If present, the value should be a UUID v4 format (RFC 4122).
//     Non-UUID values are accepted with a warning but are not rejected,
//     since the middleware layer does not own the idempotency semantics
//     — the service layer does.
//   - The header name is case-insensitive per HTTP/1.1.
//
// This middleware does not decide which endpoints require idempotency.
// That responsibility belongs to the controller/service layer, which
// calls GetIdempotencyKey to retrieve the value and determines whether
// a missing key is an error.
//
// Only POST, PUT, and PATCH requests are inspected for the header.
// GET, DELETE, HEAD, OPTIONS, and other safe/idempotent methods are
// passed through without inspection.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only inspect mutating methods. Safe methods (GET, HEAD, OPTIONS)
		// and DELETE (which is idempotent by HTTP semantics) do not need
		// idempotency key validation.
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch:
			// Inspect the header.
		default:
			next.ServeHTTP(w, r)
			return
		}

		key := r.Header.Get(IdempotencyKeyHeader)
		if key == "" {
			// No key provided. The controller decides whether this is an error.
			next.ServeHTTP(w, r)
			return
		}

		// Trim whitespace. HTTP header values should not have leading/trailing
		// whitespace, but some clients add it.
		key = strings.TrimSpace(key)

		if err := ValidateKey(key); err != nil {
			// Key format is invalid. Reject immediately with a clear error.
			http.Error(w, fmt.Sprintf(`{"error":{"code":"IDEMPOTENCY_KEY_INVALID","message":%q}}`, err.Error()),
				http.StatusBadRequest)
			return
		}

		// Inject the validated key into the context for downstream handlers.
		ctx := WithIdempotencyKey(r.Context(), key)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ValidateKey checks that an idempotency key conforms to the required format.
//
// Rules:
//   - Must not be empty.
//   - Must not exceed MaxKeyLength (255) characters.
//   - SHOULD be a valid UUID v4 (RFC 4122). Non-UUID keys are rejected
//     at this validation layer because random UUIDs are required for the
//     at-most-once guarantee to be meaningful. Short or predictable keys
//     increase the risk of accidental collision.
//
// Returns nil if the key is valid, or a descriptive error if it fails
// validation.
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("idempotency key is empty")
	}

	if len(key) > MaxKeyLength {
		return fmt.Errorf("idempotency key exceeds maximum length of %d characters", MaxKeyLength)
	}

	// Validate UUID v4 format: 8-4-4-4-12 hex digits with dashes.
	// Format: xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
	if !isValidUUIDv4(key) {
		return fmt.Errorf("idempotency key must be a valid UUID v4 (RFC 4122)")
	}

	return nil
}

// isValidUUIDv4 checks whether a string is a valid UUID v4 (RFC 4122).
// The expected format is: xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx
// where x is a lowercase hexadecimal digit.
//
// This is a fast, allocation-free check that validates:
//   - Exact length of 36 characters.
//   - Dashes at positions 8, 13, 18, 23.
//   - Version nibble is 4.
//   - Variant nibble is 8, 9, a, or b.
//   - All other characters are valid hex digits.
func isValidUUIDv4(s string) bool {
	if len(s) != 36 {
		return false
	}

	// Check dash positions.
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}

	// Check version nibble (position 14 must be '4').
	if s[14] != '4' {
		return false
	}

	// Check variant nibble (position 19 must be 8, 9, a, or b).
	switch s[19] {
	case '8', '9', 'a', 'b', 'A', 'B':
		// Valid variant.
	default:
		return false
	}

	// Verify all remaining characters are hex digits.
	for i := 0; i < 36; i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue // already checked dashes
		}
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

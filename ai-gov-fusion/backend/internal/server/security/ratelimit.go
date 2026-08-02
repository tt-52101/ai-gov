package security

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ── 令牌桶限流器 ────────────────────────────────────────────────────────────

// TokenBucket 基于令牌桶算法的限流器，用于异常流量拦截与告警（PRD SEC-04）。
//
// 以恒定的速率向桶中添加令牌，每次请求消耗一个令牌。当桶中令牌耗尽时，
// 请求被拒绝并记录告警日志。支持突发流量（桶容量 = 速率 × 突发倍数）。
//
// 实现线程安全——所有公共方法均可安全并发调用。
type TokenBucket struct {
	mu         sync.Mutex
	rate       float64       // 每秒令牌生成速率
	capacity   float64       // 桶容量（最大突发量）
	tokens     float64       // 当前令牌数
	lastRefill time.Time     // 上次补充令牌的时间
	window     time.Duration // 统计窗口（仅用于日志上下文）
}

// NewTokenBucket 创建令牌桶限流器。
//
//   - rate: 每秒允许的请求数。
//   - burst: 最大突发请求数（桶容量），建议 >= rate。
//
// 若 rate <= 0 视为不限流（所有 Allow 返回 true）。
func NewTokenBucket(rate float64, burst int) *TokenBucket {
	if rate <= 0 {
		return &TokenBucket{rate: 0, capacity: 0, tokens: 0, lastRefill: time.Now()}
	}
	capacity := float64(burst)
	if capacity < 1 {
		capacity = 1
	}
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     capacity,
		lastRefill: time.Now(),
		window:     time.Minute,
	}
}

// Allow 检查是否允许一个请求通过。消耗一个令牌。
// 若不限流（rate <= 0），始终返回 true。
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN 检查是否允许 N 个请求通过。消耗 N 个令牌。
// 若不限流（rate <= 0），始终返回 true。
func (tb *TokenBucket) AllowN(n int) bool {
	if tb == nil || tb.rate <= 0 {
		return true
	}
	if n <= 0 {
		return true
	}

	tb.mu.Lock()
	defer tb.mu.Unlock()

	// 补充令牌。
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed > 0 {
		tb.tokens += elapsed * tb.rate
		if tb.tokens > tb.capacity {
			tb.tokens = tb.capacity
		}
		tb.lastRefill = now
	}

	// 检查是否有足够令牌。
	needed := float64(n)
	if tb.tokens >= needed {
		tb.tokens -= needed
		return true
	}
	return false
}

// ── 基于 Key 的限流器 ──────────────────────────────────────────────────────

// KeyRateLimiter 基于 key（如 API Key、用户 ID、IP 地址）的限流器。
// 每个 key 对应一个独立的令牌桶。
type KeyRateLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket
	rate     float64 // 每秒每个 key 允许的请求数
	burst    int     // 每个 key 的最大突发请求数
	interval time.Duration
}

// NewKeyRateLimiter 创建基于 key 的限流器。
//
//   - rate: 每秒每个 key 允许的请求数。
//   - burst: 每个 key 的最大突发请求数。
func NewKeyRateLimiter(rate float64, burst int) *KeyRateLimiter {
	return &KeyRateLimiter{
		buckets:  make(map[string]*TokenBucket),
		rate:     rate,
		burst:    burst,
		interval: time.Minute,
	}
}

// Allow 检查指定 key 是否允许一个请求通过。
// 若 key 不存在则自动创建桶。
func (k *KeyRateLimiter) Allow(key string) bool {
	if k == nil || k.rate <= 0 {
		return true
	}

	k.mu.RLock()
	bucket, exists := k.buckets[key]
	k.mu.RUnlock()

	if !exists {
		k.mu.Lock()
		// 双重检查。
		if bucket, exists = k.buckets[key]; !exists {
			bucket = NewTokenBucket(k.rate, k.burst)
			k.buckets[key] = bucket
		}
		k.mu.Unlock()
	}

	return bucket.Allow()
}

// ── HTTP 中间件 ────────────────────────────────────────────────────────────

// RateLimitExceededHandler 限流触发时的处理函数类型。
// 默认返回 429 Too Many Requests。
type RateLimitExceededHandler func(w http.ResponseWriter, r *http.Request)

// DefaultRateLimitExceededHandler 默认的限流触发处理函数——返回 429 并记录告警日志。
func DefaultRateLimitExceededHandler(w http.ResponseWriter, r *http.Request) {
	slog.WarnContext(r.Context(), "异常流量告警：请求被限流",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"user_agent", r.Header.Get("User-Agent"),
	)
	http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
}

// RateLimitMiddleware 基于 Key 的 HTTP 限流中间件。
//
// 使用示例：
//
//	limiter := security.NewKeyRateLimiter(100, 200) // 每秒 100 请求，突发 200
//	mux.Handle("/v1/chat/completions", security.RateLimitMiddleware(limiter, nextHandler))
//
// 从请求中提取 key 的策略：优先使用 X-API-Key 请求头，其次使用 RemoteAddr。
func RateLimitMiddleware(limiter *KeyRateLimiter, next http.Handler) http.Handler {
	return RateLimitMiddlewareWithKeyFunc(limiter, next, func(r *http.Request) string {
		if key := r.Header.Get("X-API-Key"); key != "" {
			return key
		}
		return r.RemoteAddr
	})
}

// RateLimitMiddlewareWithKeyFunc 支持自定义 key 提取函数的限流中间件。
func RateLimitMiddlewareWithKeyFunc(limiter *KeyRateLimiter, next http.Handler, keyFn func(*http.Request) string) http.Handler {
	if limiter == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := keyFn(r)
		if !limiter.Allow(key) {
			DefaultRateLimitExceededHandler(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ── 编译期接口断言 ──────────────────────────────────────────────────────────

var _ = context.Background // 确保 context 包被使用（保留供后续扩展）
package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ── TokenBucket 测试 ───────────────────────────────────────────────────────

// TestTokenBucket_Allow 验证令牌桶的基本放行行为。
func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(100, 100) // 每秒 100 请求，突发 100

	// 桶初始满，应允许前 100 个请求。
	for i := 0; i < 100; i++ {
		if !tb.Allow() {
			t.Fatalf("第 %d 个请求被拒绝，桶初始应满", i+1)
		}
	}

	// 第 101 个请求应被拒绝（桶空）。
	if tb.Allow() {
		t.Fatal("桶空后请求应被拒绝")
	}
}

// TestTokenBucket_Refill 验证令牌桶随时间恢复。
func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(10, 10) // 每秒 10 请求，突发 10

	// 消耗所有令牌。
	for i := 0; i < 10; i++ {
		tb.Allow()
	}

	// 桶空，应拒绝。
	if tb.Allow() {
		t.Fatal("桶空后请求应被拒绝")
	}

	// 等待 200ms，应恢复约 2 个令牌。
	time.Sleep(200 * time.Millisecond)

	// 应允许约 2 个请求。
	allowed := 0
	for i := 0; i < 5; i++ {
		if tb.Allow() {
			allowed++
		}
	}
	if allowed < 1 || allowed > 3 {
		t.Fatalf("200ms 后应恢复约 2 个令牌，实际允许了 %d 个", allowed)
	}
}

// TestTokenBucket_ZeroRate 验证 rate <= 0 时不限流。
func TestTokenBucket_ZeroRate(t *testing.T) {
	tb := NewTokenBucket(0, 0) // 不限流

	for i := 0; i < 1000; i++ {
		if !tb.Allow() {
			t.Fatal("不限流时所有请求应被放行")
		}
	}
}

// TestTokenBucket_AllowN 验证批量消耗令牌。
func TestTokenBucket_AllowN(t *testing.T) {
	tb := NewTokenBucket(10, 10) // 每秒 10 请求，突发 10

	// 一次消耗 10 个令牌。
	if !tb.AllowN(10) {
		t.Fatal("桶初始满，应允许 10 个令牌")
	}

	// 桶空，应拒绝。
	if tb.AllowN(1) {
		t.Fatal("桶空后 AllowN 应被拒绝")
	}
}

// TestTokenBucket_NilSafety 验证 nil 安全。
func TestTokenBucket_NilSafety(t *testing.T) {
	var tb *TokenBucket
	if !tb.Allow() {
		t.Fatal("nil 桶应放行所有请求")
	}
	if !tb.AllowN(100) {
		t.Fatal("nil 桶应放行所有请求")
	}
}

// ── KeyRateLimiter 测试 ────────────────────────────────────────────────────

// TestKeyRateLimiter_Allow 验证基于 key 的限流。
func TestKeyRateLimiter_Allow(t *testing.T) {
	limiter := NewKeyRateLimiter(5, 5) // 每秒 5 请求，突发 5

	// 同一个 key 应允许 5 个请求。
	for i := 0; i < 5; i++ {
		if !limiter.Allow("key-a") {
			t.Fatalf("key-a 第 %d 个请求被拒绝", i+1)
		}
	}

	// 第 6 个请求应被拒绝。
	if limiter.Allow("key-a") {
		t.Fatal("key-a 桶空后应被拒绝")
	}

	// 另一个 key 应独立限流，允许 5 个请求。
	for i := 0; i < 5; i++ {
		if !limiter.Allow("key-b") {
			t.Fatalf("key-b 第 %d 个请求被拒绝", i+1)
		}
	}
}

// TestKeyRateLimiter_ZeroRate 验证 rate <= 0 时不限流。
func TestKeyRateLimiter_ZeroRate(t *testing.T) {
	limiter := NewKeyRateLimiter(0, 0) // 不限流

	for i := 0; i < 100; i++ {
		if !limiter.Allow("any-key") {
			t.Fatal("不限流时所有请求应被放行")
		}
	}
}

// TestKeyRateLimiter_NilSafety 验证 nil 安全。
func TestKeyRateLimiter_NilSafety(t *testing.T) {
	var limiter *KeyRateLimiter
	if !limiter.Allow("any-key") {
		t.Fatal("nil limiter 应放行所有请求")
	}
}

// ── RateLimitMiddleware 测试 ───────────────────────────────────────────────

// TestRateLimitMiddleware 验证 HTTP 中间件限流行为。
func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewKeyRateLimiter(2, 2) // 每秒 2 请求，突发 2
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(limiter, next)

	// 前 2 个请求应成功。
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-API-Key", "test-key")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("第 %d 个请求应成功，实际: %d", i+1, rec.Code)
		}
	}

	// 第 3 个请求应返回 429。
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-API-Key", "test-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("第 3 个请求应返回 429，实际: %d", rec.Code)
	}
}

// TestRateLimitMiddleware_NilLimiter 验证 nil limiter 时中间件直接放行。
func TestRateLimitMiddleware_NilLimiter(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := RateLimitMiddleware(nil, next)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("nil limiter 应放行，实际: %d", rec.Code)
	}
}

// TestRateLimitMiddleware_CustomKeyFunc 验证自定义 key 提取函数。
func TestRateLimitMiddleware_CustomKeyFunc(t *testing.T) {
	limiter := NewKeyRateLimiter(1, 1)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// 使用 URL 参数作为 key。
	handler := RateLimitMiddlewareWithKeyFunc(limiter, next, func(r *http.Request) string {
		return r.URL.Query().Get("api_key")
	})

	// 第一个请求应成功。
	req := httptest.NewRequest(http.MethodGet, "/test?api_key=custom", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("第一个请求应成功，实际: %d", rec.Code)
	}

	// 第二个请求应被限流。
	req2 := httptest.NewRequest(http.MethodGet, "/test?api_key=custom", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusTooManyRequests {
		t.Fatalf("第二个请求应返回 429，实际: %d", rec2.Code)
	}

	// 不同 key 应独立限流。
	req3 := httptest.NewRequest(http.MethodGet, "/test?api_key=other", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("不同 key 的请求应成功，实际: %d", rec3.Code)
	}
}
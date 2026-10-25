package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestLimiterAllowsABurstThenRefillsSmoothly(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(3, time.Second)
	limiter.now = func() time.Time { return now }

	for i := 1; i <= 3; i++ {
		if !limiter.Allow("session-a") {
			t.Fatalf("request %d should have been allowed", i)
		}
	}
	if limiter.Allow("session-a") {
		t.Fatal("the fourth request in the same instant should be refused")
	}

	// A third of the window refills one token.
	now = now.Add(340 * time.Millisecond)
	if !limiter.Allow("session-a") {
		t.Fatal("one token should have refilled")
	}
	if limiter.Allow("session-a") {
		t.Fatal("only one token had refilled")
	}

	// Other keys have their own budget.
	if !limiter.Allow("session-b") {
		t.Fatal("a different key must not be affected by another key's burst")
	}
}

func TestLimiterSweepsIdleBuckets(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter := NewLimiter(1, time.Minute)
	limiter.now = func() time.Time { return now }

	for _, key := range []string{"a", "b", "c"} {
		limiter.Allow(key)
	}

	now = now.Add(2 * time.Minute)
	limiter.Allow("d")

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	if _, ok := limiter.buckets["a"]; ok {
		t.Error("idle buckets should be swept so memory does not grow forever")
	}
}

func TestRateLimitMiddlewareReturns429WithRetryAfter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	limiter := NewLimiter(1, time.Minute)
	router := gin.New()
	router.GET("/x",
		RateLimit(limiter, func(*gin.Context) string { return "fixed" }),
		func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	first := httptest.NewRecorder()
	router.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/x", nil))
	if first.Code != http.StatusOK {
		t.Fatalf("first request status = %d, want 200", first.Code)
	}

	second := httptest.NewRecorder()
	router.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/x", nil))
	if second.Code != http.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", second.Code)
	}
	if second.Header().Get("Retry-After") == "" {
		t.Error("a rate limited response should carry Retry-After")
	}
}

func TestNilLimiterNeverBlocks(t *testing.T) {
	var limiter *Limiter
	if !limiter.Allow("anything") {
		t.Fatal("a nil limiter should allow everything")
	}
	if limiter.RetryAfterSeconds() < 1 {
		t.Error("RetryAfterSeconds should always be at least 1")
	}
}

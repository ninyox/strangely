package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"strgback/utils"
)

// Limiter is a token bucket per key. Buckets refill continuously, so a burst
// is allowed but a sustained flood is not. Keys are session ids or peer
// addresses; no key is ever persisted.
type Limiter struct {
	mu       sync.Mutex
	buckets  map[string]*bucket
	capacity float64
	perSec   float64
	window   time.Duration
	now      func() time.Time
	swept    time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

// NewLimiter allows perWindow requests per window for each key, refilled
// smoothly across the window.
func NewLimiter(perWindow int, window time.Duration) *Limiter {
	if perWindow <= 0 {
		perWindow = 1
	}
	if window <= 0 {
		window = time.Minute
	}
	seconds := window.Seconds()
	return &Limiter{
		buckets:  make(map[string]*bucket),
		capacity: float64(perWindow),
		perSec:   float64(perWindow) / seconds,
		window:   window,
		now:      time.Now,
	}
}

// Allow consumes one token for key and reports whether the request may run.
func (l *Limiter) Allow(key string) bool {
	if l == nil {
		return true
	}

	now := l.now()

	l.mu.Lock()
	defer l.mu.Unlock()

	l.sweepLocked(now)

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.capacity, last: now}
		l.buckets[key] = b
	} else {
		elapsed := now.Sub(b.last).Seconds()
		if elapsed > 0 {
			b.tokens = min(l.capacity, b.tokens+elapsed*l.perSec)
			b.last = now
		}
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// sweepLocked drops buckets that have been idle long enough to be full again,
// so a long running process does not accumulate one entry per visitor forever.
func (l *Limiter) sweepLocked(now time.Time) {
	if now.Sub(l.swept) < time.Minute && len(l.buckets) < 4096 {
		return
	}
	l.swept = now
	refill := l.window
	for key, b := range l.buckets {
		if now.Sub(b.last) > refill {
			delete(l.buckets, key)
		}
	}
}

// RetryAfterSeconds is a hint for the Retry-After header.
func (l *Limiter) RetryAfterSeconds() int {
	if l == nil || l.perSec <= 0 {
		return 1
	}
	seconds := int(1 / l.perSec)
	if seconds < 1 {
		seconds = 1
	}
	return seconds
}

// RateLimit rejects requests that exceed the limiter's budget for their key.
func RateLimit(limiter *Limiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if limiter == nil || limiter.Allow(keyFunc(c)) {
			c.Next()
			return
		}

		c.Header("Retry-After", strconv.Itoa(limiter.RetryAfterSeconds()))
		utils.ErrorResponseCode(c, http.StatusTooManyRequests, "rate_limited",
			"Too many requests. Please wait a moment and try again.", nil)
		c.Abort()
	}
}

// ByIP keys the limiter by the peer address. gin reads the socket address
// unless TRUSTED_PROXIES is configured, so the key cannot be spoofed by a
// header on a directly exposed server.
func ByIP(c *gin.Context) string {
	return c.ClientIP()
}

// BySession keys the limiter by the authenticated session.
func BySession(c *gin.Context) string {
	if id := SessionIDFrom(c); id != "" {
		return id
	}
	return c.ClientIP()
}

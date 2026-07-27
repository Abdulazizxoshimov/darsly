package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	hs "github.com/zoom/darsly/api/http_status"
)

type rateBucket struct {
	tokens   float64
	lastSeen time.Time
}

type rateLimiter struct {
	mu              sync.Mutex
	buckets         map[string]*rateBucket
	rate            float64 // tokens per second
	burst           float64
	cleanupInterval time.Duration
}

func newRateLimiter(rps float64, burst int) *rateLimiter {
	rl := &rateLimiter{
		buckets:         make(map[string]*rateBucket),
		rate:            rps,
		burst:           float64(burst),
		cleanupInterval: 5 * time.Minute,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, ok := rl.buckets[key]
	if !ok {
		rl.buckets[key] = &rateBucket{tokens: rl.burst - 1, lastSeen: now}
		return true
	}

	elapsed := now.Sub(b.lastSeen).Seconds()
	b.tokens = min(rl.burst, b.tokens+elapsed*rl.rate)
	b.lastSeen = now

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

func (rl *rateLimiter) cleanup() {
	t := time.NewTicker(rl.cleanupInterval)
	defer t.Stop()
	for range t.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.cleanupInterval)
		for k, b := range rl.buckets {
			if b.lastSeen.Before(cutoff) {
				delete(rl.buckets, k)
			}
		}
		rl.mu.Unlock()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// RateLimit applies a token-bucket rate limiter keyed by client IP.
// rps: max requests per second; burst: max burst size.
func RateLimit(rps float64, burst int) gin.HandlerFunc {
	rl := newRateLimiter(rps, burst)
	return func(c *gin.Context) {
		if !rl.allow(c.ClientIP()) {
			hs.AbortError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
			return
		}
		c.Next()
	}
}

// RateLimitByUser applies a token-bucket rate limiter keyed by authenticated user ID.
// Falls back to IP if no user ID is set in context.
func RateLimitByUser(rps float64, burst int) gin.HandlerFunc {
	rl := newRateLimiter(rps, burst)
	return func(c *gin.Context) {
		key := c.GetString(CtxUserID)
		if key == "" {
			key = c.ClientIP()
		}
		if !rl.allow("user:" + key) {
			hs.AbortError(c, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests")
			return
		}
		c.Next()
	}
}

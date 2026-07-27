package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/internal/infrastructure/redis"
)

// RateLimitRedis — Redis-backed fixed-window rate limiter (IP bo'yicha).
// In-memory limiter'dan farqli — ko'p-instansda umumiy hisoblagich (brute-force
// himoyasi masshtabda zaiflashmaydi).
//
// Fail-safe: Redis mavjud bo'lmasa (cache==nil) yoki Incr xato bersa, per-instans
// in-memory token-bucket fallback'ga o'tadi — shunda Redis o'chsa ham auth brute-force
// cheklovsiz qolmaydi (fail-open emas). Fallback tezligi Redis limitiga tenglashtiriladi.
//
//	limit  — window ichida ruxsat etilgan so'rovlar soni
//	window — oyna davomiyligi
func RateLimitRedis(cache redis.Cache, prefix string, limit int64, window time.Duration) gin.HandlerFunc {
	return RateLimitRedisKeyed(cache, prefix, limit, window, func(c *gin.Context) string {
		return c.ClientIP()
	})
}

// RateLimitRedisKeyed — RateLimitRedis'ning kalit-ajratuvchisi sozlanadigan varianti.
// keyFn bo'sh satr qaytarsa (masalan body'da kutilgan maydon yo'q) cheklov shu so'rov
// uchun O'TKAZIB YUBORILADI — bunday holatni chaqiruvchi boshqa o'lcham bilan (masalan
// IP shifti) qoplashi kerak.
func RateLimitRedisKeyed(
	cache redis.Cache,
	prefix string,
	limit int64,
	window time.Duration,
	keyFn func(*gin.Context) string,
) gin.HandlerFunc {
	// Fallback token-bucket: window davomida ~limit so'rov (rps = limit/window),
	// burst = limit. Redis ishlaganda umuman ishlatilmaydi.
	secs := window.Seconds()
	rps := float64(limit)
	if secs > 0 {
		rps = float64(limit) / secs
	}
	fallback := newRateLimiter(rps, int(limit))

	return func(c *gin.Context) {
		id := keyFn(c)
		if id == "" {
			c.Next()
			return
		}

		if cache == nil {
			if !fallback.allow(prefix + ":" + id) {
				rejectRateLimited(c, window)
				return
			}
			c.Next() // ruxsat berilgan — zanjir davom etishi SHART (aks holda bo'sh 200)
			return
		}
		key := "rl:" + prefix + ":" + id
		n, err := cache.Incr(c.Request.Context(), key, window)
		if err != nil {
			// Redis xatosi — fail-open o'rniga in-memory limiterga tayanamiz.
			if !fallback.allow(prefix + ":" + id) {
				rejectRateLimited(c, window)
				return
			}
			c.Next()
			return
		}
		if n > limit {
			rejectRateLimited(c, window)
			return
		}
		c.Next()
	}
}

// rejectRateLimited 429 bilan so'rovni to'xtatadi (Redis va fallback yo'llari uchun umumiy).
func rejectRateLimited(c *gin.Context, window time.Duration) {
	c.Header("Retry-After", strconv.Itoa(int(window.Seconds())))
	hs.AbortError(c, http.StatusTooManyRequests, "RATE_LIMITED", "rate limit exceeded")
}

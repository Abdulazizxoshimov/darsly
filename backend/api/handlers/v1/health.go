package v1

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/internal/pkg/logger"
)

func HealthCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// ReadyCheck returns 200 when DB and Redis are reachable, 503 otherwise.
// Used as a Kubernetes readiness probe (/ready).
//
// Xavfsizlik: /ready OCHIQ (auth'siz) endpoint, shuning uchun ichki xato matni
// klientga QAYTARILMAYDI — readyFn `fmt.Errorf("postgres: %w", ...)` qaytaradi va
// pgx ulanish xatosi DSN tafsilotlarini (host/user/database) o'z ichiga oladi.
// To'liq xato faqat log/Sentry'ga; javobda umumiy matn qoladi.
// `status` maydoni SAQLANADI — monitoring probe'lari shuni parse qiladi.
//
// Natija readyCacheTTL (2s) keshlanadi: /ready ochiq, har hit PG+Redis+MinIO'ni
// ping qilmasin (probe tez-tez uriladi, hujumchi esa DB'ni shu bilan bosishi mumkin).
func ReadyCheck(check func() error, log logger.Logger) gin.HandlerFunc {
	return newReadyHandler(check, log, readyCacheTTL, time.Now)
}

// readyCacheTTL — readiness natijasi qancha vaqt qayta ishlatiladi.
const readyCacheTTL = 2 * time.Second

func newReadyHandler(check func() error, log logger.Logger, ttl time.Duration, now func() time.Time) gin.HandlerFunc {
	var (
		mu      sync.Mutex
		at      time.Time
		lastErr error
		valid   bool
	)
	cached := func() error {
		mu.Lock()
		defer mu.Unlock()
		if valid && now().Sub(at) < ttl {
			return lastErr
		}
		lastErr = check()
		at, valid = now(), true
		return lastErr
	}
	return func(c *gin.Context) {
		if err := cached(); err != nil {
			log.Error(c.Request.Context(), "readiness check failed", logger.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unavailable",
				"error":  "dependency unavailable",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	}
}

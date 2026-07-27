package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// Recover catches panics, logs the stack trace, and returns 500.
func Recover(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				ctx := c.Request.Context()
				log.Error(ctx, "panic recovered",
					logger.Any("error", r),
					logger.String("stack", string(debug.Stack())),
					logger.String("path", c.Request.URL.Path),
					logger.String("method", c.Request.Method),
				)
				hs.AbortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
		}()
		c.Next()
	}
}

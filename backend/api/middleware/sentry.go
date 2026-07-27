package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	sentry "github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"

	hs "github.com/zoom/darsly/api/http_status"
)

// Sentry captures panics and HTTP 5xx responses, sending them to Sentry.
// Must be registered before other middleware so it wraps the full call stack.
func Sentry() gin.HandlerFunc {
	return func(c *gin.Context) {
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(c.Request)
		hub.Scope().SetTag("path", c.FullPath())

		defer func() {
			if r := recover(); r != nil {
				hub.Scope().SetTag("stack", string(debug.Stack()))
				hub.RecoverWithContext(c.Request.Context(), r)
				hs.AbortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
		}()

		c.Next()

		status := c.Writer.Status()
		if status >= 500 {
			hub.Scope().SetTag("status", fmt.Sprintf("%d", status))
			if len(c.Errors) > 0 {
				hub.CaptureException(c.Errors.Last().Err)
			} else {
				hub.CaptureMessage(fmt.Sprintf("%s %s → %d", c.Request.Method, c.Request.URL.Path, status))
			}
		}
	}
}

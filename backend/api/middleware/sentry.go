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
		hub.Scope().SetRequest(sanitizedRequest(c))
		hub.Scope().SetTag("path", c.FullPath())
		if id, ok := c.Get(HeaderRequestID); ok {
			if s, ok := id.(string); ok {
				hub.Scope().SetTag("request_id", s)
			}
		}

		defer func() {
			if r := recover(); r != nil {
				hub.Scope().SetTag("stack", string(debug.Stack()))
				hub.RecoverWithContext(c.Request.Context(), r)
				hs.AbortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
			}
		}()

		c.Next()

		status := c.Writer.Status()
		// /ready 503 — bu bog'liqlik uzilishining PROBE signali (har 5-10s takrorlanadi);
		// u allaqachon log/alert'da va Sentry'da shovqin yaratardi.
		if status >= 500 && c.Request.URL.Path != "/ready" {
			hub.Scope().SetTag("status", fmt.Sprintf("%d", status))
			if len(c.Errors) > 0 {
				hub.CaptureException(c.Errors.Last().Err)
			} else {
				hub.CaptureMessage(fmt.Sprintf("%s %s → %d", c.Request.Method, routeOrUnmatched(c), status))
			}
		}
	}
}

// routeOrUnmatched — route shabloni (capability'siz), mos route bo'lmasa "unmatched".
func routeOrUnmatched(c *gin.Context) string {
	if p := c.FullPath(); p != "" {
		return p
	}
	return "unmatched"
}

// sanitizedRequest Sentry'ga yuboriladigan so'rov nusxasini yasaydi: sezgir query
// paramlar (?token=JWT, ?request_id=capability) niqoblanadi, path esa route shabloniga
// almashtiriladi (path parametrlarida ham capability bor). Asl so'rov o'zgarmaydi.
func sanitizedRequest(c *gin.Context) *http.Request {
	req := c.Request.Clone(c.Request.Context())
	if req.URL != nil {
		if req.URL.RawQuery != "" {
			req.URL.RawQuery = redactQuery(req.URL.RawQuery)
		}
		if p := c.FullPath(); p != "" {
			req.URL.Path = p
			req.URL.RawPath = ""
		}
	}
	return req
}

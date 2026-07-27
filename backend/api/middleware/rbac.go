package middleware

import (
	"net/http"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"

	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// Context keys for downstream handlers.
const (
	CtxUserID    = "user_id"
	CtxSessionID = "session_id"
	CtxRole      = "role"
)

// EnforceCasbin checks Casbin policy using the role already injected into
// context by the Auth middleware. Must be placed AFTER Auth in the chain.
// Returns 403 when the role lacks permission for the route+method.
func EnforceCasbin(enforcer *casbin.Enforcer, log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Enforcer yuklanmagan bo'lsa — fail-closed (nil.Enforce panic o'rniga 503).
		if enforcer == nil {
			log.Error(c.Request.Context(), "casbin enforcer not loaded — denying request")
			hs.AbortError(c, http.StatusServiceUnavailable, "AUTHZ_UNAVAILABLE", "authorization unavailable")
			return
		}

		role := c.GetString(CtxRole)
		if role == "" {
			role = "student" // safe fallback
		}

		allowed, err := enforcer.Enforce(role, c.FullPath(), c.Request.Method)
		if err != nil {
			log.Error(c.Request.Context(), "casbin enforce error", logger.Error(err))
			hs.AbortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
			return
		}
		if !allowed {
			hs.AbortError(c, http.StatusForbidden, "FORBIDDEN", "access denied")
			return
		}

		c.Next()
	}
}

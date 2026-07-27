package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"
)

// Auth validates the JWT and injects user identity into the Gin context.
// Unlike RBAC, it does not consult Casbin — use it for routes that only
// need authentication, not role-based access control.
func Auth(maker token.Maker) gin.HandlerFunc {
	return func(c *gin.Context) {
		// WebSocket clients cannot send custom headers during HTTP upgrade,
		// so we also accept the token as a ?token= query parameter.
		var tokenStr string
		if raw := c.GetHeader("Authorization"); raw != "" {
			s, ok := strings.CutPrefix(raw, "Bearer ")
			if !ok || s == "" {
				hs.AbortError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid authorization header format")
				return
			}
			tokenStr = s
		} else if q := c.Query("token"); q != "" {
			tokenStr = q
		} else {
			hs.AbortError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			return
		}

		claims, err := maker.ValidateAccess(c.Request.Context(), tokenStr)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				hs.AbortError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "token expired")
			} else {
				hs.AbortError(c, http.StatusUnauthorized, "TOKEN_INVALID", "invalid token")
			}
			return
		}

		c.Set(CtxUserID, claims.Sub)
		c.Set(CtxSessionID, claims.SessionID)
		c.Set(CtxRole, claims.Role)
		// user_id ni request context'ga ham — usecase loglariga tushadi.
		c.Request = c.Request.WithContext(logger.WithUser(c.Request.Context(), claims.Sub))
		c.Next()
	}
}

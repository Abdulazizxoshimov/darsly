package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	hs "github.com/zoom/darsly/api/http_status"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
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
		} else if q := c.Query("token"); q != "" && isWebSocketRequest(c) {
			// ?token= FAQAT WS uchun: URL'lar access log/proxy/Referer'ga tushadi,
			// shuning uchun oddiy route'larda token query'da qabul qilinmaydi.
			tokenStr = q
		} else {
			hs.AbortError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing authorization header")
			return
		}

		claims, err := maker.ValidateAccess(c.Request.Context(), tokenStr)
		if err != nil {
			switch {
			case errors.Is(err, jwt.ErrTokenExpired):
				hs.AbortError(c, http.StatusUnauthorized, "TOKEN_EXPIRED", "token expired")
			case errors.Is(err, token.ErrSessionRevoked):
				// Sessiya server tomonidan tugatilgan — eng ko'p uchraydigan sabab
				// "bitta akkaunt = bitta faol sessiya" siyosati (boshqa qurilmada
				// kirish). Klient shu kodga qarab «Boshqa qurilmada kirildi» deb
				// tushuntiradi; `TOKEN_INVALID` bunday farqni bermasdi.
				hs.AbortError(c, http.StatusUnauthorized, string(apperr.CodeSessionRevoked),
					"session ended: signed in on another device")
			default:
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

// isWebSocketRequest — so'rov WS yo'li (`/api/v1/ws...`) yoki Upgrade: websocket ekanini aniqlaydi.
func isWebSocketRequest(c *gin.Context) bool {
	if strings.EqualFold(c.GetHeader("Upgrade"), "websocket") {
		return true
	}
	p := c.FullPath()
	if p == "" {
		p = c.Request.URL.Path
	}
	return p == "/api/v1/ws" || strings.HasPrefix(p, "/api/v1/ws/")
}

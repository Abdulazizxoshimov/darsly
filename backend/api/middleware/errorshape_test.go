package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"
)

// BE-4: barcha middleware xato javoblari YAGONA {code, message} konvertida bo'lishi
// shart. Avval ular {error, code} qaytarardi — klient (web api.jsx `parseError` va
// mobil ApiError) `message`ni topolmasdi va eng muhim holatlarda (401 TOKEN_EXPIRED,
// 403, 429) foydalanuvchiga bo'sh matn ko'rsatilardi.

// stubMaker — ValidateAccess natijasini test boshqaradigan token.Maker (Redis kerak emas).
type stubMaker struct{ err error }

func (s *stubMaker) Generate(context.Context, string, string, string) (string, string, error) {
	return "a", "r", nil
}
func (s *stubMaker) ValidateAccess(context.Context, string) (*token.Claims, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &token.Claims{Sub: "u1", SessionID: "s1", Role: "mentor"}, nil
}
func (s *stubMaker) Rotate(context.Context, string) (string, string, error) { return "a", "r", nil }
func (s *stubMaker) SessionFromRefresh(string) (string, error)              { return "s1", nil }
func (s *stubMaker) Revoke(context.Context, string) error                   { return nil }
func (s *stubMaker) RevokeRefresh(context.Context, string) error            { return nil }
func (s *stubMaker) RevokeAllUserSessions(context.Context, string) error    { return nil }
func (s *stubMaker) StoreSession(context.Context, string, string, time.Duration) error {
	return nil
}
func (s *stubMaker) RevokeSession(context.Context, string) error { return nil }

// assertErrorShape javob tanasini tekshiradi: `code` va `message` to'ldirilgan,
// eskirgan `error` maydoni esa YO'Q (yagona shakl).
func assertErrorShape(t *testing.T, body []byte, wantCode string) {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(body, &m), "xato javobi JSON bo'lishi kerak: %s", body)

	code, ok := m["code"].(string)
	require.True(t, ok, "`code` maydoni string bo'lishi kerak: %s", body)
	require.Equal(t, wantCode, code)

	msg, ok := m["message"].(string)
	require.True(t, ok, "`message` maydoni string bo'lishi kerak: %s", body)
	require.NotEmpty(t, msg, "`message` bo'sh bo'lmasligi kerak (klient shuni ko'rsatadi)")

	_, hasErr := m["error"]
	require.False(t, hasErr, "eskirgan `error` maydoni qaytmasligi kerak: %s", body)
}

func TestAuthMiddleware_ErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		maker      token.Maker
		header     string
		wantStatus int
		wantCode   string
	}{
		{"header yo'q", &stubMaker{}, "", http.StatusUnauthorized, "UNAUTHORIZED"},
		{"Bearer prefiksi yo'q", &stubMaker{}, "Token abc", http.StatusUnauthorized, "UNAUTHORIZED"},
		{"token muddati tugagan", &stubMaker{err: jwt.ErrTokenExpired}, "Bearer abc", http.StatusUnauthorized, "TOKEN_EXPIRED"},
		{"token yaroqsiz", &stubMaker{err: errors.New("bad signature")}, "Bearer abc", http.StatusUnauthorized, "TOKEN_INVALID"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/x", Auth(tc.maker), func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/x", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			require.Equal(t, tc.wantStatus, w.Code)
			assertErrorShape(t, w.Body.Bytes(), tc.wantCode)
		})
	}
}

func TestRBACMiddleware_ErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)
	log := logger.New("error", "test", "v1")

	t.Run("403 forbidden", func(t *testing.T) {
		r := gin.New()
		r.Use(func(c *gin.Context) { c.Set(CtxRole, "student"); c.Next() })
		r.Use(EnforceCasbin(realEnforcer(t), log))
		r.GET("/api/v1/lessons", func(c *gin.Context) { c.Status(http.StatusOK) })

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/lessons", nil))

		require.Equal(t, http.StatusForbidden, w.Code)
		assertErrorShape(t, w.Body.Bytes(), "FORBIDDEN")
	})

	t.Run("503 enforcer yuklanmagan", func(t *testing.T) {
		r := gin.New()
		r.Use(EnforceCasbin(nil, log))
		r.GET("/api/v1/lessons", func(c *gin.Context) { c.Status(http.StatusOK) })

		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/lessons", nil))

		require.Equal(t, http.StatusServiceUnavailable, w.Code)
		assertErrorShape(t, w.Body.Bytes(), "AUTHZ_UNAVAILABLE")
	})
}

func TestRateLimitMiddleware_ErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// burst=1 → ikkinchi so'rov 429.
	r := gin.New()
	r.GET("/x", RateLimit(0.001, 1), func(c *gin.Context) { c.Status(http.StatusOK) })

	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusOK, first.Code)

	second := httptest.NewRecorder()
	r.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	assertErrorShape(t, second.Body.Bytes(), "RATE_LIMITED")
}

func TestRateLimitByUserMiddleware_ErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set(CtxUserID, "u1"); c.Next() })
	r.GET("/x", RateLimitByUser(0.001, 1), func(c *gin.Context) { c.Status(http.StatusOK) })

	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusOK, first.Code)

	second := httptest.NewRecorder()
	r.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	assertErrorShape(t, second.Body.Bytes(), "RATE_LIMITED")
}

// RateLimitRedis: cache==nil bo'lganda in-memory fallback ishlaydi. Bu yerda ikki narsa
// tekshiriladi: (1) 429 javobi to'g'ri shaklda, (2) RUXSAT berilgan so'rov zanjirni
// davom ettiradi (avval `c.Next()` chaqirilmagani uchun bo'sh 200 qaytardi).
func TestRateLimitRedisMiddleware_ErrorShapeAndPassThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x", RateLimitRedis(nil, "test", 1, time.Minute), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"data": "ok"})
	})

	first := httptest.NewRecorder()
	r.ServeHTTP(first, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusOK, first.Code)
	require.JSONEq(t, `{"data":"ok"}`, first.Body.String(), "fallback ruxsat berganda handler ishlashi kerak")

	second := httptest.NewRecorder()
	r.ServeHTTP(second, httptest.NewRequest(http.MethodGet, "/x", nil))
	require.Equal(t, http.StatusTooManyRequests, second.Code)
	assertErrorShape(t, second.Body.Bytes(), "RATE_LIMITED")
}

func TestRecoverMiddleware_ErrorShape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(Recover(logger.New("error", "test", "v1")))
	r.GET("/x", func(c *gin.Context) { panic("boom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	require.Equal(t, http.StatusInternalServerError, w.Code)
	assertErrorShape(t, w.Body.Bytes(), "INTERNAL_ERROR")
}

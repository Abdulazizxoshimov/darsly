package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	pcasbin "github.com/zoom/darsly/internal/pkg/casbin"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// realEnforcer — haqiqiy model.conf + policy.csv bilan (mock emas), shunda RBAC
// qoidalarining o'zi sinovdan o'tadi.
//
// DIQQAT: bu `internal/pkg/casbin` (joylashtirilgan siyosat), upstream
// `casbin/v2` EMAS. Upstream'ning `NewEnforcer()` si variadik — argumentsiz
// chaqirilsa modelsiz, siyosatsiz enforcer qaytaradi va hamma tekshiruv
// ma'nosiz bo'lib qoladi (test yashil turib RBAC'ni umuman sinamaydi).
func realEnforcer(t *testing.T) *casbin.Enforcer {
	t.Helper()
	e, err := pcasbin.NewEnforcer()
	require.NoError(t, err)
	return e
}

func rbacEngine(enforcer *casbin.Enforcer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	// Auth o'rniga: rolni test header'idan olib context'ga qo'yamiz.
	r.Use(func(c *gin.Context) {
		if role := c.GetHeader("X-Test-Role"); role != "" {
			c.Set(CtxRole, role)
		}
		c.Next()
	})
	r.Use(EnforceCasbin(enforcer, logger.New("error", "test", "v1")))
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/api/v1/users/:id", ok)
	r.DELETE("/api/v1/users/:id", ok)
	r.POST("/api/v1/lessons/:id/participants/:identity/remove", ok)
	r.GET("/api/v1/notifications", ok)
	r.GET("/api/v1/lessons", ok)
	return r
}

func doReq(r *gin.Engine, role, method, path string) int {
	req := httptest.NewRequest(method, path, nil)
	if role != "" {
		req.Header.Set("X-Test-Role", role)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestEnforceCasbin_RolePermissions(t *testing.T) {
	r := rbacEngine(realEnforcer(t))

	cases := []struct {
		name       string
		role       string
		method     string
		path       string
		wantStatus int
	}{
		{"student o'z profilini ko'radi", "student", http.MethodGet, "/api/v1/users/u1", http.StatusOK},
		{"student user o'chira olmaydi (admin-only)", "student", http.MethodDelete, "/api/v1/users/u1", http.StatusForbidden},
		{"mentor endi user o'chira OLMAYDI (H-2: user-CRUD admin'da)", "mentor", http.MethodDelete, "/api/v1/users/u1", http.StatusForbidden},
		{"admin user o'chiradi", "admin", http.MethodDelete, "/api/v1/users/u1", http.StatusOK},
		{"admin mentor huquqlarini meros oladi (ishtirokchini chiqaradi)", "admin", http.MethodPost, "/api/v1/lessons/l1/participants/g1/remove", http.StatusOK},
		{"student ishtirokchini chiqara olmaydi", "student", http.MethodPost, "/api/v1/lessons/l1/participants/g1/remove", http.StatusForbidden},
		{"mentor ishtirokchini chiqaradi", "mentor", http.MethodPost, "/api/v1/lessons/l1/participants/g1/remove", http.StatusOK},
		{"student bildirishnomalarni ko'radi", "student", http.MethodGet, "/api/v1/notifications", http.StatusOK},
		{"mentor student huquqlarini meros oladi (notifications)", "mentor", http.MethodGet, "/api/v1/notifications", http.StatusOK},
		{"student darslar ro'yxatini ko'ra olmaydi", "student", http.MethodGet, "/api/v1/lessons", http.StatusForbidden},
		{"noma'lum rol rad etiladi", "hacker", http.MethodGet, "/api/v1/lessons", http.StatusForbidden},
		{"rolsiz (bo'sh) student'ga fallback → lessons rad", "", http.MethodGet, "/api/v1/lessons", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := doReq(r, tc.role, tc.method, tc.path)
			require.Equal(t, tc.wantStatus, got)
		})
	}
}

func TestEnforceCasbin_MentorRemovesParticipant(t *testing.T) {
	r := rbacEngine(realEnforcer(t))
	// mentor ishtirokchini chiqarishga ruxsatli.
	require.Equal(t, http.StatusOK, doReq(r, "mentor", http.MethodPost, "/api/v1/lessons/l1/participants/g1/remove"))
}

func TestEnforceCasbin_NilEnforcer_FailsClosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(EnforceCasbin(nil, logger.New("error", "test", "v1")))
	r.GET("/api/v1/lessons", func(c *gin.Context) { c.Status(http.StatusOK) })
	require.Equal(t, http.StatusServiceUnavailable, doReq(r, "mentor", http.MethodGet, "/api/v1/lessons"))
}

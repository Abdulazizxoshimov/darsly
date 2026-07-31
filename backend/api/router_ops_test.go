package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/api/handlers"
	pcasbin "github.com/zoom/darsly/internal/pkg/casbin"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/testutil"
)

// Operatsion endpointlar (`/metrics`, `/swagger`) muhitga qarab OCHILADI yoki
// YOPILADI. Bu qaror xavfsizlik qaroridir va u test bilan qotirilishi kerak:
//
//   - `/swagger` production'da butun API sxemasini (barcha endpointlar, so'rov
//     shakllari, validatsiya qoidalari) autentifikatsiyasiz oshkor qilardi;
//   - `/metrics` esa ichki topologiyani, so'rovlar hajmini va xatolar
//     taqsimotini bergan.
//
// Regressiya bu yerda JIMGINA bo'ladi: endpoint qayta ochilib qolsa hech narsa
// buzilmaydi va hech kim sezmaydi — shuning uchun avtomatik tekshiruv shart.
func newOpsRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	enforcer, err := pcasbin.NewEnforcer()
	require.NoError(t, err)
	return NewRouter(
		&handlers.Handler{},
		testutil.NewFakeTokenMaker(),
		enforcer,
		testutil.NewFakeCache(),
		logger.New("error", "test", "v1"),
		func() error { return nil },
	)
}

func doGet(r *gin.Engine, path, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestOpsEndpoints_NonProduction(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	r := newOpsRouter(t)

	require.Equal(t, http.StatusOK, doGet(r, "/metrics", "").Code,
		"dev'da /metrics tokensiz ochiq bo'lishi kerak")
	// Swagger UI redirect bilan boshlanadi — 404 BO'LMASLIGI muhim.
	require.NotEqual(t, http.StatusNotFound, doGet(r, "/swagger/index.html", "").Code,
		"dev'da /swagger ochiq bo'lishi kerak")
}

func TestOpsEndpoints_ProductionWithToken(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("METRICS_TOKEN", "s3cret-scrape-token")
	r := newOpsRouter(t)

	t.Run("/swagger production'da YO'Q", func(t *testing.T) {
		require.Equal(t, http.StatusNotFound, doGet(r, "/swagger/index.html", "").Code)
	})

	t.Run("/metrics tokensiz rad etiladi", func(t *testing.T) {
		require.Equal(t, http.StatusUnauthorized, doGet(r, "/metrics", "").Code)
	})

	t.Run("noto'g'ri token rad etiladi", func(t *testing.T) {
		require.Equal(t, http.StatusUnauthorized, doGet(r, "/metrics", "Bearer boshqa-token").Code)
	})

	t.Run("to'g'ri token bilan ochiladi", func(t *testing.T) {
		w := doGet(r, "/metrics", "Bearer s3cret-scrape-token")
		require.Equal(t, http.StatusOK, w.Code)
		// Prometheus formati qaytayotganini tasdiqlaymiz (bo'sh 200 emas).
		require.Contains(t, w.Body.String(), "# HELP")
	})
}

// METRICS_TOKEN berilmasa endpoint production'da UMUMAN yoqilmaydi.
//
// "Himoyasiz ochiq qolgan"dan ko'ra "yo'q" holati xavfsizroq: konfiguratsiya
// unutilsa nosozlik darhol ko'rinadi (Prometheus 404 oladi), jimgina oqib
// turmaydi.
func TestOpsEndpoints_ProductionWithoutToken(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("METRICS_TOKEN", "")
	r := newOpsRouter(t)

	require.Equal(t, http.StatusNotFound, doGet(r, "/metrics", "").Code,
		"token sozlanmagan bo'lsa /metrics ro'yxatdan o'tmasligi kerak")
}

// Sog'liq endpointlari HAR DOIM ochiq: ular konteyner healthcheck'i va
// Caddy/LB uchun kerak, tokensiz.
func TestHealthEndpointsAlwaysOpen(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	r := newOpsRouter(t)

	require.Equal(t, http.StatusOK, doGet(r, "/health", "").Code)
	require.Equal(t, http.StatusOK, doGet(r, "/ready", "").Code)
}

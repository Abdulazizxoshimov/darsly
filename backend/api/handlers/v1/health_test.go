package v1_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	v1 "github.com/zoom/darsly/api/handlers/v1"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// silentLogger — bu testda xato ATAYLAB yuz beradi, shuning uchun log yozilmasin
// (aks holda o'tayotgan to'plamda chalg'ituvchi qizil ERROR satri chiqadi). To'liq
// xato log'ga ketishi TestReadyCheck_DoesNotLeakInternalError mantiqiga ta'sir qilmaydi.
func silentLogger() logger.Logger { return logger.New(logger.LevelFatal, "test", "test") }

// /ready OCHIQ endpoint — ichki xato matni (pgx DSN: host/user/database) klientga
// SIZIB CHIQMASLIGI shart. `status` maydoni saqlanadi (monitoring uni parse qiladi).
func TestReadyCheck_DoesNotLeakInternalError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Haqiqiy readyFn shunday xato qaytaradi — DSN tafsilotlari ichida.
	leaky := fmt.Errorf("postgres: %w", errors.New(
		`failed to connect to host=10.0.0.5 user=darsly_admin database=darsly: password authentication failed`))

	r := gin.New()
	r.GET("/ready", v1.ReadyCheck(func() error { return leaky }, silentLogger()))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))

	require.Equal(t, http.StatusServiceUnavailable, w.Code)

	body := w.Body.String()
	require.NotContains(t, body, "10.0.0.5", "host manzili oshkor bo'lmasligi kerak")
	require.NotContains(t, body, "darsly_admin", "DB foydalanuvchisi oshkor bo'lmasligi kerak")
	require.NotContains(t, body, "password authentication failed", "ichki xato matni oshkor bo'lmasligi kerak")
	require.NotContains(t, body, "postgres:", "qaysi bog'liqlik yiqilgani oshkor bo'lmasligi kerak")

	var m map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	require.Equal(t, "unavailable", m["status"], "monitoring uchun `status` saqlanishi shart")
	require.Equal(t, "dependency unavailable", m["error"])
}

func TestReadyCheck_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/ready", v1.ReadyCheck(func() error { return nil }, silentLogger()))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ready", nil))

	require.Equal(t, http.StatusOK, w.Code)

	var m map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &m))
	require.Equal(t, "ready", m["status"])
}

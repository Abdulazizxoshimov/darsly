package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/testutil"
)

// ?token= FAQAT WS yo'lida; oddiy route'da faqat Authorization header.
func TestAuth_QueryToken_OnlyOnWebSocketPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	auth := Auth(testutil.NewFakeTokenMaker())
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.GET("/api/v1/ws", auth, ok)
	r.GET("/api/v1/users", auth, ok)

	do := func(path string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil).WithContext(context.Background()))
		return w.Code
	}
	require.Equal(t, http.StatusOK, do("/api/v1/ws?token=abc"), "WS yo'lida query token qabul qilinadi")
	require.Equal(t, http.StatusUnauthorized, do("/api/v1/users?token=abc"), "oddiy route'da query token rad etiladi")

	// Header har qanday route'da ishlaydi.
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer abc")
	r.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

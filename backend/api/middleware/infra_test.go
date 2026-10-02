package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/zoom/darsly/internal/pkg/logger"
)

// capLog — yozilgan log maydonlarini ushlab oladi (faqat bizga kerak metodlar ma'noli).
type capLog struct {
	mu      sync.Mutex
	entries []map[string]any
}

func (l *capLog) rec(fields []logger.Field) {
	m := map[string]any{}
	for _, f := range fields {
		if f.Type == zapcore.StringType {
			m[f.Key] = f.String
		}
	}
	l.mu.Lock()
	l.entries = append(l.entries, m)
	l.mu.Unlock()
}
func (l *capLog) Debug(_ context.Context, _ string, f ...logger.Field) { l.rec(f) }
func (l *capLog) Info(_ context.Context, _ string, f ...logger.Field)  { l.rec(f) }
func (l *capLog) Warn(_ context.Context, _ string, f ...logger.Field)  { l.rec(f) }
func (l *capLog) Error(_ context.Context, _ string, f ...logger.Field) { l.rec(f) }
func (l *capLog) Fatal(_ context.Context, _ string, f ...logger.Field) { l.rec(f) }

func TestRequestID_RejectsUnsafeClientValues(t *testing.T) {
	require.True(t, validRequestID("abc-123_X.y"))
	require.False(t, validRequestID(""))
	require.False(t, validRequestID(strings.Repeat("a", 65)))
	require.False(t, validRequestID("bad\nlog injection"))
	require.False(t, validRequestID(`a"b`))

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderRequestID, "good-id")
	r.ServeHTTP(w, req)
	require.Equal(t, "good-id", w.Header().Get(HeaderRequestID))

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderRequestID, strings.Repeat("x", 500))
	r.ServeHTTP(w, req)
	require.Len(t, w.Header().Get(HeaderRequestID), 36, "uzun ID o'rniga yangi UUID")
}

func TestCORS_SetsVaryOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS("https://a.example"))
	r.GET("/", func(c *gin.Context) { c.Status(200) })

	for _, origin := range []string{"https://a.example", "https://evil.example", ""} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		r.ServeHTTP(w, req)
		require.Contains(t, w.Header().Values("Vary"), "Origin", "origin=%q", origin)
	}
}

func TestRecover_RecordsErrorForSentry(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var captured error
	r.Use(func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			captured = c.Errors.Last().Err
		}
	})
	r.Use(Recover(&capLog{}))
	r.GET("/boom", func(c *gin.Context) { panic("kaboom") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/boom", nil))
	require.Equal(t, 500, w.Code)
	require.Error(t, captured)
	require.Contains(t, captured.Error(), "panic: kaboom")
}

func TestLogger_LogsRouteTemplateNotRawPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cl := &capLog{}
	r := gin.New()
	r.Use(Logger(cl))
	r.GET("/waitingroom/:request_id/status", func(c *gin.Context) { c.Status(200) })

	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/waitingroom/SECRET-CAP/status", nil))
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/nope/SECRET-CAP", nil))

	require.Len(t, cl.entries, 2)
	require.Equal(t, "/waitingroom/:request_id/status", cl.entries[0]["path"])
	require.Equal(t, "unmatched", cl.entries[1]["path"])
}

func TestSanitizedRequest_RedactsSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var got *http.Request
	r.GET("/joinlink/:slug", func(c *gin.Context) { got = sanitizedRequest(c) })

	req := httptest.NewRequest(http.MethodGet, "/joinlink/SLUG123?token=JWT.SECRET&page=2", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, got)
	require.NotContains(t, got.URL.String(), "JWT.SECRET")
	require.NotContains(t, got.URL.String(), "SLUG123")
	require.Contains(t, got.URL.RawQuery, "page=2")
	require.Equal(t, "/joinlink/SLUG123", req.URL.Path, "asl so'rov o'zgarmasligi kerak")
}

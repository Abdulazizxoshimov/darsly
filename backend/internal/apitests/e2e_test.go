package apitests_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/zoom/darsly/api"
	"github.com/zoom/darsly/internal/app"
	"github.com/zoom/darsly/internal/infrastructure/email"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/pkg/casbin"
	"github.com/zoom/darsly/internal/pkg/config"
	"github.com/zoom/darsly/internal/pkg/hasher"
	pgpkg "github.com/zoom/darsly/internal/pkg/postgres"
	"github.com/zoom/darsly/internal/pkg/token"
	"github.com/zoom/darsly/internal/storage"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase"
)

func rootDir() string {
	_, file, _, _ := runtime.Caller(0) // .../internal/apitests/e2e_test.go
	return filepath.Join(filepath.Dir(file), "..", "..")
}

// newTestServer butun wired stack'ni (real PG+Redis, nop minio/livekit/email) ko'taradi.
func newTestServer(t *testing.T) (*httptest.Server, *pgpkg.Postgres) {
	t.Helper()
	pg := testutil.SetupTestDB(t)
	log := testutil.NewLogger()

	cache, err := redis.New(config.RedisConfig{
		Host: getenv("TEST_REDIS_HOST", "localhost"), Port: getenv("TEST_REDIS_PORT", "6399"),
	})
	if err != nil {
		testutil.SkipOrFail(t, "redis mavjud emas: %v", err)
	}
	if err := cache.Ping(context.Background()); err != nil {
		testutil.SkipOrFail(t, "redis ping: %v", err)
	}

	tokenMaker := token.NewJWTMaker([]byte("e2e-secret-at-least-32-characters-000"), 15*time.Minute, 720*time.Hour, token.DefaultRefreshGrace, cache.Client(), "e2etest", log)
	enforcer, err := casbin.NewEnforcer(
		filepath.Join(rootDir(), "internal/pkg/casbin/model.conf"),
		filepath.Join(rootDir(), "internal/pkg/casbin/policy.csv"),
	)
	require.NoError(t, err)

	hub := websocket.NewHub(log)
	lk := newTestLiveKit()
	uc := usecase.New(usecase.Deps{
		Store: storage.New(pg), TokenMaker: tokenMaker, Hasher: hasher.New(4),
		Minio: minio.NewNop(), Cache: cache, Log: log, Hub: hub,
		EmailSender: email.NewNopSender(), LiveKit: lk, RefreshTTL: 720 * time.Hour,
		FrontendBaseURL: "http://frontend",
	})
	cfg := &config.Config{}
	cfg.App.AllowOpenRegistration = true
	cfg.App.FrontendBaseURL = "http://frontend"
	// Mobil versiya nazorati (BE-5) — /app-config javobini tekshirish uchun.
	cfg.Mobile = config.MobileConfig{
		AndroidMinVersion:    "1.0.0",
		AndroidLatestVersion: "1.1.0",
		AndroidAPKURL:        "https://example.test/darsly-mentor.apk",
	}

	h := app.BuildHandler(uc, hub, lk, cfg)
	router := api.NewRouter(h, tokenMaker, enforcer, cache, log, func() error { return nil })
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, pg
}

func TestE2E_AuthLessonJoinFlow(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	// 1) Validatsiya: zaif parol → 400.
	code, _ := cl.post("/api/v1/auth/register", "", map[string]string{"full_name": "A", "email": "a@x.uz", "password": "123"})
	require.Equal(t, http.StatusBadRequest, code, "zaif parol validatsiyadan o'tmasligi kerak")

	// 2) To'g'ri register → 201 + token.
	code, body := cl.post("/api/v1/auth/register", "", map[string]string{"full_name": "Dilnoza", "email": "d@darsly.uz", "password": "parol12345"})
	require.Equal(t, http.StatusCreated, code)
	access := gjson(body, "data", "access_token")
	require.NotEmpty(t, access)

	// 3) /auth/me → student.
	code, body = cl.get("/api/v1/auth/me", access)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "student", gjson(body, "data", "role"))

	// 4) RBAC: student dars yarata olmaydi → 403.
	code, _ = cl.post("/api/v1/lessons", access, map[string]any{"title": "X"})
	require.Equal(t, http.StatusForbidden, code, "student lesson yarata olmaydi (RBAC)")

	// 5) Foydalanuvchini mentor qilamiz (DB) va qayta login qilamiz.
	_, err := pg.DB.Exec(context.Background(), "UPDATE users SET role='mentor' WHERE email='d@darsly.uz'")
	require.NoError(t, err)

	code, body = cl.post("/api/v1/auth/login", "", map[string]string{"email": "d@darsly.uz", "password": "parol12345"})
	require.Equal(t, http.StatusOK, code)
	mentorTok := gjson(body, "data", "access_token")

	// 6) Mentor dars yaratadi (kutish xonasi ON — livekit kerak emas) → 201.
	code, body = cl.post("/api/v1/lessons", mentorTok, map[string]any{"title": "Matematika", "is_waiting_room_enabled": true})
	require.Equal(t, http.StatusCreated, code)
	slug := gjson(body, "data", "join_slug")
	require.NotEmpty(t, slug)

	// 7) Public preview → 200.
	code, body = cl.get("/api/v1/joinlink/"+slug, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "Dilnoza", gjson(body, "data", "mentor_name"))

	// 8) Public join (kutish xonasi) → next_step=waiting_room + request_id.
	code, body = cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Aziz"})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "waiting_room", gjson(body, "data", "next_step"))
	require.NotEmpty(t, gjson(body, "data", "request_id"))
}

// ─── mayda HTTP klient ───────────────────────────────────────────────────────

type httpClient struct {
	t    *testing.T
	base string
}

func (c *httpClient) do(method, path, token string, payload any) (int, []byte) {
	var body io.Reader
	if payload != nil {
		b, _ := json.Marshal(payload)
		body = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, c.base+path, body)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(c.t, err)
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, data
}
func (c *httpClient) post(path, token string, payload any) (int, []byte) {
	return c.do(http.MethodPost, path, token, payload)
}
func (c *httpClient) get(path, token string) (int, []byte) {
	return c.do(http.MethodGet, path, token, nil)
}

func gjson(data []byte, path ...string) string {
	var v any
	if json.Unmarshal(data, &v) != nil {
		return ""
	}
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		v = m[k]
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

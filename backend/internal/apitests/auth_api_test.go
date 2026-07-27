package apitests_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/pkg/config"
	pgpkg "github.com/zoom/darsly/internal/pkg/postgres"
)

// ─── umumiy yordamchilar (faqat auth/user/lesson/joinlink testlari uchun) ──────

// clearRateLimits Redis'dagi rate-limit kalitlarini (rl:*) tozalaydi. Redis
// test'lar orasida truncate qilinmaydi, shuning uchun 127.0.0.1 IP bo'yicha
// auth/join hisoblagichi to'planib 429'ga olib kelmasligi uchun har test boshida
// tozalab qo'yamiz. Faqat rl:* kalitlarini o'chiradi — boshqa sessiyalarga xavfsiz.
func clearRateLimits(t *testing.T) {
	t.Helper()
	cache, err := redis.New(config.RedisConfig{
		Host: getenv("TEST_REDIS_HOST", "localhost"), Port: getenv("TEST_REDIS_PORT", "6399"),
	})
	if err != nil {
		return
	}
	_ = cache.ScanDel(context.Background(), "rl:*")
}

// mustRegister foydalanuvchini ro'yxatdan o'tkazadi va (access, refresh) qaytaradi.
func mustRegister(t *testing.T, cl *httpClient, name, email, pw string) (string, string) {
	t.Helper()
	code, body := cl.post("/api/v1/auth/register", "", map[string]string{
		"full_name": name, "email": email, "password": pw,
	})
	require.Equal(t, http.StatusCreated, code, "register muvaffaqiyatli bo'lishi kerak: %s", body)
	access := gjson(body, "data", "access_token")
	refresh := gjson(body, "data", "refresh_token")
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)
	return access, refresh
}

// promoteMentor foydalanuvchini DB darajasida mentor qiladi.
func promoteMentor(t *testing.T, pg *pgpkg.Postgres, email string) {
	t.Helper()
	_, err := pg.DB.Exec(context.Background(), "UPDATE users SET role='mentor' WHERE email=$1", email)
	require.NoError(t, err)
}

// loginToken login qilib access_token qaytaradi.
func loginToken(t *testing.T, cl *httpClient, email, pw string) string {
	t.Helper()
	code, body := cl.post("/api/v1/auth/login", "", map[string]string{"email": email, "password": pw})
	require.Equal(t, http.StatusOK, code, "login muvaffaqiyatli bo'lishi kerak: %s", body)
	tok := gjson(body, "data", "access_token")
	require.NotEmpty(t, tok)
	return tok
}

// ─── AUTH: success ─────────────────────────────────────────────────────────────

func TestAuthRegisterSuccess(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	code, body := cl.post("/api/v1/auth/register", "", map[string]string{
		"full_name": "Dilnoza", "email": "reg@darsly.uz", "password": "parol12345",
	})
	require.Equal(t, http.StatusCreated, code)
	require.NotEmpty(t, gjson(body, "data", "access_token"))
	require.NotEmpty(t, gjson(body, "data", "refresh_token"))

	// Yangi user default student rol bilan yaratiladi.
	access := gjson(body, "data", "access_token")
	code, body = cl.get("/api/v1/auth/me", access)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "student", gjson(body, "data", "role"))
	require.Equal(t, "reg@darsly.uz", gjson(body, "data", "email"))
}

func TestAuthLoginRefreshLogout(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Aziz", "flow@darsly.uz", "parol12345")

	// Login → yangi token juftligi.
	code, body := cl.post("/api/v1/auth/login", "", map[string]string{"email": "flow@darsly.uz", "password": "parol12345"})
	require.Equal(t, http.StatusOK, code)
	access := gjson(body, "data", "access_token")
	refresh := gjson(body, "data", "refresh_token")
	require.NotEmpty(t, access)
	require.NotEmpty(t, refresh)

	// Refresh → yangi token juftligi.
	code, body = cl.post("/api/v1/auth/refresh", "", map[string]string{"refresh_token": refresh})
	require.Equal(t, http.StatusOK, code)
	require.NotEmpty(t, gjson(body, "data", "access_token"))
	newRefresh := gjson(body, "data", "refresh_token")
	require.NotEmpty(t, newRefresh)

	// Logout (yangi refresh bilan) → 204.
	newAccess := gjson(body, "data", "access_token")
	code, _ = cl.post("/api/v1/auth/logout", newAccess, map[string]string{"refresh_token": newRefresh})
	require.Equal(t, http.StatusNoContent, code)
}

func TestAuthMe(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	access, _ := mustRegister(t, cl, "Kamola", "me@darsly.uz", "parol12345")
	code, body := cl.get("/api/v1/auth/me", access)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "me@darsly.uz", gjson(body, "data", "email"))
	require.Equal(t, "Kamola", gjson(body, "data", "full_name"))
}

// ─── AUTH: bad ─────────────────────────────────────────────────────────────────

func TestAuthRegisterWeakPassword(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	code, _ := cl.post("/api/v1/auth/register", "", map[string]string{
		"full_name": "Ali", "email": "weak@darsly.uz", "password": "123",
	})
	require.Equal(t, http.StatusBadRequest, code, "zaif parol (min=8) validatsiyadan o'tmasligi kerak")
}

func TestAuthRegisterDuplicateEmail(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Birinchi", "dup@darsly.uz", "parol12345")

	code, _ := cl.post("/api/v1/auth/register", "", map[string]string{
		"full_name": "Ikkinchi", "email": "dup@darsly.uz", "password": "parol12345",
	})
	require.Equal(t, http.StatusConflict, code, "mavjud email → 409 Conflict")
}

func TestAuthLoginWrongPassword(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Nodir", "wrongpw@darsly.uz", "parol12345")

	code, _ := cl.post("/api/v1/auth/login", "", map[string]string{"email": "wrongpw@darsly.uz", "password": "notaravalid"})
	require.Equal(t, http.StatusUnauthorized, code, "noto'g'ri parol → 401")
}

func TestAuthMeNoToken(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	code, _ := cl.get("/api/v1/auth/me", "")
	require.Equal(t, http.StatusUnauthorized, code, "token'siz → 401")
}

func TestAuthMeInvalidToken(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	code, _ := cl.get("/api/v1/auth/me", "obviously.not.a.jwt")
	require.Equal(t, http.StatusUnauthorized, code, "yaroqsiz token → 401")
}

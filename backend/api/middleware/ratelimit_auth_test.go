package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/pkg/token"
	"github.com/zoom/darsly/internal/testutil"
)

// BE-11: CGNAT ostida auth rate-limit. Asosiy talab — bitta IP ortidagi TURLI
// foydalanuvchilar bir-birini bloklamasin, LEKIN brute-force himoyasi kuchsizlanmasin.

// loginEngine — login limiter zanjiri + body'ni qaytarib beruvchi handler
// (limiter body'ni o'qib, handler uchun tiklashini ham tekshiradi).
func loginEngine(ipEmail, email, ip int64) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cache := testutil.NewFakeCache()
	h := append(RateLimitLogin(cache, ipEmail, email, ip, time.Minute), func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"got": string(body)})
	})
	r.POST("/api/v1/auth/login", h...)
	return r
}

func postLogin(r *gin.Engine, clientIP, email string) *httptest.ResponseRecorder {
	body := `{"email":"` + email + `","password":"parol12345"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = clientIP + ":12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// Limiter body'ni o'qiydi — handler'ga body BUZILMAGAN yetib borishi shart,
// aks holda barcha login/refresh so'rovlari 400 bo'lardi.
func TestRateLimitLogin_RestoresBodyForHandler(t *testing.T) {
	r := loginEngine(100, 100, 100)

	w := postLogin(r, "10.0.0.1", "a@x.uz")
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct{ Got string }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.JSONEq(t, `{"email":"a@x.uz","password":"parol12345"}`, resp.Got,
		"limiter body'ni o'qigandan keyin handler uchun tiklashi shart")
}

// CGNAT: bitta IP, TURLI foydalanuvchilar. Qattiq (IP+email) chegara bir foydalanuvchini
// to'sganda BOSHQA foydalanuvchi shu IP'dan baribir kira olishi kerak.
func TestRateLimitLogin_CGNAT_DifferentUsersNotBlocked(t *testing.T) {
	const perIPEmail = 3
	r := loginEngine(perIPEmail, 100, 1000)
	const ip = "100.64.0.1" // CGNAT diapazoni

	// 1-foydalanuvchi o'z chegarasini tugatadi.
	for i := 0; i < perIPEmail; i++ {
		require.Equal(t, http.StatusOK, postLogin(r, ip, "user1@x.uz").Code, "urinish %d", i+1)
	}
	require.Equal(t, http.StatusTooManyRequests, postLogin(r, ip, "user1@x.uz").Code,
		"o'z chegarasidan oshgan foydalanuvchi to'silishi kerak")

	// Ayni shu IP'dagi boshqa foydalanuvchilar ta'sirlanmasligi SHART (BE-11 mag'zi).
	for _, e := range []string{"user2@x.uz", "user3@x.uz", "user4@x.uz"} {
		require.Equal(t, http.StatusOK, postLogin(r, ip, e).Code,
			"CGNAT ortidagi boshqa foydalanuvchi (%s) bloklanmasligi kerak", e)
	}
}

// Nishonli brute-force: bitta akkauntga urinish qattiq cheklanadi (avval 60/min edi).
func TestRateLimitLogin_TargetedBruteForceBlocked(t *testing.T) {
	const perIPEmail = 10
	r := loginEngine(perIPEmail, 30, 200)

	for i := 0; i < perIPEmail; i++ {
		require.Equal(t, http.StatusOK, postLogin(r, "1.2.3.4", "victim@x.uz").Code)
	}
	w := postLogin(r, "1.2.3.4", "victim@x.uz")
	require.Equal(t, http.StatusTooManyRequests, w.Code, "nishonli parol-tanlash to'silishi kerak")
	require.NotEmpty(t, w.Header().Get("Retry-After"))
	assertErrorShape(t, w.Body.Bytes(), "RATE_LIMITED")
}

// TAQSIMLANGAN hujum: hujumchi IP almashtiradi, nishon bitta akkaunt.
// Avval bu umuman cheklanmagan edi (har IP o'z kvotasini olardi) — endi email
// bo'yicha chegara IP'dan qat'i nazar ishlaydi.
func TestRateLimitLogin_DistributedAttackOnOneAccountBlocked(t *testing.T) {
	const perEmail = 5
	r := loginEngine(100, perEmail, 1000) // IP+email va IP shifti ataylab baland

	for i := 0; i < perEmail; i++ {
		ip := "203.0.113." + strconv.Itoa(i+1) // har urinish boshqa IP
		require.Equal(t, http.StatusOK, postLogin(r, ip, "victim@x.uz").Code)
	}
	require.Equal(t, http.StatusTooManyRequests, postLogin(r, "203.0.113.99", "victim@x.uz").Code,
		"IP almashtirib bitta akkauntga hujum qilish to'silishi kerak")
}

// CREDENTIAL STUFFING: bitta IP, har safar BOSHQA email. Per-email kalit yolg'iz
// qolsa bu chetlab o'tilardi — IP shifti shuning uchun saqlanadi.
func TestRateLimitLogin_CredentialStuffingCappedByIP(t *testing.T) {
	const perIP = 6
	r := loginEngine(100, 100, perIP) // faqat IP shifti ishlaydigan holat

	for i := 0; i < perIP; i++ {
		email := "target" + strconv.Itoa(i) + "@x.uz"
		require.Equal(t, http.StatusOK, postLogin(r, "198.51.100.7", email).Code)
	}
	require.Equal(t, http.StatusTooManyRequests, postLogin(r, "198.51.100.7", "another@x.uz").Code,
		"bitta IP'dan ko'p akkauntga urinish IP shifti bilan to'silishi kerak")
}

// Email bo'lmasa ham (bo'sh/buzuq body) cheklovsiz teshik qolmasligi kerak —
// IP shifti har doim ishlaydi.
func TestRateLimitLogin_NoEmailStillCappedByIP(t *testing.T) {
	const perIP = 4
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cache := testutil.NewFakeCache()
	h := append(RateLimitLogin(cache, 100, 100, perIP, time.Minute), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	r.POST("/api/v1/auth/login", h...)

	send := func() int {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		req.RemoteAddr = "192.0.2.9:1111"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}
	for i := 0; i < perIP; i++ {
		require.Equal(t, http.StatusOK, send())
	}
	require.Equal(t, http.StatusTooManyRequests, send(), "email'siz so'rovlar ham IP bo'yicha cheklanishi kerak")
}

// ─── /auth/refresh ───────────────────────────────────────────────────────────

// refreshSecret — testda haqiqiy imzo uchun kalit (SessionFromRefresh Redis'ga tegmaydi,
// shuning uchun maker'ga nil redis klienti berish xavfsiz).
var refreshSecret = []byte("test-secret-at-least-32-chars-long-000")

func refreshMaker(t *testing.T) token.Maker {
	t.Helper()
	return token.NewJWTMaker(refreshSecret, 15*time.Minute, 720*time.Hour,
		token.DefaultRefreshGrace, nil, "rltest", testutil.NewLogger())
}

// signedRefresh — HAQIQIY imzolangan refresh token (limiter shundaylarni qabul qiladi).
func signedRefresh(t *testing.T, sid string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "u1", "sid": sid, "role": "mentor", "type": "refresh",
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Hour).Unix(),
		"jti": "jti-" + sid,
	})
	str, err := tok.SignedString(refreshSecret)
	require.NoError(t, err)
	return str
}

// forgedRefresh — QURBONNING sid'i, lekin SOXTA imzo. Aynan QA takrorlagan hujum:
// sid maxfiy emas (muddati tugagan access token/log/skrinshotdan o'qiladi).
func forgedRefresh(t *testing.T, victimSID string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "attacker", "sid": victimSID, "role": "mentor", "type": "refresh",
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	str, err := tok.SignedString([]byte("boshqa-kalit-hujumchida-haqiqiy-sir-yoq"))
	require.NoError(t, err)
	return str
}

func refreshEngine(t *testing.T, perSID, perIP int64) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cache := testutil.NewFakeCache()
	h := append(RateLimitRefresh(cache, refreshMaker(t), perSID, perIP, time.Minute), func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.JSON(http.StatusOK, gin.H{"got": string(body)})
	})
	r.POST("/api/v1/auth/refresh", h...)
	return r
}

func postRefresh(r *gin.Engine, clientIP, tok string) *httptest.ResponseRecorder {
	body := `{"refresh_token":"` + tok + `"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = clientIP + ":2222"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// 🔴 QA ssenariysi: hujumchi QURBONNING sid'i bilan soxta imzoli token yuborib
// qurbonning bucket'ini to'ldirishga urinadi. `refresh:sid` bucket'i ataylab IP'dan
// mustaqil bo'lgani uchun tekshirilmagan sid qabul qilinsa zarar QURBONGA tushardi
// (refresh 429 → frontend foydalanuvchini tizimdan chiqaradi).
//
// Endi imzo tekshirilgani uchun soxta token'ning sid'i bucket'ga UMUMAN kirmaydi.
func TestRateLimitRefresh_ForgedSidCannotPoisonVictimBucket(t *testing.T) {
	const perSID = 5
	// IP shifti hujumchini to'smasin — zarar QURBONGA tushmasligini ko'rsatmoqchimiz.
	r := refreshEngine(t, perSID, 10000)
	const victimSID = "a02c42df-1111-2222-3333-444455556666"

	// Hujumchi perSID'dan ANCHA ko'p so'rov yuboradi (boshqa IP'dan).
	forged := forgedRefresh(t, victimSID)
	for i := 0; i < perSID*4; i++ {
		code := postRefresh(r, "203.0.113.66", forged).Code
		// Soxta token limiterdan o'tadi (401'ni handler beradi) — muhimi 429 EMAS.
		require.NotEqual(t, http.StatusTooManyRequests, code,
			"soxta token qurbon bucket'ini to'ldirmasligi kerak (urinish %d)", i+1)
	}

	// ⇒ QURBONNING haqiqiy refresh'i BUZILMAGAN bo'lishi SHART.
	real := signedRefresh(t, victimSID)
	for i := 0; i < perSID; i++ {
		require.Equal(t, http.StatusOK, postRefresh(r, "100.64.1.1", real).Code,
			"qurbonning haqiqiy refresh'i ishlashi kerak (urinish %d)", i+1)
	}
}

// Haqiqiy sessiyaning O'Z limiti hamon ishlaydi — cheksiz refresh sikli to'siladi.
func TestRateLimitRefresh_RealSessionStillLimited(t *testing.T) {
	const perSID = 4
	r := refreshEngine(t, perSID, 10000)
	tok := signedRefresh(t, "aaaaaaaa-1111-2222-3333-444444444444")

	for i := 0; i < perSID; i++ {
		require.Equal(t, http.StatusOK, postRefresh(r, "10.0.0.1", tok).Code, "urinish %d", i+1)
	}
	w := postRefresh(r, "10.0.0.1", tok)
	require.Equal(t, http.StatusTooManyRequests, w.Code, "haqiqiy sessiya o'z chegarasidan oshsa to'silishi kerak")
	assertErrorShape(t, w.Body.Bytes(), "RATE_LIMITED")
}

// CGNAT regressiyasi: bitta IP, ikki xil HAQIQIY sessiya — biri 429 bo'lganda
// ikkinchisi 200 olishi shart (BE-11 ning butun maqsadi).
func TestRateLimitRefresh_PerSessionNotPerIP(t *testing.T) {
	const perSID = 3
	r := refreshEngine(t, perSID, 10000)
	const ip = "100.64.5.5" // CGNAT

	t1 := signedRefresh(t, "11111111-aaaa-bbbb-cccc-111111111111")
	for i := 0; i < perSID; i++ {
		require.Equal(t, http.StatusOK, postRefresh(r, ip, t1).Code, "urinish %d", i+1)
	}
	require.Equal(t, http.StatusTooManyRequests, postRefresh(r, ip, t1).Code,
		"bitta sessiya o'z chegarasidan oshsa to'silishi kerak")

	// Ayni IP'dagi boshqa sessiyalar ta'sirlanmaydi.
	for _, sid := range []string{
		"22222222-aaaa-bbbb-cccc-222222222222",
		"33333333-aaaa-bbbb-cccc-333333333333",
	} {
		require.Equal(t, http.StatusOK, postRefresh(r, ip, signedRefresh(t, sid)).Code,
			"CGNAT ortidagi boshqa sessiya (%s) bloklanmasligi kerak", sid)
	}
}

// Refresh limiter body'ni handler uchun tiklashi shart.
func TestRateLimitRefresh_RestoresBody(t *testing.T) {
	r := refreshEngine(t, 100, 10000)
	tok := signedRefresh(t, "44444444-aaaa-bbbb-cccc-444444444444")

	w := postRefresh(r, "10.1.1.1", tok)
	require.Equal(t, http.StatusOK, w.Code)

	var resp struct{ Got string }
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.JSONEq(t, `{"refresh_token":"`+tok+`"}`, resp.Got)
}

// Soxta/buzuq token bilan cheklovni chetlab o'tishga urinish hujumchining O'Z IP
// shifti bilan chegaralanadi (sid ajratilmasa ham teshik qolmaydi).
func TestRateLimitRefresh_GarbageTokenCappedByIP(t *testing.T) {
	const perIP = 5
	r := refreshEngine(t, 10000, perIP)

	for i := 0; i < perIP; i++ {
		require.Equal(t, http.StatusOK, postRefresh(r, "192.0.2.44", "buzuq-token").Code)
	}
	require.Equal(t, http.StatusTooManyRequests, postRefresh(r, "192.0.2.44", "buzuq-token-2").Code,
		"sid ajratilmagan so'rovlar IP bo'yicha cheklanishi kerak")
}

// Soxta imzoli tokenlar ham hujumchining O'Z IP shiftiga tushadi — ya'ni hujumchi
// cheklovsiz urinib, boshqalarga zarar ham bermaydi.
func TestRateLimitRefresh_ForgedTokenCappedByAttackerOwnIP(t *testing.T) {
	const perIP = 6
	r := refreshEngine(t, 10000, perIP)
	forged := forgedRefresh(t, "55555555-aaaa-bbbb-cccc-555555555555")

	for i := 0; i < perIP; i++ {
		require.Equal(t, http.StatusOK, postRefresh(r, "198.51.100.13", forged).Code)
	}
	require.Equal(t, http.StatusTooManyRequests, postRefresh(r, "198.51.100.13", forged).Code,
		"soxta tokenli hujum hujumchining o'z IP shifti bilan to'silishi kerak")
}

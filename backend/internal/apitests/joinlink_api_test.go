package apitests_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	api "github.com/zoom/darsly/api"
	"github.com/zoom/darsly/internal/app"
	"github.com/zoom/darsly/internal/infrastructure/email"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
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

// newTestServerLiveKit newTestServer bilan bir xil, faqat LiveKit yoqilgan (dummy
// kalitlar bilan). Token generatsiyasi mahalliy (tarmoqsiz) bo'lgani uchun
// "next_step=join" (kutish xonasisiz to'g'ridan-to'g'ri kirish) yo'lini sinash
// imkonini beradi — bu yo'l participant tokeni talab qiladi.
func newTestServerLiveKit(t *testing.T) (*httptest.Server, *pgpkg.Postgres) {
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
	// Siyosat binarga joylashtirilgan — testda yo'l qurish shart emas
	// (avval `rootDir()` bilan qidirilardi, bu esa nisbiy-yo'l muammosining
	// aynan o'zi edi).
	enforcer, err := casbin.NewEnforcer()
	require.NoError(t, err)

	hub := websocket.NewHub(log)
	// LiveKit yoqilgan — token imzolash uchun 32+ belgili secret.
	lk := livekit.New(config.LiveKitConfig{
		Host:      "ws://localhost:7880",
		APIKey:    "devkey",
		APISecret: "secret_at_least_32_characters_long_000000",
	})
	uc := usecase.New(usecase.Deps{
		Store: storage.New(pg), TokenMaker: tokenMaker, Hasher: hasher.New(4),
		Minio: minio.NewNop(), Cache: cache, Log: log, Hub: hub,
		EmailSender: email.NewNopSender(), LiveKit: lk, RefreshTTL: 720 * time.Hour,
		FrontendBaseURL: "http://frontend",
	})
	cfg := &config.Config{}
	cfg.App.AllowOpenRegistration = true
	cfg.App.FrontendBaseURL = "http://frontend"

	h := app.BuildHandler(uc, hub, lk, cfg)
	router := api.NewRouter(h, tokenMaker, enforcer, cache, log, func() error { return nil })
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv, pg
}

// Eslatma: createLesson helperi umumiy helpers_test.go faylida (backend-agent-2)
// e'lon qilingan va (lessonID, slug) qaytaradi.

// ─── JOINLINK: success ─────────────────────────────────────────────────────────

func TestJoinPreviewPublic(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "jp@darsly.uz", "parol12345")
	promoteMentor(t, pg, "jp@darsly.uz")
	tok := loginToken(t, cl, "jp@darsly.uz", "parol12345")
	_, slug := createLesson(t, cl, tok, map[string]any{"title": "Ochiq dars", "is_waiting_room_enabled": true})

	// Public preview — token'siz.
	code, body := cl.get("/api/v1/joinlink/"+slug, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "Dilnoza", gjson(body, "data", "mentor_name"))
	require.Equal(t, "Ochiq dars", gjson(body, "data", "title"))
}

func TestJoinWaitingRoom(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "jw@darsly.uz", "parol12345")
	promoteMentor(t, pg, "jw@darsly.uz")
	tok := loginToken(t, cl, "jw@darsly.uz", "parol12345")
	_, slug := createLesson(t, cl, tok, map[string]any{"title": "Kutish dars", "is_waiting_room_enabled": true})

	// Guest kiradi — kutish xonasiga tushadi.
	code, body := cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Aziz"})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "waiting_room", gjson(body, "data", "next_step"))
	require.NotEmpty(t, gjson(body, "data", "request_id"))
}

// TestJoinDirect — kutish xonasisiz dars → to'g'ridan-to'g'ri kirish (participant token).
func TestJoinDirect(t *testing.T) {
	srv, pg := newTestServerLiveKit(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "jd@darsly.uz", "parol12345")
	promoteMentor(t, pg, "jd@darsly.uz")
	tok := loginToken(t, cl, "jd@darsly.uz", "parol12345")
	_, slug := createLesson(t, cl, tok, map[string]any{"title": "To'g'ridan dars", "is_waiting_room_enabled": false})

	code, body := cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Aziz"})
	require.Equal(t, http.StatusOK, code, "body: %s", body)
	require.Equal(t, "join", gjson(body, "data", "next_step"))
	require.NotEmpty(t, gjson(body, "data", "room", "token"), "participant tokeni qaytishi kerak")
}

// TestJoinWithCorrectPasscode — parolli darsga to'g'ri parol bilan kirish.
func TestJoinWithCorrectPasscode(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "jpass@darsly.uz", "parol12345")
	promoteMentor(t, pg, "jpass@darsly.uz")
	tok := loginToken(t, cl, "jpass@darsly.uz", "parol12345")
	// Parol + kutish xonasi (LiveKit talab qilmaslik uchun kutish xonasi ON).
	_, slug := createLesson(t, cl, tok, map[string]any{
		"title": "Parolli dars", "passcode": "1234", "is_waiting_room_enabled": true,
	})

	code, body := cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Aziz", "passcode": "1234"})
	require.Equal(t, http.StatusOK, code, "to'g'ri parol qabul qilinishi kerak: %s", body)
	require.Equal(t, "waiting_room", gjson(body, "data", "next_step"))
}

// ─── JOINLINK: bad ─────────────────────────────────────────────────────────────

func TestJoinInvalidSlug(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	code, _ := cl.get("/api/v1/joinlink/nosuchslug123", "")
	require.Equal(t, http.StatusNotFound, code, "yaroqsiz slug preview → 404")

	code, _ = cl.post("/api/v1/joinlink/nosuchslug123", "", map[string]string{"guest_name": "Aziz"})
	require.Equal(t, http.StatusNotFound, code, "yaroqsiz slug join → 404")
}

func TestJoinWrongPasscode(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "jwp@darsly.uz", "parol12345")
	promoteMentor(t, pg, "jwp@darsly.uz")
	tok := loginToken(t, cl, "jwp@darsly.uz", "parol12345")
	_, slug := createLesson(t, cl, tok, map[string]any{
		"title": "Parolli dars", "passcode": "1234", "is_waiting_room_enabled": true,
	})

	code, _ := cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Aziz", "passcode": "0000"})
	require.Equal(t, http.StatusUnauthorized, code, "noto'g'ri parol → 401")
}

func TestJoinLocked(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "jl@darsly.uz", "parol12345")
	promoteMentor(t, pg, "jl@darsly.uz")
	tok := loginToken(t, cl, "jl@darsly.uz", "parol12345")
	id, slug := createLesson(t, cl, tok, map[string]any{"title": "Qulflanadigan", "is_waiting_room_enabled": true})

	// Darsni qulflaymiz.
	code, _ := cl.do(http.MethodPatch, "/api/v1/lessons/"+id, tok, map[string]any{"is_locked": true})
	require.Equal(t, http.StatusOK, code)

	// Qulflangan darsga kirishga urinish → 403.
	code, _ = cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Aziz"})
	require.Equal(t, http.StatusForbidden, code, "qulflangan dars → 403")
}

// №3 — yakunlangan dars havolasi: preview 200 (ma'lumot), join esa token
// bermaydi — next_step="lesson_ended" (xato emas, holat).
func TestJoinEndedLesson(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "je@darsly.uz", "parol12345")
	promoteMentor(t, pg, "je@darsly.uz")
	tok := loginToken(t, cl, "je@darsly.uz", "parol12345")
	id, slug := createLesson(t, cl, tok, map[string]any{"title": "Tugagan dars", "is_waiting_room_enabled": true})

	// Darsni yakunlaymiz (status PATCH orqali — LiveKit'siz yo'l).
	code, _ := cl.do(http.MethodPatch, "/api/v1/lessons/"+id, tok, map[string]any{"status": "ended"})
	require.Equal(t, http.StatusOK, code)

	// Preview — 200, status ichida (avval 400 edi).
	code, body := cl.get("/api/v1/joinlink/"+slug, "")
	require.Equal(t, http.StatusOK, code, "tugagan dars preview'i ochiq qoladi: %s", body)
	require.Equal(t, "ended", gjson(body, "data", "status"))
	require.Equal(t, "Tugagan dars", gjson(body, "data", "title"))

	// Join — 200 + lesson_ended, token YO'Q (waiting_room so'rovi ham yaratilmaydi).
	code, body = cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Kechikkan"})
	require.Equal(t, http.StatusOK, code, "body: %s", body)
	require.Equal(t, "lesson_ended", gjson(body, "data", "next_step"))
	require.Empty(t, gjson(body, "data", "room", "token"), "tugagan darsga token berilmasin")
	require.Empty(t, gjson(body, "data", "request_id"), "kutish so'rovi ham yaratilmasin")
	require.Equal(t, "ended", gjson(body, "data", "lesson", "status"))
}

// ⭐ Ovoz siyosati OCHIQ joinlink javobida bo'lishi kerak.
//
// O'quvchi mikrofon tugmasini shu qiymatlar bo'yicha chizadi. Avval ular faqat
// ustoz KLIENTIDAN data-message bilan kelardi va mobil ustoz uni umuman
// yubormasdi — telefondan o'tilgan darsda tugma yolg'on ko'rsatardi.
func TestJoinPreview_ReturnsAudioPolicy(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Dilnoza", "jpol@darsly.uz", "parol12345")
	promoteMentor(t, pg, "jpol@darsly.uz")
	tok := loginToken(t, cl, "jpol@darsly.uz", "parol12345")
	_, slug := createLesson(t, cl, tok, map[string]any{
		"title": "Ovoz siyosati", "mute_on_entry": true, "allow_self_unmute": false,
	})

	code, body := cl.get("/api/v1/joinlink/"+slug, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, true, jsonGet(body, "data", "mute_on_entry"), "preview: %s", body)
	require.Equal(t, false, jsonGet(body, "data", "allow_self_unmute"), "preview: %s", body)
}

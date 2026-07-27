package apitests_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/pkg/config"
	pgpkg "github.com/zoom/darsly/internal/pkg/postgres"
)

// newTestLiveKit — agar TEST_LIVEKIT_URL/KEY/SECRET o'rnatilgan bo'lsa real LiveKit
// klientini, aks holda o'chirilgan (nop) klientni qaytaradi.
func newTestLiveKit() *livekit.Client {
	host := os.Getenv("TEST_LIVEKIT_URL")
	key := os.Getenv("TEST_LIVEKIT_KEY")
	secret := os.Getenv("TEST_LIVEKIT_SECRET")
	if host == "" || key == "" || secret == "" {
		return livekit.New(config.LiveKitConfig{}) // nop (disabled)
	}
	return livekit.New(config.LiveKitConfig{Host: host, APIKey: key, APISecret: secret})
}

// liveKitAvailable — real LiveKit env berilganmi (success token yo'llarini gate qilish uchun).
func liveKitAvailable() bool {
	return os.Getenv("TEST_LIVEKIT_URL") != "" &&
		os.Getenv("TEST_LIVEKIT_KEY") != "" &&
		os.Getenv("TEST_LIVEKIT_SECRET") != ""
}

// ─── umumiy oqim helperlari ──────────────────────────────────────────────────

// registerStudent yangi foydalanuvchi (student) ro'yxatga oladi va access token qaytaradi.
func registerStudent(t *testing.T, cl *httpClient, name, email string) string {
	t.Helper()
	code, body := cl.post("/api/v1/auth/register", "", map[string]string{
		"full_name": name, "email": email, "password": "parol12345",
	})
	require.Equal(t, 201, code, "register: %s", body)
	tok := gjson(body, "data", "access_token")
	require.NotEmpty(t, tok)
	return tok
}

// makeMentorAndLogin foydalanuvchini mentor qiladi va yangi (mentor claim'li) token qaytaradi.
func makeMentorAndLogin(t *testing.T, cl *httpClient, pg *pgpkg.Postgres, email string) string {
	t.Helper()
	_, err := pg.DB.Exec(context.Background(), "UPDATE users SET role='mentor' WHERE email=$1", email)
	require.NoError(t, err)
	code, body := cl.post("/api/v1/auth/login", "", map[string]string{"email": email, "password": "parol12345"})
	require.Equal(t, 200, code, "login: %s", body)
	tok := gjson(body, "data", "access_token")
	require.NotEmpty(t, tok)
	return tok
}

// registerMentor — register + mentor qilish + qayta login (bitta qadamda).
func registerMentor(t *testing.T, cl *httpClient, pg *pgpkg.Postgres, name, email string) string {
	t.Helper()
	registerStudent(t, cl, name, email)
	return makeMentorAndLogin(t, cl, pg, email)
}

// registerAdmin — register + admin qilish + qayta login (bitta qadamda).
// User-management endi faqat admin'da (H-2), shu rolni sinash uchun.
func registerAdmin(t *testing.T, cl *httpClient, pg *pgpkg.Postgres, name, email string) string {
	t.Helper()
	registerStudent(t, cl, name, email)
	_, err := pg.DB.Exec(context.Background(), "UPDATE users SET role='admin' WHERE email=$1", email)
	require.NoError(t, err)
	code, body := cl.post("/api/v1/auth/login", "", map[string]string{"email": email, "password": "parol12345"})
	require.Equal(t, 200, code, "login: %s", body)
	tok := gjson(body, "data", "access_token")
	require.NotEmpty(t, tok)
	return tok
}

// userID — email bo'yicha foydalanuvchi UUID sini DB'dan oladi.
func userID(t *testing.T, pg *pgpkg.Postgres, email string) string {
	t.Helper()
	var id string
	err := pg.DB.QueryRow(context.Background(), "SELECT id FROM users WHERE email=$1", email).Scan(&id)
	require.NoError(t, err)
	return id
}

// createLesson mentor sifatida dars yaratadi, (lessonID, slug) qaytaradi.
func createLesson(t *testing.T, cl *httpClient, mentorTok string, payload map[string]any) (string, string) {
	t.Helper()
	code, body := cl.post("/api/v1/lessons", mentorTok, payload)
	require.Equal(t, 201, code, "createLesson: %s", body)
	id := gjson(body, "data", "id")
	slug := gjson(body, "data", "join_slug")
	require.NotEmpty(t, id)
	require.NotEmpty(t, slug)
	return id, slug
}

// setLessonLive darsni to'g'ridan-to'g'ri DB orqali 'live' holatiga o'tkazadi
// (recording testlari uchun — LiveKit'siz jonli holat kerak).
func setLessonLive(t *testing.T, pg *pgpkg.Postgres, lessonID string) {
	t.Helper()
	_, err := pg.DB.Exec(context.Background(),
		"UPDATE lessons SET status='live', started_at=NOW() WHERE id=$1", lessonID)
	require.NoError(t, err)
}

// ─── JSON o'qish yordamchilari (gjson faqat string qaytaradi) ─────────────────

// jsonGet umumiy qiymatni (har qanday tip) yo'l bo'yicha qaytaradi.
func jsonGet(data []byte, path ...string) any {
	var v any
	if json.Unmarshal(data, &v) != nil {
		return nil
	}
	for _, k := range path {
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[k]
	}
	return v
}

// jsonInt son maydonini int sifatida qaytaradi (JSON raqamlari float64).
func jsonInt(data []byte, path ...string) int {
	if f, ok := jsonGet(data, path...).(float64); ok {
		return int(f)
	}
	return -1
}

// jsonLen massiv uzunligini qaytaradi (yo'q/massiv emas → 0).
func jsonLen(data []byte, path ...string) int {
	if arr, ok := jsonGet(data, path...).([]any); ok {
		return len(arr)
	}
	return 0
}

// jsonArr massiv elementlarini qaytaradi.
func jsonArr(data []byte, path ...string) []any {
	if arr, ok := jsonGet(data, path...).([]any); ok {
		return arr
	}
	return nil
}

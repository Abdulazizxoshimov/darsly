package apitests_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// BE-5: `GET /api/v1/app-config` ochiq (auth'siz) bo'lishi va mobil klient kutgan
// shaklni qaytarishi shart. Router orqali — route haqiqatan `protected` guruhidan
// TASHQARIDA ro'yxatga olinganini tasdiqlaydi.
func TestAppConfig_PublicAndShape(t *testing.T) {
	srv, _ := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	code, body := cl.get("/api/v1/app-config", "") // token YO'Q
	require.Equal(t, http.StatusOK, code, "app-config auth talab qilmasligi kerak: %s", body)

	require.Equal(t, "1.0.0", gjson(body, "data", "android", "min_version"))
	require.Equal(t, "1.1.0", gjson(body, "data", "android", "latest_version"))
	require.Equal(t, "https://example.test/darsly-mentor.apk", gjson(body, "data", "android", "apk_url"))

	// force_update bool bo'lishi shart (klient shuni tekshiradi).
	var parsed struct {
		Data struct {
			Android struct {
				ForceUpdate  bool   `json:"force_update"`
				ReleaseNotes string `json:"release_notes"`
			} `json:"android"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.False(t, parsed.Data.Android.ForceUpdate)
	require.Equal(t, "", parsed.Data.Android.ReleaseNotes)
}

// BE-4: to'liq router orqali xato javobi YAGONA {code, message} shaklida bo'lishi shart
// (middleware'lar avval {error, code} qaytarardi → klientda matn bo'sh chiqardi).
func TestErrorEnvelope_IsUniform(t *testing.T) {
	srv, _ := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	assertEnvelope := func(t *testing.T, body []byte, wantCode string) {
		t.Helper()
		var m map[string]any
		require.NoError(t, json.Unmarshal(body, &m), "javob JSON bo'lishi kerak: %s", body)
		require.Equal(t, wantCode, m["code"], "kod mos kelishi kerak: %s", body)
		msg, ok := m["message"].(string)
		require.True(t, ok, "`message` string bo'lishi kerak: %s", body)
		require.NotEmpty(t, msg, "`message` bo'sh bo'lmasligi kerak: %s", body)
		_, hasErr := m["error"]
		require.False(t, hasErr, "eskirgan `error` maydoni qaytmasligi kerak: %s", body)
	}

	t.Run("401 token yo'q (Auth middleware)", func(t *testing.T) {
		code, body := cl.get("/api/v1/auth/me", "")
		require.Equal(t, http.StatusUnauthorized, code)
		assertEnvelope(t, body, "UNAUTHORIZED")
	})

	t.Run("401 token yaroqsiz (Auth middleware)", func(t *testing.T) {
		code, body := cl.get("/api/v1/auth/me", "aniq.yaroqsiz.token")
		require.Equal(t, http.StatusUnauthorized, code)
		assertEnvelope(t, body, "TOKEN_INVALID")
	})

	t.Run("403 RBAC (EnforceCasbin middleware)", func(t *testing.T) {
		student := registerStudent(t, cl, "Shakl", "shape-403@darsly.uz")
		code, body := cl.get("/api/v1/lessons", student) // student'ga ruxsat yo'q
		require.Equal(t, http.StatusForbidden, code)
		assertEnvelope(t, body, "FORBIDDEN")
	})

	t.Run("404 handler (hs.Error) — bir xil shakl", func(t *testing.T) {
		mentor := registerStudent(t, cl, "Shakl2", "shape-404@darsly.uz")
		code, body := cl.get("/api/v1/users/00000000-0000-0000-0000-000000000000", mentor)
		require.Contains(t, []int{http.StatusNotFound, http.StatusForbidden}, code)
		var m map[string]any
		require.NoError(t, json.Unmarshal(body, &m))
		require.NotEmpty(t, m["code"])
		require.NotEmpty(t, m["message"])
		_, hasErr := m["error"]
		require.False(t, hasErr)
	})
}

// BE-2 (uchdan-uchga): mobil tarmoqda rotatsiya javobi yo'qolgan holat — klient AYNI
// o'sha refresh bilan qayta uradi. Grace oynasi ichida ikkala so'rov ham 200 va BIR XIL
// juftlik qaytarishi, mavjud sessiya esa TIRIK qolishi shart.
func TestRefresh_GraceWindow_DoesNotKillSessions(t *testing.T) {
	srv, _ := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	code, body := cl.post("/api/v1/auth/register", "", map[string]string{
		"full_name": "Mobil", "email": "grace@darsly.uz", "password": "parol12345",
	})
	require.Equal(t, http.StatusCreated, code, "%s", body)
	access := gjson(body, "data", "access_token")
	refresh := gjson(body, "data", "refresh_token")
	require.NotEmpty(t, refresh)

	// 1-refresh: muvaffaqiyatli (javob "yo'qolgan" deb tasavvur qilamiz).
	code, body1 := cl.post("/api/v1/auth/refresh", "", map[string]string{"refresh_token": refresh})
	require.Equal(t, http.StatusOK, code, "%s", body1)
	newAccess := gjson(body1, "data", "access_token")
	newRefresh := gjson(body1, "data", "refresh_token")
	require.NotEmpty(t, newRefresh)

	// 2-refresh: AYNI o'sha (eski) refresh bilan qayta urinish → 200 + bir xil juftlik.
	code, body2 := cl.post("/api/v1/auth/refresh", "", map[string]string{"refresh_token": refresh})
	require.Equal(t, http.StatusOK, code, "grace oynasida takroriy refresh 401 bermasligi kerak: %s", body2)
	require.Equal(t, newAccess, gjson(body2, "data", "access_token"))
	require.Equal(t, newRefresh, gjson(body2, "data", "refresh_token"))

	// Sessiyalar tirik: eski access ham, yangi access ham ishlaydi.
	code, _ = cl.get("/api/v1/auth/me", access)
	require.Equal(t, http.StatusOK, code, "grace holatida eski access token o'lmasligi kerak")
	code, _ = cl.get("/api/v1/auth/me", newAccess)
	require.Equal(t, http.StatusOK, code, "yangi access token ishlashi kerak")

	// Grace'dan qaytgan refresh keyingi normal rotatsiyada ham ishlaydi.
	code, body3 := cl.post("/api/v1/auth/refresh", "", map[string]string{"refresh_token": newRefresh})
	require.Equal(t, http.StatusOK, code, "%s", body3)
	require.NotEqual(t, newRefresh, gjson(body3, "data", "refresh_token"))
}

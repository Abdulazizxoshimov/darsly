package apitests_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRoomToken_Success (LiveKit kerak): owner mentor host token oladi.
func TestRoomToken_Success(t *testing.T) {
	if !liveKitAvailable() {
		t.Skip("TEST_LIVEKIT yo'q — host token success yo'li skip")
	}
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "Host M", "rm_host@darsly.uz")
	lessonID, _ := createLesson(t, cl, mentorTok, map[string]any{"title": "Algebra"})

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/token", mentorTok, nil)
	require.Equal(t, http.StatusOK, code, "host token: %s", body)
	require.NotEmpty(t, gjson(body, "data", "token"))
	require.NotEmpty(t, gjson(body, "data", "ws_url"))
	require.Equal(t, "host", gjson(body, "data", "role"))
}

// TestRoomToken_BadCases: begona mentor → 403, mavjud emas dars → 404, token'siz → 401.
// (Egalik/mavjudlik tekshiruvi LiveKit'dan oldin bo'lgani uchun LiveKit'siz ham ishlaydi.)
func TestRoomToken_BadCases(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "Owner M", "rm_owner@darsly.uz")
	otherTok := registerMentor(t, cl, pg, "Other M", "rm_other@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "Geometriya"})

	// Begona mentor host token so'raydi → 403 (ownership).
	code, _ := cl.post("/api/v1/lessons/"+lessonID+"/token", otherTok, nil)
	require.Equal(t, http.StatusForbidden, code, "begona mentor token → 403")

	// Mavjud emas dars → 404.
	code, _ = cl.post("/api/v1/lessons/00000000-0000-0000-0000-000000000000/token", ownerTok, nil)
	require.Equal(t, http.StatusNotFound, code, "mavjud emas dars token → 404")

	// Token'siz → 401.
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/token", "", nil)
	require.Equal(t, http.StatusUnauthorized, code, "token'siz → 401")

	// Student token so'raydi → 403 (RBAC).
	studentTok := registerStudent(t, cl, "Std M", "rm_student@darsly.uz")
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/token", studentTok, nil)
	require.Equal(t, http.StatusForbidden, code, "student token → 403 (RBAC)")
}

// TestRoom_EndLesson: owner darsni yakunlaydi → 204; begona mentor → 403.
func TestRoom_EndLesson(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "End Owner", "rm_end_owner@darsly.uz")
	otherTok := registerMentor(t, cl, pg, "End Other", "rm_end_other@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "Tarix"})

	// Begona mentor → 403.
	code, _ := cl.post("/api/v1/lessons/"+lessonID+"/end", otherTok, nil)
	require.Equal(t, http.StatusForbidden, code, "begona mentor end → 403")

	// Owner → 204 (LiveKit yo'q bo'lsa ham: DeleteRoom faqat enabled bo'lsa chaqiriladi).
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/end", ownerTok, nil)
	require.Equal(t, http.StatusNoContent, code, "owner end → 204")
}

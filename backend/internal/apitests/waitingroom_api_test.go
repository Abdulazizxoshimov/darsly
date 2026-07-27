package apitests_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// guestJoinWaiting darsga kutish-xonasi orqali kiradi va request_id qaytaradi.
func guestJoinWaiting(t *testing.T, cl *httpClient, slug, guestName string) string {
	t.Helper()
	code, body := cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": guestName})
	require.Equal(t, http.StatusOK, code, "join: %s", body)
	require.Equal(t, "waiting_room", gjson(body, "data", "next_step"))
	rid := gjson(body, "data", "request_id")
	require.NotEmpty(t, rid)
	return rid
}

// TestWaitingRoom_HappyPath: join → mentor ro'yxatda ko'radi → public status pending →
// mentor reject → status rejected.
func TestWaitingRoom_HappyPath(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "Mentor A", "wr_mentor@darsly.uz")
	lessonID, slug := createLesson(t, cl, mentorTok, map[string]any{
		"title": "Fizika", "is_waiting_room_enabled": true,
	})

	// Guest kutish xonasiga so'rov yuboradi.
	rid := guestJoinWaiting(t, cl, slug, "Aziz")

	// Mentor kutayotganlar ro'yxatini oladi → so'rov bor.
	code, body := cl.get("/api/v1/lessons/"+lessonID+"/waitingroom", mentorTok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 1, jsonLen(body, "data"), "kutayotganlar ro'yxatida 1 so'rov: %s", body)
	first := jsonArr(body, "data")[0].(map[string]any)
	require.Equal(t, rid, first["id"])
	require.Equal(t, "pending", first["status"])
	require.Equal(t, "Aziz", first["requester_name"])

	// Public status (ochiq, token'siz) → pending.
	code, body = cl.get("/api/v1/waitingroom/"+rid+"/status", "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "pending", gjson(body, "data", "status"))

	// Mentor rad etadi → 204.
	code, _ = cl.post("/api/v1/waitingroom/"+rid+"/reject", mentorTok, nil)
	require.Equal(t, http.StatusNoContent, code)

	// Public status → rejected.
	code, body = cl.get("/api/v1/waitingroom/"+rid+"/status", "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "rejected", gjson(body, "data", "status"))

	// Ikkinchi reject → 409 (atomik TransitionFromPending — allaqachon hal qilingan).
	code, _ = cl.post("/api/v1/waitingroom/"+rid+"/reject", mentorTok, nil)
	require.Equal(t, http.StatusConflict, code, "qayta reject → 409 (atomik)")

	_ = pg
}

// TestWaitingRoom_BadCases: begona mentor admit → 403, mavjud emas request → 404.
func TestWaitingRoom_BadCases(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "Owner", "wr_owner@darsly.uz")
	otherTok := registerMentor(t, cl, pg, "Other", "wr_other@darsly.uz")
	_, slug := createLesson(t, cl, ownerTok, map[string]any{
		"title": "Kimyo", "is_waiting_room_enabled": true,
	})
	rid := guestJoinWaiting(t, cl, slug, "Bek")

	// Begona mentor (darsning egasi emas) admit qila olmaydi → 403.
	code, _ := cl.post("/api/v1/waitingroom/"+rid+"/admit", otherTok, nil)
	require.Equal(t, http.StatusForbidden, code, "begona mentor admit → 403")

	// Begona mentor reject ham 403.
	code, _ = cl.post("/api/v1/waitingroom/"+rid+"/reject", otherTok, nil)
	require.Equal(t, http.StatusForbidden, code, "begona mentor reject → 403")

	// Mavjud emas request admit → 404.
	code, _ = cl.post("/api/v1/waitingroom/00000000-0000-0000-0000-000000000000/admit", ownerTok, nil)
	require.Equal(t, http.StatusNotFound, code, "mavjud emas request admit → 404")

	// Mavjud emas request status (public) → 404.
	code, _ = cl.get("/api/v1/waitingroom/00000000-0000-0000-0000-000000000000/status", "")
	require.Equal(t, http.StatusNotFound, code, "mavjud emas request status → 404")

	// Student (mentor emas) admit qila olmaydi → 403 (RBAC middleware).
	studentTok := registerStudent(t, cl, "Student", "wr_student@darsly.uz")
	code, _ = cl.post("/api/v1/waitingroom/"+rid+"/admit", studentTok, nil)
	require.Equal(t, http.StatusForbidden, code, "student admit → 403 (RBAC)")

	// Token'siz admit → 401.
	code, _ = cl.post("/api/v1/waitingroom/"+rid+"/admit", "", nil)
	require.Equal(t, http.StatusUnauthorized, code, "token'siz admit → 401")

	_ = pg
}

// TestWaitingRoom_AdmitTOCTOU: ketma-ket ikki admit — atomik TransitionFromPending
// ikkinchisini 409 bilan rad etadi (LiveKit bor bo'lsa birinchisi 200 token).
func TestWaitingRoom_AdmitTOCTOU(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "Mentor T", "wr_toctou@darsly.uz")
	_, slug := createLesson(t, cl, mentorTok, map[string]any{
		"title": "Biologiya", "is_waiting_room_enabled": true,
	})
	rid := guestJoinWaiting(t, cl, slug, "Dilnoza")

	code1, body1 := cl.post("/api/v1/waitingroom/"+rid+"/admit", mentorTok, nil)
	code2, _ := cl.post("/api/v1/waitingroom/"+rid+"/admit", mentorTok, nil)

	// Ikkinchi admit har doim 409 (birinchisi statusni allaqachon 'admitted' qildi).
	require.Equal(t, http.StatusConflict, code2, "2-admit → 409 (atomik)")

	if liveKitAvailable() {
		require.Equal(t, http.StatusOK, code1, "LiveKit bor: 1-admit → 200 token")
		require.NotEmpty(t, gjson(body1, "data", "token"), "admit token bo'sh emas")
		require.Equal(t, "participant", gjson(body1, "data", "role"))
		// Admit'dan keyin public status → admitted + room token.
		code, body := cl.get("/api/v1/waitingroom/"+rid+"/status", "")
		require.Equal(t, http.StatusOK, code)
		require.Equal(t, "admitted", gjson(body, "data", "status"))
		require.NotEmpty(t, gjson(body, "data", "room", "token"))
	} else {
		// LiveKit yo'q: token bosqichida 500 (video servis o'chirilgan), lekin
		// transition atomik bajarilgani sabab 2-admit baribir 409.
		require.Equal(t, http.StatusInternalServerError, code1,
			"LiveKit yo'q: 1-admit token bosqichida 500 (video disabled)")
	}
}

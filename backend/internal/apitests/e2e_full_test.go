package apitests_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestE2E_WaitingRoomAdmitChain: to'liq zanjir — register → mentor → dars (kutish xonasi)
// → guest join (waiting) → mentor ro'yxatda ko'radi → admit → (LiveKit bor bo'lsa) token.
func TestE2E_WaitingRoomAdmitChain(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "E2E Mentor", "e2e_wr_mentor@darsly.uz")
	lessonID, slug := createLesson(t, cl, mentorTok, map[string]any{
		"title": "E2E kutish xonasi", "is_waiting_room_enabled": true,
	})

	rid := guestJoinWaiting(t, cl, slug, "Guest 1")

	// Mentor kutayotganlar ro'yxatida ko'radi.
	code, body := cl.get("/api/v1/lessons/"+lessonID+"/waitingroom", mentorTok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 1, jsonLen(body, "data"))

	// Admit.
	code, body = cl.post("/api/v1/waitingroom/"+rid+"/admit", mentorTok, nil)
	if liveKitAvailable() {
		require.Equal(t, http.StatusOK, code, "admit: %s", body)
		require.NotEmpty(t, gjson(body, "data", "token"))
		require.Equal(t, "participant", gjson(body, "data", "role"))
	} else {
		require.Equal(t, http.StatusInternalServerError, code,
			"LiveKit yo'q: admit token bosqichida 500 (video disabled)")
	}

	// Admit'dan keyin ro'yxat bo'sh (endi pending emas).
	code, body = cl.get("/api/v1/lessons/"+lessonID+"/waitingroom", mentorTok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 0, jsonLen(body, "data"), "admit'dan keyin pending ro'yxat bo'sh")

	_ = pg
}

// TestE2E_PasscodeJoin: parolli dars — noto'g'ri parol → 401, to'g'ri parol → next_step.
func TestE2E_PasscodeJoin(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "E2E Pass", "e2e_pass_mentor@darsly.uz")
	_, slug := createLesson(t, cl, mentorTok, map[string]any{
		"title": "Parolli dars", "is_waiting_room_enabled": true, "passcode": "secret12",
	})

	// Preview → has_passcode true.
	code, body := cl.get("/api/v1/joinlink/"+slug, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, true, jsonGet(body, "data", "has_passcode"), "preview parol talab qiladi: %s", body)

	// Parolsiz join → 401.
	code, _ = cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "X"})
	require.Equal(t, http.StatusUnauthorized, code, "parolsiz join → 401")

	// Noto'g'ri parol → 401.
	code, _ = cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "X", "passcode": "wrong999"})
	require.Equal(t, http.StatusUnauthorized, code, "noto'g'ri parol → 401")

	// To'g'ri parol → kutish xonasiga o'tadi (waiting_room + request_id).
	code, body = cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "X", "passcode": "secret12"})
	require.Equal(t, http.StatusOK, code, "to'g'ri parol: %s", body)
	require.Equal(t, "waiting_room", gjson(body, "data", "next_step"))
	require.NotEmpty(t, gjson(body, "data", "request_id"))
}

// TestE2E_DirectJoinNoWaitingRoom: kutish xonasisiz dars — to'g'ridan-to'g'ri kirish.
// next_step=join + participant token (LiveKit kerak). LiveKit yo'q bo'lsa token bosqichi 500.
func TestE2E_DirectJoinNoWaitingRoom(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "E2E Direct", "e2e_direct_mentor@darsly.uz")
	_, slug := createLesson(t, cl, mentorTok, map[string]any{
		"title": "To'g'ridan-to'g'ri dars", "is_waiting_room_enabled": false,
	})

	code, body := cl.post("/api/v1/joinlink/"+slug, "", map[string]string{"guest_name": "Direct Guest"})
	if liveKitAvailable() {
		require.Equal(t, http.StatusOK, code, "direct join: %s", body)
		require.Equal(t, "join", gjson(body, "data", "next_step"))
		require.NotEmpty(t, gjson(body, "data", "room", "token"))
		require.Equal(t, "participant", gjson(body, "data", "room", "role"))
	} else {
		require.Equal(t, http.StatusInternalServerError, code,
			"LiveKit yo'q: to'g'ridan-to'g'ri join token bosqichida 500 (video disabled)")
	}
}

// TestE2E_MultiGuestSequentialJoin: bir necha guest ketma-ket kutish xonasiga kiradi,
// hammasi mentor ro'yxatida ko'rinadi.
func TestE2E_MultiGuestSequentialJoin(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "E2E Multi", "e2e_multi_mentor@darsly.uz")
	lessonID, slug := createLesson(t, cl, mentorTok, map[string]any{
		"title": "Ko'p guest", "is_waiting_room_enabled": true,
	})

	names := []string{"Ali", "Vali", "Guli"}
	ids := make([]string, 0, len(names))
	for _, n := range names {
		ids = append(ids, guestJoinWaiting(t, cl, slug, n))
	}

	// Mentor ro'yxatida 3 ta pending.
	code, body := cl.get("/api/v1/lessons/"+lessonID+"/waitingroom", mentorTok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 3, jsonLen(body, "data"), "3 ta pending: %s", body)

	// Bittasini reject → ro'yxatda 2 qoladi.
	code, _ = cl.post("/api/v1/waitingroom/"+ids[0]+"/reject", mentorTok, nil)
	require.Equal(t, http.StatusNoContent, code)

	code, body = cl.get("/api/v1/lessons/"+lessonID+"/waitingroom", mentorTok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 2, jsonLen(body, "data"), "reject'dan keyin 2 pending")
}

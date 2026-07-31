package apitests_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// Xona holati endpointlari ikki xil autentifikatsiyaga ega:
//   · ochiq yo'l (`/rooms/...`)   — LiveKit ROOM-TOKEN (guest'da JWT yo'q)
//   · host yo'li (`/lessons/...`) — JWT + RBAC + egalik
// Bu testlar aynan shu chegarani qo'riqlaydi: ochiq endpointlar tokensiz
// o'tmasligi, host endpointlari esa begona mentorga berilmasligi kerak.

// TestRoomState_OpenEndpointsRequireRoomToken: token UMUMAN berilmasa → 401.
//
// Bu tekshiruv LiveKit'siz ham ishlaydi va ataylab shunday tartiblangan:
// "token yo'q" degan xulosa uchun imzoni tekshirish shart emas. Soxta tokenni
// rad etish esa imzo tekshiruvini talab qiladi — u pastdagi gated testda.
func TestRoomState_OpenEndpointsRequireRoomToken(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "RS Host", "rs_host@darsly.uz")
	lessonID, _ := createLesson(t, cl, mentorTok, map[string]any{"title": "Fizika"})

	code, body := cl.get("/api/v1/rooms/"+lessonID+"/state", "")
	require.Equal(t, http.StatusUnauthorized, code, "tokensiz state → 401: %s", body)

	code, body = cl.post("/api/v1/rooms/"+lessonID+"/hand", "", map[string]any{"raised": true})
	require.Equal(t, http.StatusUnauthorized, code, "tokensiz qo'l → 401: %s", body)

	code, _ = cl.post("/api/v1/rooms/"+lessonID+"/reaction", "", map[string]any{"emoji": "👍"})
	require.Equal(t, http.StatusUnauthorized, code, "tokensiz reaksiya → 401")
}

// TestRoomState_RejectsForgedToken (LiveKit kerak): imzosi yaroqsiz token → 401.
func TestRoomState_RejectsForgedToken(t *testing.T) {
	if !liveKitAvailable() {
		t.Skip("TEST_LIVEKIT yo'q — imzo tekshiruvi skip")
	}
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "RS Forge", "rs_forge@darsly.uz")
	lessonID, _ := createLesson(t, cl, mentorTok, map[string]any{"title": "Fizika 2"})

	code, body := cl.post("/api/v1/rooms/"+lessonID+"/hand", "", map[string]any{
		"token": "soxta-token", "raised": true,
	})
	require.Equal(t, http.StatusUnauthorized, code, "soxta token → 401: %s", body)

	code, _ = cl.get("/api/v1/rooms/"+lessonID+"/state?token=soxta-token", "")
	require.Equal(t, http.StatusUnauthorized, code, "soxta token bilan state → 401")
}

// TestRoomState_ValidationBeforeAuthLeaks: bo'sh tana / bo'sh emoji → 400,
// ya'ni validatsiya ishlaydi va 500 generatoriga aylanmaydi.
func TestRoomState_BadPayload(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "RS Bad", "rs_bad@darsly.uz")
	lessonID, _ := createLesson(t, cl, mentorTok, map[string]any{"title": "Kimyo"})

	// Emoji bo'sh → 400. Bog'lash validatsiyasi TOKEN tekshiruvidan oldin ishlaydi,
	// shuning uchun bu holat LiveKit'siz ham aniq 400 beradi.
	code, body := cl.post("/api/v1/rooms/"+lessonID+"/reaction", "", map[string]any{"token": "x", "emoji": ""})
	require.Equal(t, http.StatusBadRequest, code, "bo'sh emoji → 400: %s", body)

	// Emoji juda uzun (emoji emas, matn) → 400.
	code, _ = cl.post("/api/v1/rooms/"+lessonID+"/reaction", "", map[string]any{
		"token": "x", "emoji": "bu emoji emas, uzun matn",
	})
	require.Equal(t, http.StatusBadRequest, code, "uzun emoji → 400")
}

// TestRoomState_HostEndpointsRBAC: qo'l tushirish faqat mentor rolida va faqat
// dars egasida ishlaydi.
func TestRoomState_HostEndpointsRBAC(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "RS Owner", "rs_owner@darsly.uz")
	otherTok := registerMentor(t, cl, pg, "RS Other", "rs_other@darsly.uz")
	studentTok := registerStudent(t, cl, "RS Std", "rs_std@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "Biologiya"})

	body := map[string]any{"identity": "guest-1"}

	// Token'siz → 401.
	code, _ := cl.post("/api/v1/lessons/"+lessonID+"/hands/lower", "", body)
	require.Equal(t, http.StatusUnauthorized, code, "tokensiz → 401")

	// Student → 403 (RBAC: policy'da faqat mentor).
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/hands/lower", studentTok, body)
	require.Equal(t, http.StatusForbidden, code, "student qo'l tushira olmaydi → 403")

	// Begona mentor → 403 (egalik).
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/hands/lower", otherTok, body)
	require.Equal(t, http.StatusForbidden, code, "begona mentor → 403")
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/hands/lower-all", otherTok, nil)
	require.Equal(t, http.StatusForbidden, code, "begona mentor lower-all → 403")

	// Ega → 204 (hech kim qo'l ko'tarmagan bo'lsa ham amal muvaffaqiyatli:
	// "yo'q edi" va "o'chirildi" farqi chaqiruvchi uchun ahamiyatsiz).
	code, respBody := cl.post("/api/v1/lessons/"+lessonID+"/hands/lower", ownerTok, body)
	require.Equal(t, http.StatusNoContent, code, "ega qo'l tushiradi → 204: %s", respBody)
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/hands/lower-all", ownerTok, nil)
	require.Equal(t, http.StatusNoContent, code, "ega hammasini tushiradi → 204")
}

// TestRoomState_HandFlow (LiveKit kerak): host token oladi → o'sha token bilan
// qo'l ko'taradi → holatda ko'rinadi → tushiradi → yo'qoladi.
//
// Bu YAGONA joyda qo'lning to'liq aylanishi HTTP orqali tekshiriladi: real
// room-token, real Redis, real broadcast yo'li.
func TestRoomState_HandFlow(t *testing.T) {
	if !liveKitAvailable() {
		t.Skip("TEST_LIVEKIT yo'q — room-token bilan to'liq oqim skip")
	}
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "RS Flow", "rs_flow@darsly.uz")
	lessonID, _ := createLesson(t, cl, mentorTok, map[string]any{"title": "Ona tili"})

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/token", mentorTok, nil)
	require.Equal(t, http.StatusOK, code, "host token: %s", body)
	roomToken := gjson(body, "data", "token")
	require.NotEmpty(t, roomToken)
	require.Equal(t, lessonID, gjson(body, "data", "lesson_id"), "room token dars ID'sini qaytarishi kerak")

	// Qo'l ko'tarish.
	code, body = cl.post("/api/v1/rooms/"+lessonID+"/hand", "", map[string]any{
		"token": roomToken, "raised": true,
	})
	require.Equal(t, http.StatusNoContent, code, "qo'l ko'tarish: %s", body)

	// Holatda ko'rinadi.
	code, body = cl.get("/api/v1/rooms/"+lessonID+"/state?token="+roomToken, "")
	require.Equal(t, http.StatusOK, code, "state: %s", body)
	require.Contains(t, body, "raised_at", "qo'l yozuvi qaytishi kerak")

	// Tushirish.
	code, _ = cl.post("/api/v1/rooms/"+lessonID+"/hand", "", map[string]any{
		"token": roomToken, "raised": false,
	})
	require.Equal(t, http.StatusNoContent, code)

	code, body = cl.get("/api/v1/rooms/"+lessonID+"/state?token="+roomToken, "")
	require.Equal(t, http.StatusOK, code)
	require.NotContains(t, body, "raised_at", "tushirilgan qo'l holatda qolmasligi kerak")
}

// ⭐ TestRoomState_AudioPolicyLive (LiveKit kerak): xona holati ovoz siyosatini
// qaytaradi va u dars O'RTASIDAGI o'zgarishdan keyin YANGILANADI.
//
// Bu kech ulangan/qayta ulangan o'quvchi uchun yagona ishonchli manba:
// `mute-all` bilan yuborilgan data-message bir martalik, uni o'tkazib yuborgan
// klient siyosatni boshqa hech qayerdan bila olmasdi.
func TestRoomState_AudioPolicyLive(t *testing.T) {
	if !liveKitAvailable() {
		t.Skip("TEST_LIVEKIT yo'q — room-token bilan holat oqimi skip")
	}
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	mentorTok := registerMentor(t, cl, pg, "RS Policy", "rs_policy@darsly.uz")
	lessonID, _ := createLesson(t, cl, mentorTok, map[string]any{
		"title": "Ovoz", "mute_on_entry": true, "allow_self_unmute": true,
	})

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/token", mentorTok, nil)
	require.Equal(t, http.StatusOK, code, "host token: %s", body)
	roomToken := gjson(body, "data", "token")

	code, body = cl.get("/api/v1/rooms/"+lessonID+"/state?token="+roomToken, "")
	require.Equal(t, http.StatusOK, code, "state: %s", body)
	require.Equal(t, true, jsonGet(body, "data", "mute_on_entry"), "state: %s", body)
	require.Equal(t, true, jsonGet(body, "data", "allow_self_unmute"), "state: %s", body)

	// Ustoz dars o'rtasida "hammani o'chirish + o'zi ocholmasin" qiladi.
	code, body = cl.post("/api/v1/lessons/"+lessonID+"/mute-all", mentorTok,
		map[string]any{"allow_self_unmute": false})
	require.Equal(t, http.StatusNoContent, code, "mute-all: %s", body)

	code, body = cl.get("/api/v1/rooms/"+lessonID+"/state?token="+roomToken, "")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, false, jsonGet(body, "data", "allow_self_unmute"),
		"o'zgargan siyosat holatda ko'rinishi kerak: %s", body)
}

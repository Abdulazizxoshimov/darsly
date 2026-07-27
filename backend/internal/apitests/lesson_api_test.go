package apitests_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// ─── LESSON: success ───────────────────────────────────────────────────────────

func TestLessonCreateSuccess(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Ustoz", "lc@darsly.uz", "parol12345")
	promoteMentor(t, pg, "lc@darsly.uz")
	tok := loginToken(t, cl, "lc@darsly.uz", "parol12345")

	code, body := cl.post("/api/v1/lessons", tok, map[string]any{
		"title": "Matematika", "is_waiting_room_enabled": true,
	})
	require.Equal(t, http.StatusCreated, code)
	require.NotEmpty(t, gjson(body, "data", "join_slug"))
	require.Equal(t, "Matematika", gjson(body, "data", "title"))
	require.Equal(t, "scheduled", gjson(body, "data", "status"))
}

func TestLessonList(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Ustoz", "ll@darsly.uz", "parol12345")
	promoteMentor(t, pg, "ll@darsly.uz")
	tok := loginToken(t, cl, "ll@darsly.uz", "parol12345")

	for _, title := range []string{"Fizika", "Kimyo"} {
		code, _ := cl.post("/api/v1/lessons", tok, map[string]any{"title": title, "is_waiting_room_enabled": true})
		require.Equal(t, http.StatusCreated, code)
	}

	code, _ := cl.get("/api/v1/lessons", tok)
	require.Equal(t, http.StatusOK, code)
}

func TestLessonGetByIDOwner(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Ustoz", "lg@darsly.uz", "parol12345")
	promoteMentor(t, pg, "lg@darsly.uz")
	tok := loginToken(t, cl, "lg@darsly.uz", "parol12345")

	_, body := cl.post("/api/v1/lessons", tok, map[string]any{"title": "Biologiya", "is_waiting_room_enabled": true})
	id := gjson(body, "data", "id")
	require.NotEmpty(t, id)

	code, body := cl.get("/api/v1/lessons/"+id, tok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "Biologiya", gjson(body, "data", "title"))
}

func TestLessonUpdate(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Ustoz", "lu@darsly.uz", "parol12345")
	promoteMentor(t, pg, "lu@darsly.uz")
	tok := loginToken(t, cl, "lu@darsly.uz", "parol12345")

	_, body := cl.post("/api/v1/lessons", tok, map[string]any{"title": "Eski nom", "is_waiting_room_enabled": true})
	id := gjson(body, "data", "id")

	code, body := cl.do(http.MethodPatch, "/api/v1/lessons/"+id, tok, map[string]any{"title": "Yangi nom"})
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "Yangi nom", gjson(body, "data", "title"))
}

func TestLessonDelete(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Ustoz", "ld@darsly.uz", "parol12345")
	promoteMentor(t, pg, "ld@darsly.uz")
	tok := loginToken(t, cl, "ld@darsly.uz", "parol12345")

	_, body := cl.post("/api/v1/lessons", tok, map[string]any{"title": "O'chiriladi", "is_waiting_room_enabled": true})
	id := gjson(body, "data", "id")

	code, _ := cl.do(http.MethodDelete, "/api/v1/lessons/"+id, tok, nil)
	require.Equal(t, http.StatusNoContent, code)

	// O'chirilgan dars endi topilmaydi.
	code, _ = cl.get("/api/v1/lessons/"+id, tok)
	require.Equal(t, http.StatusNotFound, code)
}

// ─── LESSON: bad ───────────────────────────────────────────────────────────────

func TestLessonCreateStudentForbidden(t *testing.T) {
	srv, _ := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	access, _ := mustRegister(t, cl, "Talaba", "ls@darsly.uz", "parol12345")
	code, _ := cl.post("/api/v1/lessons", access, map[string]any{"title": "Ruxsatsiz"})
	require.Equal(t, http.StatusForbidden, code, "student dars yarata olmaydi (RBAC)")
}

func TestLessonCreateEmptyTitle(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Ustoz", "le@darsly.uz", "parol12345")
	promoteMentor(t, pg, "le@darsly.uz")
	tok := loginToken(t, cl, "le@darsly.uz", "parol12345")

	code, _ := cl.post("/api/v1/lessons", tok, map[string]any{"title": ""})
	require.Equal(t, http.StatusBadRequest, code, "bo'sh title → 400")
}

func TestLessonGetNotFound(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	_, _ = mustRegister(t, cl, "Ustoz", "lnf@darsly.uz", "parol12345")
	promoteMentor(t, pg, "lnf@darsly.uz")
	tok := loginToken(t, cl, "lnf@darsly.uz", "parol12345")

	code, _ := cl.get("/api/v1/lessons/"+uuid.NewString(), tok)
	require.Equal(t, http.StatusNotFound, code, "mavjud bo'lmagan dars → 404")
}

// TestLessonGetOtherMentorForbidden — boshqa mentor darsini olishga urinish → 403 (ownership/IDOR).
func TestLessonGetOtherMentorForbidden(t *testing.T) {
	srv, pg := newTestServer(t)
	clearRateLimits(t)
	cl := &httpClient{t: t, base: srv.URL}

	// 1-mentor dars yaratadi.
	_, _ = mustRegister(t, cl, "Mentor1", "m1@darsly.uz", "parol12345")
	promoteMentor(t, pg, "m1@darsly.uz")
	tok1 := loginToken(t, cl, "m1@darsly.uz", "parol12345")
	_, body := cl.post("/api/v1/lessons", tok1, map[string]any{"title": "Yopiq", "is_waiting_room_enabled": true})
	id := gjson(body, "data", "id")
	require.NotEmpty(t, id)

	// 2-mentor uni olishga urinadi → 403.
	_, _ = mustRegister(t, cl, "Mentor2", "m2@darsly.uz", "parol12345")
	promoteMentor(t, pg, "m2@darsly.uz")
	tok2 := loginToken(t, cl, "m2@darsly.uz", "parol12345")
	code, _ := cl.get("/api/v1/lessons/"+id, tok2)
	require.Equal(t, http.StatusForbidden, code, "boshqa mentor darsi → 403 (ownership)")
}

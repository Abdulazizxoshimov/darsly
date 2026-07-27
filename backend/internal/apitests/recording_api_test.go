package apitests_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	pgpkg "github.com/zoom/darsly/internal/pkg/postgres"
)

// insertRecording DB'ga to'g'ridan-to'g'ri yozuv qo'shadi va uning ID sini qaytaradi.
func insertRecording(t *testing.T, pg *pgpkg.Postgres, lessonID, status string) string {
	t.Helper()
	var id string
	err := pg.DB.QueryRow(context.Background(),
		`INSERT INTO recordings (lesson_id, egress_id, object_key, status)
		 VALUES ($1, 'eg_'||gen_random_uuid(), $2, $3) RETURNING id`,
		lessonID, "recordings/"+lessonID+"/x.mp4", status).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestRecording_StartBadCases: student → 403 (RBAC), begona mentor → 403 (ownership).
func TestRecording_StartBadCases(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "Rec Owner", "rec_owner@darsly.uz")
	otherTok := registerMentor(t, cl, pg, "Rec Other", "rec_other@darsly.uz")
	studentTok := registerStudent(t, cl, "Rec Std", "rec_student@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "Yozuv darsi"})

	// Student yozib olishni boshlay olmaydi → 403 (RBAC middleware).
	code, _ := cl.post("/api/v1/lessons/"+lessonID+"/recording/start", studentTok, nil)
	require.Equal(t, http.StatusForbidden, code, "student recording start → 403 (RBAC)")

	// Begona mentor → 403 (ownership, LiveKit'dan oldin tekshiriladi).
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/recording/start", otherTok, nil)
	require.Equal(t, http.StatusForbidden, code, "begona mentor recording start → 403")

	// Token'siz → 401.
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/recording/start", "", nil)
	require.Equal(t, http.StatusUnauthorized, code, "token'siz → 401")

	// Mavjud emas dars → 404 (ownership tekshiruvida).
	code, _ = cl.post("/api/v1/lessons/00000000-0000-0000-0000-000000000000/recording/start", ownerTok, nil)
	require.Equal(t, http.StatusNotFound, code, "mavjud emas dars recording → 404")
}

// TestRecording_StartSuccess (LiveKit+Egress kerak). Egress bo'lmasa skip.
// Eslatma: webhook (egress_ended → ready/failed) yo'li backend'da alohida imzo-tekshiruvli
// integratsiyada sinalgan (bu yerda faqat start API si).
func TestRecording_StartSuccess(t *testing.T) {
	if !liveKitAvailable() {
		t.Skip("TEST_LIVEKIT/Egress yo'q — recording start success skip (webhook yo'li alohida sinalgan)")
	}
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "Rec S Owner", "rec_s_owner@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "Yozuv jonli"})
	setLessonLive(t, pg, lessonID)

	code, body := cl.post("/api/v1/lessons/"+lessonID+"/recording/start", ownerTok, nil)
	require.Equal(t, http.StatusCreated, code, "recording start: %s", body)
	require.Equal(t, "recording", gjson(body, "data", "status"))
}

// TestRecording_ListAndDownload: list (owner/begona), download (mavjud emas/tayyor emas/begona).
func TestRecording_ListAndDownload(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "RL Owner", "rl_owner@darsly.uz")
	otherTok := registerMentor(t, cl, pg, "RL Other", "rl_other@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "Yozuvlar"})

	// DB'ga ikkita yozuv qo'shamiz.
	recActive := insertRecording(t, pg, lessonID, "recording")
	insertRecording(t, pg, lessonID, "ready")

	// Owner ro'yxatni oladi → 2 yozuv.
	code, body := cl.get("/api/v1/lessons/"+lessonID+"/recordings", ownerTok)
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, 2, jsonLen(body, "data"), "owner 2 yozuv ko'radi: %s", body)

	// Begona mentor ro'yxat → 403.
	code, _ = cl.get("/api/v1/lessons/"+lessonID+"/recordings", otherTok)
	require.Equal(t, http.StatusForbidden, code, "begona mentor list → 403")

	// Mavjud emas yozuv download → 404.
	code, _ = cl.get("/api/v1/recordings/00000000-0000-0000-0000-000000000000/download", ownerTok)
	require.Equal(t, http.StatusNotFound, code, "mavjud emas yozuv download → 404")

	// Tayyor bo'lmagan (recording) yozuv download → 400 (not ready).
	code, _ = cl.get("/api/v1/recordings/"+recActive+"/download", ownerTok)
	require.Equal(t, http.StatusBadRequest, code, "tayyor emas yozuv download → 400")

	// Begona mentor download → 403 (ownership).
	code, _ = cl.get("/api/v1/recordings/"+recActive+"/download", otherTok)
	require.Equal(t, http.StatusForbidden, code, "begona mentor download → 403")
}

// TestRecording_StopBadCases: mavjud emas → 404, begona mentor → 403, faol emas → 400.
func TestRecording_StopBadCases(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "RS Owner", "rs_owner@darsly.uz")
	otherTok := registerMentor(t, cl, pg, "RS Other", "rs_other@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "To'xtatish"})

	// Mavjud emas yozuv stop → 404.
	code, _ := cl.post("/api/v1/recordings/00000000-0000-0000-0000-000000000000/stop", ownerTok, nil)
	require.Equal(t, http.StatusNotFound, code, "mavjud emas yozuv stop → 404")

	// Faol yozuvni begona mentor stop → 403.
	recActive := insertRecording(t, pg, lessonID, "recording")
	code, _ = cl.post("/api/v1/recordings/"+recActive+"/stop", otherTok, nil)
	require.Equal(t, http.StatusForbidden, code, "begona mentor stop → 403")

	// Faol bo'lmagan (ready) yozuvni stop → 400.
	recReady := insertRecording(t, pg, lessonID, "ready")
	code, _ = cl.post("/api/v1/recordings/"+recReady+"/stop", ownerTok, nil)
	require.Equal(t, http.StatusBadRequest, code, "faol emas yozuv stop → 400")
}

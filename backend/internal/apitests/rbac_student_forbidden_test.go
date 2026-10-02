package apitests_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// B-3 [LOW] — student (haqiqiy token) mentor-only endpointlarga kira olmasligini
// API darajasida tasdiqlaydi (RBAC middleware). Ushlaydigan xato: policy.csv da
// student uchun bu yo'llar ochilib qolsa, o'quvchi darsni yozib olishni boshlash
// yoki darsni tugatish huquqini olardi.
func TestRBAC_StudentForbiddenOnMentorEndpoints(t *testing.T) {
	srv, pg := newTestServer(t)
	cl := &httpClient{t: t, base: srv.URL}

	ownerTok := registerMentor(t, cl, pg, "RBAC Owner", "rbac_owner@darsly.uz")
	studentTok := registerStudent(t, cl, "RBAC Std", "rbac_student@darsly.uz")
	lessonID, _ := createLesson(t, cl, ownerTok, map[string]any{"title": "RBAC darsi"})

	// Student recording start → 403 (RBAC middleware, biznes-logikadan oldin).
	code, _ := cl.post("/api/v1/lessons/"+lessonID+"/recording/start", studentTok, nil)
	require.Equal(t, http.StatusForbidden, code, "student recording start → 403 (RBAC)")

	// Student darsni tugatish → 403.
	code, _ = cl.post("/api/v1/lessons/"+lessonID+"/end", studentTok, nil)
	require.Equal(t, http.StatusForbidden, code, "student lesson end → 403 (RBAC)")
}

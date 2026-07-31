package poll_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// guestToken — o'quvchining room-token'idagi xona nomi.
const guestRoom = "lesson_" + testLessonID

// №7 — DEFAULT yopiq tomonda: `results_visibility` yuborilmasa `mentor_only`.
// Eski klientlar natijani tasodifan ochib yubormasin.
func TestCreate_DefaultMentorOnly(t *testing.T) {
	uc := setup(t)
	p, err := uc.Create(context.Background(), "mentor1", testLessonID, "Q", []string{"A", "B"}, "")
	require.NoError(t, err)
	require.Equal(t, entity.PollResultsMentorOnly, p.ResultsVisibility)
	require.Nil(t, p.ResultsPublishedAt)
}

func TestCreate_NotogriRejim400(t *testing.T) {
	uc := setup(t)
	_, err := uc.Create(context.Background(), "mentor1", testLessonID, "Q", []string{"A", "B"}, "hammaga")
	require.True(t, apperr.IsBadRequest(err), "kutilgan 400, olindi: %v", err)
}

// E'LON QILINMAGAN natijani o'quvchi OLA OLMAYDI — hatto `public` rejimda ham.
func TestResults_ElonQilinmaganGuestOlaOlmaydi(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"}, entity.PollResultsPublic)
	require.NoError(t, err)
	require.NoError(t, uc.Vote(ctx, p.ID, "guest_a", guestRoom, 0))

	_, err = uc.Results(ctx, p.ID, "guest_a", guestRoom)
	require.True(t, apperr.IsForbidden(err), "e'lon qilinmagan natija berilmasligi kerak, olindi: %v", err)

	// MENTOR esa doim ko'radi (host identity = mentorID).
	res, err := uc.Results(ctx, p.ID, "mentor1", guestRoom)
	require.NoError(t, err)
	require.Equal(t, 1, res.Total)
}

// E'lon qilingach o'quvchi oladi.
func TestPublish_ElonQilingachGuestOladi(t *testing.T) {
	uc, lk := setupLK(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"}, entity.PollResultsPublic)
	require.NoError(t, err)
	require.NoError(t, uc.Vote(ctx, p.ID, "guest_a", guestRoom, 1))

	res, err := uc.Publish(ctx, "mentor1", testLessonID, p.ID)
	require.NoError(t, err)
	require.NotNil(t, res.Poll.ResultsPublishedAt)
	require.Equal(t, []int{0, 1}, res.Counts)

	got, err := uc.Results(ctx, p.ID, "guest_a", guestRoom)
	require.NoError(t, err)
	require.Equal(t, 1, got.Total)

	// Xonaga hodisa ketishi kerak: 300 kishilik xonada har biri alohida
	// so'rov yuborsa bu 300 ta ortiqcha so'rov bo'lardi.
	require.GreaterOrEqual(t, len(lk.Sent), 1)
	var ev map[string]any
	require.NoError(t, json.Unmarshal(lk.Sent[len(lk.Sent)-1], &ev))
	require.Equal(t, "poll_published", ev["kind"])
	require.NotNil(t, ev["results"])
}

// MENTOR_ONLY — e'lon qilib BO'LMAYDI va natija o'quvchiga hech qachon chiqmaydi.
// Rejim yaratishda ovoz bergan o'quvchiga berilgan va'da; uni bir tugma bilan
// buzish mumkin bo'lsa rejimning ma'nosi qolmaydi.
func TestPublish_MentorOnlyElonQilibBolmaydi(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"}, entity.PollResultsMentorOnly)
	require.NoError(t, err)
	require.NoError(t, uc.Vote(ctx, p.ID, "guest_a", guestRoom, 0))

	_, err = uc.Publish(ctx, "mentor1", testLessonID, p.ID)
	require.True(t, apperr.IsBadRequest(err), "kutilgan 400, olindi: %v", err)

	// O'quvchi baribir ololmaydi.
	_, err = uc.Results(ctx, p.ID, "guest_a", guestRoom)
	require.True(t, apperr.IsForbidden(err))
}

// YOPISH ≠ E'LON QILISH: yopilgan so'rovnoma natijasi ham avtomatik ochilmaydi.
func TestClose_ElonQilmaydi(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"}, entity.PollResultsPublic)
	require.NoError(t, err)

	res, err := uc.Close(ctx, "mentor1", p.ID)
	require.NoError(t, err)
	require.False(t, res.Poll.IsActive)
	require.Nil(t, res.Poll.ResultsPublishedAt, "yopish e'lon qilmasligi kerak")

	_, err = uc.Results(ctx, p.ID, "guest_a", guestRoom)
	require.True(t, apperr.IsForbidden(err), "yopilgan, lekin e'lon qilinmagan natija ko'rinmasligi kerak")
}

func TestPublish_BegonaMentor403(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"}, entity.PollResultsPublic)
	require.NoError(t, err)

	_, err = uc.Publish(ctx, "intruder", testLessonID, p.ID)
	require.True(t, apperr.IsForbidden(err), "kutilgan 403, olindi: %v", err)
}

// Yo'ldagi dars ID si poll'niki bilan mos kelishi SHART — begona darsning yo'li
// orqali murojaat qilib bo'lmasin.
func TestPublish_BoshqaDarsYoli404(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"}, entity.PollResultsPublic)
	require.NoError(t, err)

	other := "22222222-2222-4222-8222-222222222222"
	_, err = uc.Publish(ctx, "mentor1", other, p.ID)
	require.True(t, apperr.IsNotFound(err), "kutilgan 404, olindi: %v", err)
}

// Takroriy e'lon idempotent: vaqt surilib ketmaydi.
func TestPublish_Idempotent(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"}, entity.PollResultsPublic)
	require.NoError(t, err)

	first, err := uc.Publish(ctx, "mentor1", testLessonID, p.ID)
	require.NoError(t, err)
	second, err := uc.Publish(ctx, "mentor1", testLessonID, p.ID)
	require.NoError(t, err)
	require.Equal(t, *first.Poll.ResultsPublishedAt, *second.Poll.ResultsPublishedAt)
}

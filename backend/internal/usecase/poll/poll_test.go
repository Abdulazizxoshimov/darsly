package poll_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/poll"
)

// testLessonID — haqiqiy UUID: usecase ID formatini tekshiradi (shared.ValidateID).
const testLessonID = "11111111-1111-4111-8111-111111111111"

func setup(t *testing.T) poll.UseCase {
	t.Helper()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive}))
	return poll.New(testutil.NewFakePollRepo(), lrepo, testutil.NewLogger())
}

func TestPoll_Flow(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()

	p, err := uc.Create(ctx, "mentor1", testLessonID, "Tushunarli bo'ldimi?", []string{"Ha", "Yo'q", "Qisman"})
	require.NoError(t, err)
	require.True(t, p.IsActive)
	require.Len(t, p.Options, 3)

	// Ovozlar (upsert — bir ishtirokchi bitta ovoz). Room-token darsga bog'langan: "lesson_"+testLessonID.
	require.NoError(t, uc.Vote(ctx, p.ID, "guest_a", "lesson_"+testLessonID, 0))
	require.NoError(t, uc.Vote(ctx, p.ID, "guest_b", "lesson_"+testLessonID, 0))
	require.NoError(t, uc.Vote(ctx, p.ID, "guest_c", "lesson_"+testLessonID, 2))
	require.NoError(t, uc.Vote(ctx, p.ID, "guest_a", "lesson_"+testLessonID, 1)) // guest_a fikrini o'zgartirdi

	res, err := uc.Results(ctx, p.ID)
	require.NoError(t, err)
	require.Equal(t, 3, res.Total, "3 ta noyob ovoz")
	require.Equal(t, []int{1, 1, 1}, res.Counts, "guest_a→1, guest_b→0, guest_c→2")
}

func TestPoll_Vote_InvalidOptionAndClosed(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, _ := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"})

	require.Error(t, uc.Vote(ctx, p.ID, "g", "lesson_"+testLessonID, 5), "diapazondan tashqari variant rad etiladi")

	_, err := uc.Close(ctx, "mentor1", p.ID)
	require.NoError(t, err)
	require.Error(t, uc.Vote(ctx, p.ID, "g", "lesson_"+testLessonID, 0), "yopilgan so'rovnomaga ovoz berib bo'lmaydi")
}

func TestPoll_Vote_CrossLessonRejected(t *testing.T) {
	uc := setup(t)
	ctx := context.Background()
	p, err := uc.Create(ctx, "mentor1", testLessonID, "Q", []string{"A", "B"})
	require.NoError(t, err)

	// Begona darsning room-tokeni (yoki o'z darsi host tokeni) bilan ovoz — rad etiladi.
	err = uc.Vote(ctx, p.ID, "attacker", "lesson_OTHER", 0)
	require.True(t, apperr.IsForbidden(err), "begona dars tokeni bilan ovoz berib bo'lmaydi")

	// To'g'ri dars tokeni bilan — o'tadi.
	require.NoError(t, uc.Vote(ctx, p.ID, "legit", "lesson_"+testLessonID, 0))
}

func TestPoll_Ownership(t *testing.T) {
	uc := setup(t)
	_, err := uc.Create(context.Background(), "intruder", testLessonID, "Q", []string{"A", "B"})
	require.True(t, apperr.IsForbidden(err))
}

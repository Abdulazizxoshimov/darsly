package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
)

// Yopilgan poll'ga ovoz DB darajasida rad etiladi (usecase'dagi o'qish-keyin-yozish poygasi).
func TestPollRepo_Vote_ClosedPollRejected(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	polls := pgRepo.NewPollRepo(pg)
	mentor := makeUser(t, users, "poll_closed@darsly.uz")

	l := &entity.Lesson{ID: uuid.NewString(), MentorID: mentor.ID, Title: "P", DurationMin: 60,
		JoinSlug: "poll-closed-001", Status: entity.LessonStatusLive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, lessons.Create(ctx(), l))
	p := &entity.Poll{ID: uuid.NewString(), LessonID: l.ID, Question: "Q?",
		Options: []string{"A", "B"}, IsActive: true, CreatedAt: time.Now().UTC(),
		ResultsVisibility: entity.PollResultsPublic}
	require.NoError(t, polls.Create(ctx(), p))

	require.NoError(t, polls.Vote(ctx(), p.ID, "ali", 0))
	require.NoError(t, polls.Close(ctx(), p.ID))

	err := polls.Vote(ctx(), p.ID, "vali", 1)
	require.True(t, apperr.IsBadRequest(err), "yopilgan poll'ga yangi ovoz rad etilishi kerak: %v", err)
	// Mavjud ovozni ham o'zgartirib bo'lmaydi.
	require.True(t, apperr.IsBadRequest(polls.Vote(ctx(), p.ID, "ali", 1)))

	counts, err := polls.Counts(ctx(), p.ID, 2)
	require.NoError(t, err)
	require.Equal(t, []int{1, 0}, counts)
}

// Mentor WS snapshot: faqat scheduled/live darslardagi pending'lar (ended — "ghost" emas).
func TestWaitingRoomRepo_PendingSnapshot_SkipsEndedLessons(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	waiting := pgRepo.NewWaitingRoomRepo(pg)
	mentor := makeUser(t, users, "snap@darsly.uz")

	mk := func(slug, status string) *entity.Lesson {
		l := &entity.Lesson{ID: uuid.NewString(), MentorID: mentor.ID, Title: slug, DurationMin: 60,
			JoinSlug: slug, Status: status, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		require.NoError(t, lessons.Create(ctx(), l))
		require.NoError(t, waiting.Create(ctx(), &entity.WaitingRoomRequest{ID: uuid.NewString(), LessonID: l.ID,
			RequesterName: "G", GuestIdentity: "guest_" + slug, Status: entity.WaitingStatusPending, CreatedAt: time.Now().UTC()}))
		return l
	}
	live := mk("snap-live-001", entity.LessonStatusLive)
	mk("snap-ended-001", entity.LessonStatusEnded)

	got, err := waiting.ListPendingByMentor(ctx(), mentor.ID)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, live.ID, got[0].LessonID)

	n, err := waiting.CountPendingByLesson(ctx(), live.ID)
	require.NoError(t, err)
	require.Equal(t, 1, n)
}

package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	"github.com/zoom/darsly/internal/testutil"
)

// B-5: poll.go repozitoriysida DB testi yo'q edi. Vote va Publish ning
// idempotentligi FAQAT HAQIQIY SQL'da (ON CONFLICT / COALESCE) ko'rinadi.

// Vote `ON CONFLICT (poll_id, voter_identity) DO UPDATE` bilan ishlaydi:
// bitta ishtirokchi qayta ovoz bersa — variant YANGILANADI, ikki marta
// SANALMAYDI. Ushlaydigan xato: ON CONFLICT tushib qolsa yo unique-violation
// yoki qo'shaloq ovoz paydo bo'lardi (natijalar buziladi).
func TestPollRepo_Vote_UpsertNoDoubleCount(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	polls := pgRepo.NewPollRepo(pg)
	mentor := makeUser(t, users, "poll_vote@darsly.uz")

	l := &entity.Lesson{ID: uuid.NewString(), MentorID: mentor.ID, Title: "P", DurationMin: 60,
		JoinSlug: "poll-vote-001", Status: entity.LessonStatusLive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, lessons.Create(ctx(), l))

	p := &entity.Poll{ID: uuid.NewString(), LessonID: l.ID, Question: "Q?",
		Options: []string{"A", "B", "C"}, IsActive: true, CreatedAt: time.Now().UTC(),
		ResultsVisibility: entity.PollResultsPublic}
	require.NoError(t, polls.Create(ctx(), p))

	// Ali variant 0 ga ovoz beradi.
	require.NoError(t, polls.Vote(ctx(), p.ID, "ali", 0))
	counts, err := polls.Counts(ctx(), p.ID, 3)
	require.NoError(t, err)
	require.Equal(t, []int{1, 0, 0}, counts)

	// Ali FIKRINI o'zgartiradi → variant 2 (upsert, qo'shaloq emas).
	require.NoError(t, polls.Vote(ctx(), p.ID, "ali", 2))
	counts, err = polls.Counts(ctx(), p.ID, 3)
	require.NoError(t, err)
	require.Equal(t, []int{0, 0, 1}, counts, "qayta ovoz yangilanishi kerak, ikki marta sanalmasin")

	// Vali alohida ovoz beradi → jami 2 ta ishtirokchi.
	require.NoError(t, polls.Vote(ctx(), p.ID, "vali", 2))
	counts, err = polls.Counts(ctx(), p.ID, 3)
	require.NoError(t, err)
	require.Equal(t, []int{0, 0, 2}, counts)
}

// Publish `COALESCE(results_published_at, NOW())` bilan idempotent: ikki marta
// e'lon qilinsa BIRINCHI vaqt saqlanadi. Ushlaydigan xato: COALESCE tushib qolsa
// e'lon vaqti har bosishda surilib ketardi (o'quvchiga «yangi» natija ko'rinardi).
func TestPollRepo_Publish_Idempotent(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	polls := pgRepo.NewPollRepo(pg)
	mentor := makeUser(t, users, "poll_pub@darsly.uz")

	l := &entity.Lesson{ID: uuid.NewString(), MentorID: mentor.ID, Title: "P", DurationMin: 60,
		JoinSlug: "poll-pub-001", Status: entity.LessonStatusLive,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	require.NoError(t, lessons.Create(ctx(), l))

	p := &entity.Poll{ID: uuid.NewString(), LessonID: l.ID, Question: "Q?",
		Options: []string{"A", "B"}, IsActive: true, CreatedAt: time.Now().UTC(),
		ResultsVisibility: entity.PollResultsPublic}
	require.NoError(t, polls.Create(ctx(), p))

	first, err := polls.Publish(ctx(), p.ID)
	require.NoError(t, err)
	require.NotNil(t, first.ResultsPublishedAt, "e'lon vaqti to'ldirilishi kerak")

	time.Sleep(10 * time.Millisecond)

	second, err := polls.Publish(ctx(), p.ID)
	require.NoError(t, err)
	require.NotNil(t, second.ResultsPublishedAt)
	require.True(t, first.ResultsPublishedAt.Equal(*second.ResultsPublishedAt),
		"qayta e'lon vaqtni surmasligi kerak (COALESCE idempotentligi)")
}

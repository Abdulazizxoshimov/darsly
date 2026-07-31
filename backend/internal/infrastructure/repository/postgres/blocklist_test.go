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

// №4 — mentor darajasidagi doimiy qora ro'yxat: ism mosligi katta-kichik
// harf farqsiz, dublikat jimgina no-op, unban faqat egasiga.
func TestBlocklistRepo_CaseInsensitiveMatchAndDedup(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	repo := pgRepo.NewBlocklistRepo(pg)
	mentor := makeUser(t, users, "blk@darsly.uz")

	e := &entity.BlocklistEntry{ID: uuid.NewString(), MentorID: mentor.ID, Identity: "guest_1", DisplayName: "Bezori Aka"}
	require.NoError(t, repo.Add(ctx(), e))

	// Dublikat (boshqa registrda) — xato emas, no-op (ON CONFLICT DO NOTHING).
	dup := &entity.BlocklistEntry{ID: uuid.NewString(), MentorID: mentor.ID, Identity: "guest_2", DisplayName: "BEZORI aka"}
	require.NoError(t, repo.Add(ctx(), dup))

	items, err := repo.ListByMentor(ctx(), mentor.ID)
	require.NoError(t, err)
	require.Len(t, items, 1, "bir ism bir marta")
	require.Equal(t, "Bezori Aka", items[0].DisplayName)

	// Moslik katta-kichik harf va chekka bo'shliqlarga befarq.
	for _, name := range []string{"Bezori Aka", "bezori aka", "  BEZORI AKA  "} {
		blocked, err := repo.IsBlocked(ctx(), mentor.ID, name)
		require.NoError(t, err)
		require.True(t, blocked, "%q mos kelishi kerak", name)
	}
	blocked, err := repo.IsBlocked(ctx(), mentor.ID, "Halol O'quvchi")
	require.NoError(t, err)
	require.False(t, blocked)
	// Bo'sh ism hech qachon mos kelmaydi.
	blocked, err = repo.IsBlocked(ctx(), mentor.ID, "   ")
	require.NoError(t, err)
	require.False(t, blocked)

	// Boshqa mentorning ro'yxatiga ta'sir qilmaydi (tenant izolyatsiyasi).
	other := makeUser(t, users, "blk2@darsly.uz")
	blocked, err = repo.IsBlocked(ctx(), other.ID, "Bezori Aka")
	require.NoError(t, err)
	require.False(t, blocked, "ban faqat o'z mentoriga tegishli")
}

func TestBlocklistRepo_DeleteIsOwnerScoped(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	repo := pgRepo.NewBlocklistRepo(pg)
	mentor := makeUser(t, users, "blk3@darsly.uz")
	intruder := makeUser(t, users, "blk4@darsly.uz")

	e := &entity.BlocklistEntry{ID: uuid.NewString(), MentorID: mentor.ID, Identity: "g", DisplayName: "X"}
	require.NoError(t, repo.Add(ctx(), e))

	// Begona mentor o'chira olmaydi (IDOR emas).
	err := repo.Delete(ctx(), intruder.ID, e.ID)
	require.True(t, apperr.IsNotFound(err))

	// Egasi o'chiradi — ro'yxat bo'shaydi.
	require.NoError(t, repo.Delete(ctx(), mentor.ID, e.ID))
	blocked, err := repo.IsBlocked(ctx(), mentor.ID, "X")
	require.NoError(t, err)
	require.False(t, blocked)
}

// №11 — ovoz nazorati bayroqlari DB'da to'liq aylanib chiqadi (persist + scan).
func TestLessonRepo_AudioPolicyFlagsRoundTrip(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	lessons := pgRepo.NewLessonRepo(pg)
	mentor := makeUser(t, users, "audio@darsly.uz")

	now := time.Now().UTC()
	l := &entity.Lesson{
		ID: uuid.NewString(), MentorID: mentor.ID, Title: "Ovoz", DurationMin: 60,
		JoinSlug: "aud-test-" + uuid.NewString()[:8], Status: entity.LessonStatusScheduled,
		MuteOnEntry: true, AllowSelfUnmute: true,
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, lessons.Create(ctx(), l))

	got, err := lessons.GetByID(ctx(), l.ID)
	require.NoError(t, err)
	require.True(t, got.MuteOnEntry)
	require.True(t, got.AllowSelfUnmute)

	// Jonli o'zgartirish (mute-all checkbox / PATCH) saqlanadi.
	got.AllowSelfUnmute = false
	require.NoError(t, lessons.Update(ctx(), got))
	got2, err := lessons.GetByID(ctx(), l.ID)
	require.NoError(t, err)
	require.False(t, got2.AllowSelfUnmute)
	require.True(t, got2.MuteOnEntry, "boshqa bayroq o'zgarmasligi kerak")
}

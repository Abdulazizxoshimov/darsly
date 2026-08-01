package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
	"github.com/zoom/darsly/internal/testutil"
)

// Bu testlar HAQIQIY Postgres'ga tegadi. Sabab: Telegram arxivining butun
// mantig'i murakkab SQL'da yashaydi (`FOR UPDATE SKIP LOCKED` navbat, shartli
// `WHERE` bilan atomik claim'lar, partial indekslar). Ularni faqat fake bilan
// sinash "SQL to'g'ri yozilganmi" degan asosiy savolga javob bermasdi —
// Squirrel so'rovlari kompilyatsiyada emas, ishlash paytida tekshiriladi.

// seedReadyRecording — arxivlanishga tayyor yozuv.
//
// `ended_at` ATAYLAB to'ldirilmaydi: repozitoriy retention hisobida
// `COALESCE(ended_at, created_at)` ishlatadi va shu tarmoq ham qamrab olinsin
// (aynan u yozuvni retention'dan butunlay tushirib qoldirishi mumkin edi).
func seedReadyRecording(t *testing.T, recs repository.RecordingRepository, lessonID string, endedAgo time.Duration) *entity.Recording {
	t.Helper()
	ended := time.Now().UTC().Add(-endedAgo)
	rec := &entity.Recording{
		ID: uuid.NewString(), LessonID: lessonID, EgressID: "eg-" + uuid.NewString()[:8],
		ObjectKey: "recordings/" + uuid.NewString() + ".mp4",
		Status:    entity.RecordingStatusReady,
		StartedAt: ended.Add(-time.Hour), CreatedAt: ended,
	}
	require.NoError(t, recs.Create(ctx(), rec))
	return rec
}

// tgFixture — mentor + dars, Telegram testlari uchun umumiy tayyorgarlik.
func tgFixture(t *testing.T) (*pg.Postgres, *entity.User, string) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(db)
	lessons := pgRepo.NewLessonRepo(db)
	mentor := makeUser(t, users, "tg-"+uuid.NewString()[:8]+"@darsly.uz")
	l := &entity.Lesson{
		ID: uuid.NewString(), MentorID: mentor.ID, Title: "Algebra", DurationMin: 60,
		JoinSlug: "tg-" + uuid.NewString()[:8], Status: entity.LessonStatusEnded,
		CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	require.NoError(t, lessons.Create(ctx(), l))
	return db, mentor, l.ID
}

// Bog'lash: qo'yish, teskari qidiruv va boshqa hisobga o'tkazish.
func TestTelegramRepo_LinkAndReassign(t *testing.T) {
	db, mentor, _ := tgFixture(t)
	users := pgRepo.NewUserRepo(db)
	repo := pgRepo.NewTelegramRepo(db)
	other := makeUser(t, users, "tg2-"+uuid.NewString()[:8]+"@darsly.uz")

	require.NoError(t, repo.LinkUser(ctx(), mentor.ID, 555, "ali_tg"))

	got, err := repo.GetUserByTelegramID(ctx(), 555)
	require.NoError(t, err)
	require.Equal(t, mentor.ID, got.ID)
	require.NotNil(t, got.TelegramUsername)
	require.Equal(t, "ali_tg", *got.TelegramUsername)

	// ⭐ Ayni Telegram akkaunti boshqa hisobga bog'lansa BIRINCHISIDAN uziladi.
	// Busiz UNIQUE indeks bilan to'qnashib, bog'lash tushunarsiz 500 berardi.
	require.NoError(t, repo.LinkUser(ctx(), other.ID, 555, "ali_tg"))

	moved, err := repo.GetUserByTelegramID(ctx(), 555)
	require.NoError(t, err)
	require.Equal(t, other.ID, moved.ID)

	old, err := repo.GetLink(ctx(), mentor.ID)
	require.NoError(t, err)
	require.Nil(t, old.TelegramUserID, "eski egadan uzilishi kerak")
}

func TestTelegramRepo_UnlinkAndMissing(t *testing.T) {
	db, mentor, _ := tgFixture(t)
	repo := pgRepo.NewTelegramRepo(db)

	require.NoError(t, repo.LinkUser(ctx(), mentor.ID, 777, ""))
	require.NoError(t, repo.UnlinkUser(ctx(), mentor.ID))

	_, err := repo.GetUserByTelegramID(ctx(), 777)
	require.True(t, apperr.IsNotFound(err))

	// Idempotent: bog'lanmagan holatda ham xato bermaydi.
	require.NoError(t, repo.UnlinkUser(ctx(), mentor.ID))
}

// UpsertChat egalikni SAQLAYDI: nom o'zgargandagi hodisa boshqa (bog'lanmagan)
// odamdan kelsa ham guruh mentordan tortib olinmaydi.
func TestTelegramRepo_UpsertChatKeepsOwner(t *testing.T) {
	db, mentor, _ := tgFixture(t)
	repo := pgRepo.NewTelegramRepo(db)

	require.NoError(t, repo.UpsertChat(ctx(), &entity.TelegramChat{
		ChatID: -1001, Title: "9-A", Type: "supergroup", MentorID: &mentor.ID,
	}))
	// Ikkinchi hodisa egasiz (mentor_id = NULL).
	require.NoError(t, repo.UpsertChat(ctx(), &entity.TelegramChat{
		ChatID: -1001, Title: "9-A sinf", Type: "supergroup",
	}))

	chats, err := repo.ListChatsByMentor(ctx(), mentor.ID)
	require.NoError(t, err)
	require.Len(t, chats, 1, "egalik saqlanishi kerak")
	require.Equal(t, "9-A sinf", chats[0].Title, "nom yangilanishi kerak")

	// Deaktivatsiya ro'yxatdan chiqaradi, lekin qatorni o'chirmaydi.
	require.NoError(t, repo.DeactivateChat(ctx(), -1001))
	chats, err = repo.ListChatsByMentor(ctx(), mentor.ID)
	require.NoError(t, err)
	require.Empty(t, chats)

	still, err := repo.GetChat(ctx(), -1001)
	require.NoError(t, err, "qator tarix uchun qolishi kerak")
	require.False(t, still.IsActive)
}

// Telegram yuklash navbati: qo'yish → atomik olish → tasdiqlash.
func TestRecordingRepo_TelegramQueue(t *testing.T) {
	db, _, lessonID := tgFixture(t)
	recs := pgRepo.NewRecordingRepo(db)

	rec := seedReadyRecording(t, recs, lessonID, time.Hour)
	require.NoError(t, recs.EnqueueTelegram(ctx(), rec.EgressID))

	claimed, err := recs.ClaimTelegramUpload(ctx(), time.Now().UTC(), 3)
	require.NoError(t, err)
	require.NotNil(t, claimed)
	require.Equal(t, rec.ID, claimed.ID)

	// ⭐ Ikkinchi claim BO'SH: urinish vaqti 30 daqiqaga surilgan, ya'ni ikki
	// instans bir faylni parallel yubormaydi (guruhda dublikat video bo'lmasin).
	again, err := recs.ClaimTelegramUpload(ctx(), time.Now().UTC(), 3)
	require.NoError(t, err)
	require.Nil(t, again)

	require.NoError(t, recs.MarkTelegramSent(ctx(), rec.ID, -1001, 42, "file-abc"))
	got, err := recs.GetByID(ctx(), rec.ID)
	require.NoError(t, err)
	require.NotNil(t, got.TelegramSentAt)
	require.NotNil(t, got.TelegramFileID)
	require.Equal(t, "file-abc", *got.TelegramFileID)

	// Tasdiqlangandan keyin navbatga qaytmaydi.
	after, err := recs.ClaimTelegramUpload(ctx(), time.Now().UTC().Add(time.Hour), 3)
	require.NoError(t, err)
	require.Nil(t, after)
}

// `maxAttempts` ga yetgan yozuv navbatdan chiqadi (mentor aralashuvini kutadi).
func TestRecordingRepo_TelegramQueue_RespectsMaxAttempts(t *testing.T) {
	db, _, lessonID := tgFixture(t)
	recs := pgRepo.NewRecordingRepo(db)
	rec := seedReadyRecording(t, recs, lessonID, time.Hour)
	require.NoError(t, recs.EnqueueTelegram(ctx(), rec.EgressID))

	now := time.Now().UTC()
	for i := 0; i < 3; i++ {
		claimed, err := recs.ClaimTelegramUpload(ctx(), now, 3)
		require.NoError(t, err, "urinish %d", i+1)
		require.NotNil(t, claimed, "urinish %d bo'lishi kerak", i+1)
		next := now
		require.NoError(t, recs.MarkTelegramFailed(ctx(), rec.ID, "xato", &next))
	}
	fourth, err := recs.ClaimTelegramUpload(ctx(), now, 3)
	require.NoError(t, err)
	require.Nil(t, fourth, "3 urinishdan keyin navbatdan chiqishi kerak")
}

// ⭐ ASOSIY KAFOLAT: tasdiqlanmagan yozuv arxivlanadigan ro'yxatga TUSHMAYDI
// va `ClaimArchive` uni o'chirishga ruxsat BERMAYDI.
func TestRecordingRepo_ArchiveRequiresTelegramConfirmation(t *testing.T) {
	db, _, lessonID := tgFixture(t)
	recs := pgRepo.NewRecordingRepo(db)
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour)

	unconfirmed := seedReadyRecording(t, recs, lessonID, 40*24*time.Hour)
	confirmed := seedReadyRecording(t, recs, lessonID, 40*24*time.Hour)
	require.NoError(t, recs.MarkTelegramSent(ctx(), confirmed.ID, -1001, 1, "file-x"))

	archivable, err := recs.ListArchivable(ctx(), cutoff, 100)
	require.NoError(t, err)
	require.Len(t, archivable, 1)
	require.Equal(t, confirmed.ID, archivable[0].ID)

	pending, err := recs.ListExpiredUnconfirmed(ctx(), cutoff, 100)
	require.NoError(t, err)
	require.Len(t, pending, 1)
	require.Equal(t, unconfirmed.ID, pending[0].ID)

	// Tasdiqlanmaganni to'g'ridan-to'g'ri arxivlashga urinish ham RAD ETILADI.
	ok, err := recs.ClaimArchive(ctx(), unconfirmed.ID, time.Now().UTC())
	require.NoError(t, err)
	require.False(t, ok, "tasdiqlanmagan yozuv o'chirilmasligi kerak")

	ok, err = recs.ClaimArchive(ctx(), confirmed.ID, time.Now().UTC())
	require.NoError(t, err)
	require.True(t, ok)

	got, err := recs.GetByID(ctx(), confirmed.ID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusArchived, got.Status)
}

// Tiklash oqimi: claim (idempotent) → finish → kesh muddati → evict.
func TestRecordingRepo_RestoreAndCacheLifecycle(t *testing.T) {
	db, _, lessonID := tgFixture(t)
	recs := pgRepo.NewRecordingRepo(db)

	rec := seedReadyRecording(t, recs, lessonID, 40*24*time.Hour)
	require.NoError(t, recs.MarkTelegramSent(ctx(), rec.ID, -1001, 1, "file-y"))
	ok, err := recs.ClaimArchive(ctx(), rec.ID, time.Now().UTC())
	require.NoError(t, err)
	require.True(t, ok)

	// Birinchi claim o'tadi, ikkinchisi YO'Q (ikki parallel yuklab olish bo'lmasin).
	ok, err = recs.ClaimRestore(ctx(), rec.ID)
	require.NoError(t, err)
	require.True(t, ok)
	ok, err = recs.ClaimRestore(ctx(), rec.ID)
	require.NoError(t, err)
	require.False(t, ok, "tiklash idempotent bo'lishi kerak")

	restoring, err := recs.ListByStatus(ctx(), entity.RecordingStatusRestoring, 10)
	require.NoError(t, err)
	require.Len(t, restoring, 1, "ishchi `restoring` yozuvni topa olishi kerak")

	// Kesh muddati O'TGAN qilib yozamiz — evict yo'lini sinash uchun.
	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, recs.FinishRestore(ctx(), rec.ID, rec.ObjectKey, past))

	got, err := recs.GetByID(ctx(), rec.ID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, got.Status)
	require.NotNil(t, got.CachedUntil)

	expired, err := recs.ListCacheExpired(ctx(), time.Now().UTC(), 10)
	require.NoError(t, err)
	require.Len(t, expired, 1)

	ok, err = recs.ClaimCacheEvict(ctx(), rec.ID)
	require.NoError(t, err)
	require.True(t, ok)

	got, err = recs.GetByID(ctx(), rec.ID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusArchived, got.Status)
	require.Nil(t, got.CachedUntil)
}

// Tiklash yiqilsa `archived` ga qaytadi — `restoring` da qotib qolmaydi.
func TestRecordingRepo_FailRestore(t *testing.T) {
	db, _, lessonID := tgFixture(t)
	recs := pgRepo.NewRecordingRepo(db)

	rec := seedReadyRecording(t, recs, lessonID, 40*24*time.Hour)
	require.NoError(t, recs.MarkTelegramSent(ctx(), rec.ID, -1001, 1, "file-z"))
	_, err := recs.ClaimArchive(ctx(), rec.ID, time.Now().UTC())
	require.NoError(t, err)
	_, err = recs.ClaimRestore(ctx(), rec.ID)
	require.NoError(t, err)

	require.NoError(t, recs.FailRestore(ctx(), rec.ID, "tarmoq uzildi"))

	got, err := recs.GetByID(ctx(), rec.ID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusArchived, got.Status)
	require.NotNil(t, got.TelegramError)
}

// Kesh nusxasi retention ro'yxatiga TUSHMAYDI: uning muddati `cached_until`
// bilan boshqariladi, aks holda u ikki xil yo'ldan o'chirilardi.
func TestRecordingRepo_CachedCopyExcludedFromRetention(t *testing.T) {
	db, _, lessonID := tgFixture(t)
	recs := pgRepo.NewRecordingRepo(db)

	rec := seedReadyRecording(t, recs, lessonID, 40*24*time.Hour)
	require.NoError(t, recs.MarkTelegramSent(ctx(), rec.ID, -1001, 1, "file-c"))
	_, err := recs.ClaimArchive(ctx(), rec.ID, time.Now().UTC())
	require.NoError(t, err)
	_, err = recs.ClaimRestore(ctx(), rec.ID)
	require.NoError(t, err)
	future := time.Now().UTC().Add(24 * time.Hour)
	require.NoError(t, recs.FinishRestore(ctx(), rec.ID, rec.ObjectKey, future))

	archivable, err := recs.ListArchivable(ctx(), time.Now().UTC().Add(-30*24*time.Hour), 100)
	require.NoError(t, err)
	require.Empty(t, archivable, "kesh nusxasi retention ro'yxatiga tushmasligi kerak")
}

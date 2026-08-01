package worker_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	tg "github.com/zoom/darsly/internal/infrastructure/telegram"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/notification"
	"github.com/zoom/darsly/internal/worker"
)

const uploadRecID = "44444444-4444-4444-8444-444444444444"

type uploadEnv struct {
	w       *worker.TelegramUploadWorker
	recRepo *testutil.FakeRecordingRepo
	bot     *testutil.FakeTelegram
	notif   *testutil.FakeNotifRepo
	minio   *testutil.FakeMinio
}

// newUploadEnv — bitta tayyor va navbatga qo'yilgan yozuv.
//
// Backoff ATAYLAB nol oraliqlar bilan: testda haqiqiy kutish yo'q, lekin
// urinishlar SONI (3) saqlanadi — aynan shu tekshiriladi.
func newUploadEnv(t *testing.T) *uploadEnv {
	t.Helper()
	ctx := context.Background()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: retLessonID, MentorID: "mentor1", Title: "Algebra", Status: entity.LessonStatusEnded,
	}))
	recRepo := testutil.NewFakeRecordingRepo()
	mc := testutil.NewFakeMinio()
	notifRepo := testutil.NewFakeNotifRepo()
	bot := testutil.NewFakeTelegram()

	ended := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, recRepo.Create(ctx, &entity.Recording{
		ID: uploadRecID, LessonID: retLessonID, EgressID: "eg-up",
		ObjectKey: "recordings/up.mp4", Status: entity.RecordingStatusReady,
		SizeBytes: 2048, DurationSec: 3600, EndedAt: &ended, CreatedAt: ended,
	}))
	mc.Objects["recordings/up.mp4"] = true
	require.NoError(t, recRepo.EnqueueTelegram(ctx, "eg-up"))

	cfg := worker.DefaultTelegramUploadConfig()
	cfg.Backoff = []time.Duration{0, 0, 0} // 3 urinish, kutishsiz

	return &uploadEnv{
		w: worker.NewTelegramUploadWorker(recRepo, lrepo, mc, bot,
			notification.New(notifRepo, ws.NewHub(testutil.NewLogger()), testutil.NewLogger()),
			nil, cfg, testutil.NewLogger()),
		recRepo: recRepo, bot: bot, notif: notifRepo, minio: mc,
	}
}

func (e *uploadEnv) rec(t *testing.T) *entity.Recording {
	t.Helper()
	rec, err := e.recRepo.GetByID(context.Background(), uploadRecID)
	require.NoError(t, err)
	return rec
}

// Muvaffaqiyatli yuklash `telegram_sent_at` va `file_id` ni yozadi —
// AYNAN shu ikkisi retention'ga o'chirishga ruxsat beradi va tiklashni
// mumkin qiladi.
func TestTelegramUpload_Success(t *testing.T) {
	e := newUploadEnv(t)
	e.w.RunOnce(context.Background())

	rec := e.rec(t)
	require.NotNil(t, rec.TelegramSentAt, "muvaffaqiyat tasdiqlanishi kerak")
	require.NotNil(t, rec.TelegramFileID, "file_id siz yozuvni tiklab bo'lmaydi")
	require.Equal(t, 1, e.bot.Sent[e.bot.ArchiveID], "arxiv guruhiga bir marta yuborilishi kerak")
}

// ⭐ Yozuv IKKI MARTA yuborilmaydi: birinchi muvaffaqiyatdan keyin navbatdan
// chiqadi. Aks holda arxiv guruhida dublikat videolar to'planardi.
func TestTelegramUpload_NoDuplicateSend(t *testing.T) {
	e := newUploadEnv(t)
	ctx := context.Background()

	e.w.RunOnce(ctx)
	e.w.RunOnce(ctx)
	e.w.RunOnce(ctx)

	require.Equal(t, 1, e.bot.Sent[e.bot.ArchiveID], "faqat bitta yuborish bo'lishi kerak")
}

// Telegram javobida `file_id` bo'lmasa yuborish MUVAFFAQIYATSIZ deb
// qaraladi: `file_id` siz yozuvni qaytarib bo'lmaydi, ya'ni uni "arxivlandi"
// deb belgilash server nusxasini o'chirishga ruxsat berib, videoni butunlay
// yo'qotardi.
func TestTelegramUpload_MissingFileIDIsFailure(t *testing.T) {
	e := newUploadEnv(t)
	e.bot.EmptyFileID = true

	e.w.RunOnce(context.Background())

	rec := e.rec(t)
	require.Nil(t, rec.TelegramSentAt, "file_id kelmasa tasdiqlanmasligi kerak")
	require.NotNil(t, rec.TelegramError)
}

// 3 urinishdan keyin taslim bo'ladi, mentorga bildirishnoma ketadi va
// SERVER NUSXASI SAQLANADI (status hamon `ready`, fayl MinIO'da).
func TestTelegramUpload_GivesUpAfterThreeAttempts(t *testing.T) {
	ctx := context.Background()
	e := newUploadEnv(t)
	e.bot.SendVideoErr = fmt.Errorf("tarmoq uzildi")

	// Har `RunOnce` bitta urinish sarflaydi (backoff nol → darhol qayta olinadi).
	e.w.RunOnce(ctx)
	e.w.RunOnce(ctx)
	e.w.RunOnce(ctx)
	// To'rtinchi tsiklda navbatda hech narsa qolmaydi.
	e.w.RunOnce(ctx)

	rec := e.rec(t)
	require.Equal(t, 3, rec.TelegramAttempts, "aniq 3 urinish bo'lishi kerak")
	require.Nil(t, rec.TelegramSentAt)
	require.Equal(t, entity.RecordingStatusReady, rec.Status,
		"yuklash yiqilsa ham yozuv 'ready' bo'lib qolishi kerak")
	require.True(t, e.minio.Objects["recordings/up.mp4"],
		"⭐ server nusxasi O'CHIRILMASLIGI kerak")

	items, total, err := e.notif.ListByUser(ctx, "mentor1", &entity.NotificationFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total, "mentorga bir marta xabar berilishi kerak")
	require.Equal(t, entity.NotificationTypeTelegramFailed, items[0].Type)
	require.Contains(t, items[0].Body, "SAQLANIB QOLDI")
}

// Doimiy xato (bot guruhda emas — 403) urinishlarni SARFLAMAY darhol
// taslim bo'ladi: 15 daqiqadan keyin ham natija bir xil bo'lardi.
func TestTelegramUpload_PermanentErrorFailsFast(t *testing.T) {
	ctx := context.Background()
	e := newUploadEnv(t)
	e.bot.SendVideoErr = &tg.APIError{Code: 403, Method: "sendVideo", Description: "bot was kicked"}

	e.w.RunOnce(ctx)

	rec := e.rec(t)
	require.Equal(t, 1, rec.TelegramAttempts, "doimiy xato qayta urinilmasligi kerak")

	_, total, err := e.notif.ListByUser(ctx, "mentor1", &entity.NotificationFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total, "darhol xabar berilishi kerak")
}

// 429 (`retry_after`) urinishni SARFLAMAYDI va keyingi urinishni Telegram
// aytgan vaqtga suradi — bu bizning aybimiz emas, vaqtinchalik holat.
func TestTelegramUpload_RateLimitDoesNotExhaustAttempts(t *testing.T) {
	ctx := context.Background()
	e := newUploadEnv(t)
	e.bot.SendVideoErr = &tg.RetryAfterError{After: time.Hour}

	e.w.RunOnce(ctx)

	rec := e.rec(t)
	require.Nil(t, rec.TelegramSentAt)
	// Keyingi urinish bir soatga surilgani uchun navbatda darhol chiqmaydi.
	claimed, err := e.recRepo.ClaimTelegramUpload(ctx, time.Now().UTC(), 3)
	require.NoError(t, err)
	require.Nil(t, claimed, "rate-limit'dan keyin darhol qayta urinilmasligi kerak")

	// Ogohlantirish YUBORILMAYDI: bu tugallangan nosozlik emas.
	_, total, err := e.notif.ListByUser(ctx, "mentor1", &entity.NotificationFilter{})
	require.NoError(t, err)
	require.Zero(t, total)
}

// Chegaradan katta fayl yuklab OLINMAYDI ham: tekshiruv MinIO'dan tortishdan
// oldin bo'ladi (aks holda gigabaytlar bekorga sarflanardi).
func TestTelegramUpload_TooLargeIsRejectedEarly(t *testing.T) {
	ctx := context.Background()
	e := newUploadEnv(t)
	e.bot.MaxUpload = 1024 // yozuv 2048 bayt

	e.w.RunOnce(ctx)

	rec := e.rec(t)
	require.Nil(t, rec.TelegramSentAt)
	require.Zero(t, e.bot.Sent[e.bot.ArchiveID], "Telegramga umuman murojaat qilinmasligi kerak")
	require.NotNil(t, rec.TelegramError)
	require.True(t, e.minio.Objects["recordings/up.mp4"], "fayl saqlanib qolishi kerak")
}

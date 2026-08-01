package worker_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/notification"
	"github.com/zoom/darsly/internal/worker"
)

// tgRetentionEnv — Telegram arxivi YOQILGAN retention ishchisi.
//
// `retentionEnv` dan farqi bitta bayroqda (`TelegramArchive: true`), lekin
// bu bayroq retention ma'nosini butunlay o'zgartiradi — shuning uchun alohida
// muhit va alohida testlar.
type tgRetentionEnv struct {
	w       *worker.RetentionWorker
	recRepo *testutil.FakeRecordingRepo
	minio   *testutil.FakeMinio
	notif   *testutil.FakeNotifRepo
}

func newTGRetentionEnv(t *testing.T) *tgRetentionEnv {
	t.Helper()
	ctx := context.Background()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: retLessonID, MentorID: "mentor1", Title: "Algebra", Status: entity.LessonStatusEnded,
	}))
	recRepo := testutil.NewFakeRecordingRepo()
	mc := testutil.NewFakeMinio()
	notifRepo := testutil.NewFakeNotifRepo()
	return &tgRetentionEnv{
		w: worker.NewRetentionWorker(recRepo, lrepo, mc,
			notification.New(notifRepo, ws.NewHub(testutil.NewLogger()), testutil.NewLogger()),
			worker.RetentionConfig{
				Retention:       30 * 24 * time.Hour,
				WarnBefore:      3 * 24 * time.Hour,
				Interval:        time.Hour,
				BatchLimit:      100,
				TelegramArchive: true,
			}, testutil.NewLogger()),
		recRepo: recRepo,
		minio:   mc,
		notif:   notifRepo,
	}
}

// add — yozuv qo'shadi. `sentToTelegram=false` bo'lsa Telegramda tasdiqlanmagan.
func (e *tgRetentionEnv) add(t *testing.T, id string, endedAgo time.Duration, sentToTelegram bool) {
	t.Helper()
	ended := time.Now().UTC().Add(-endedAgo)
	rec := &entity.Recording{
		ID: id, LessonID: retLessonID, EgressID: "eg-" + id,
		ObjectKey: "recordings/" + id + ".mp4",
		Status:    entity.RecordingStatusReady,
		StartedAt: ended.Add(-time.Hour), EndedAt: &ended, CreatedAt: ended,
	}
	if sentToTelegram {
		sent := ended.Add(time.Minute)
		fileID := "file-" + id
		rec.TelegramSentAt = &sent
		rec.TelegramFileID = &fileID
	}
	require.NoError(t, e.recRepo.Create(context.Background(), rec))
	e.minio.Objects["recordings/"+id+".mp4"] = true
}

func (e *tgRetentionEnv) status(t *testing.T, id string) string {
	t.Helper()
	rec, err := e.recRepo.GetByID(context.Background(), id)
	require.NoError(t, err)
	return rec.Status
}

// ⭐ ASOSIY KAFOLAT: Telegramda TASDIQLANMAGAN yozuv muddati o'tgan bo'lsa ham
// O'CHIRILMAYDI. Bu butun mahsulot qarorining («video hech qachon
// yo'qolmaydi») yagona texnik ifodasi.
func TestTGRetention_UnconfirmedIsNeverDeleted(t *testing.T) {
	ctx := context.Background()
	e := newTGRetentionEnv(t)
	e.add(t, "unconfirmed", 40*24*time.Hour, false)

	e.w.RunOnce(ctx)

	require.Equal(t, entity.RecordingStatusReady, e.status(t, "unconfirmed"),
		"Telegramda tasdiqlanmagan yozuv holati o'zgarmasligi kerak")
	require.True(t, e.minio.Objects["recordings/unconfirmed.mp4"],
		"fayl MinIO'da SAQLANIB QOLISHI kerak")
}

// Tasdiqlanmagan yozuv haqida mentor OGOHLANTIRILADI (jimgina disk to'lmasin)
// va bu ogohlantirish faqat BIR MARTA.
func TestTGRetention_WarnsAboutUnconfirmedOnce(t *testing.T) {
	ctx := context.Background()
	e := newTGRetentionEnv(t)
	e.add(t, "unconfirmed", 40*24*time.Hour, false)

	e.w.RunOnce(ctx)
	e.w.RunOnce(ctx)

	items, total, err := e.notif.ListByUser(ctx, "mentor1", &entity.NotificationFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total, "ogohlantirish takrorlanmasligi kerak")
	require.Equal(t, entity.NotificationTypeTelegramFailed, items[0].Type)
	require.Contains(t, items[0].Body, "SAQLANIB TURIBDI")
}

// Telegramda tasdiqlangan yozuv `archived` bo'ladi (`expired` EMAS) va fayl
// serverdan olib tashlanadi.
func TestTGRetention_ConfirmedBecomesArchived(t *testing.T) {
	ctx := context.Background()
	e := newTGRetentionEnv(t)
	e.add(t, "sent", 40*24*time.Hour, true)

	e.w.RunOnce(ctx)

	require.Equal(t, entity.RecordingStatusArchived, e.status(t, "sent"),
		"Telegramda bor yozuv 'expired' emas, 'archived' bo'lishi kerak")
	require.False(t, e.minio.Objects["recordings/sent.mp4"], "server nusxasi o'chishi kerak")

	// Mentor holat o'zgarganini bilishi kerak — aks holda «yozuvim yo'qoldi».
	items, total, err := e.notif.ListByUser(ctx, "mentor1", &entity.NotificationFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Equal(t, entity.NotificationTypeRecordingArchived, items[0].Type)
}

// Kesh muddati tugagan TIKLANGAN nusxa yana o'chiriladi va `archived` ga
// qaytadi. Bo'lmasa har ochilgan eski dars diskda abadiy qolardi.
func TestTGRetention_EvictsExpiredCache(t *testing.T) {
	ctx := context.Background()
	e := newTGRetentionEnv(t)

	sent := time.Now().UTC().Add(-40 * 24 * time.Hour)
	past := time.Now().UTC().Add(-time.Hour) // kesh muddati o'tgan
	fileID := "file-cached"
	require.NoError(t, e.recRepo.Create(ctx, &entity.Recording{
		ID: "cached", LessonID: retLessonID, EgressID: "eg-cached",
		ObjectKey: "recordings/cached.mp4",
		Status:    entity.RecordingStatusReady,
		EndedAt:   &sent, CreatedAt: sent,
		TelegramSentAt: &sent, TelegramFileID: &fileID,
		CachedUntil: &past,
	}))
	e.minio.Objects["recordings/cached.mp4"] = true

	e.w.RunOnce(ctx)

	require.Equal(t, entity.RecordingStatusArchived, e.status(t, "cached"))
	require.False(t, e.minio.Objects["recordings/cached.mp4"], "kesh nusxasi o'chishi kerak")
}

// Kesh muddati HALI TUGAMAGAN nusxa tegilmaydi (mentor uni ko'rib turgan
// bo'lishi mumkin) — retention uni "muddati o'tgan" deb ham hisoblamaydi.
func TestTGRetention_ActiveCacheUntouched(t *testing.T) {
	ctx := context.Background()
	e := newTGRetentionEnv(t)

	old := time.Now().UTC().Add(-40 * 24 * time.Hour)
	future := time.Now().UTC().Add(12 * time.Hour)
	fileID := "file-live-cache"
	require.NoError(t, e.recRepo.Create(ctx, &entity.Recording{
		ID: "livecache", LessonID: retLessonID, EgressID: "eg-livecache",
		ObjectKey: "recordings/livecache.mp4",
		Status:    entity.RecordingStatusReady,
		EndedAt:   &old, CreatedAt: old,
		TelegramSentAt: &old, TelegramFileID: &fileID,
		CachedUntil: &future,
	}))
	e.minio.Objects["recordings/livecache.mp4"] = true

	e.w.RunOnce(ctx)

	require.Equal(t, entity.RecordingStatusReady, e.status(t, "livecache"))
	require.True(t, e.minio.Objects["recordings/livecache.mp4"])
}

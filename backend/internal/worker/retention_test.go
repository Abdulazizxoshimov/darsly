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

const retLessonID = "22222222-2222-4222-8222-222222222222"

type retentionEnv struct {
	w        *worker.RetentionWorker
	recRepo  *testutil.FakeRecordingRepo
	minio    *testutil.FakeMinio
	notifRep *testutil.FakeNotifRepo
}

// newRetentionEnv — mentorga tegishli bitta dars + sozlangan retention ishchisi.
func newRetentionEnv(t *testing.T) *retentionEnv {
	t.Helper()
	ctx := context.Background()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(ctx, &entity.Lesson{
		ID: retLessonID, MentorID: "mentor1", Title: "Algebra", Status: entity.LessonStatusEnded,
	}))
	recRepo := testutil.NewFakeRecordingRepo()
	mc := testutil.NewFakeMinio()
	notifRepo := testutil.NewFakeNotifRepo()
	notif := notification.New(notifRepo, ws.NewHub(testutil.NewLogger()), testutil.NewLogger())
	cfg := worker.RetentionConfig{
		Retention:  30 * 24 * time.Hour,
		WarnBefore: 3 * 24 * time.Hour,
		Interval:   time.Hour,
		BatchLimit: 100,
	}
	return &retentionEnv{
		w:        worker.NewRetentionWorker(recRepo, lrepo, mc, notif, cfg, testutil.NewLogger()),
		recRepo:  recRepo,
		minio:    mc,
		notifRep: notifRepo,
	}
}

// addRecording — `ready` yozuv, `endedAgo` vaqt oldin tugagan.
func (e *retentionEnv) addRecording(t *testing.T, id string, endedAgo time.Duration) {
	t.Helper()
	ended := time.Now().UTC().Add(-endedAgo)
	require.NoError(t, e.recRepo.Create(context.Background(), &entity.Recording{
		ID: id, LessonID: retLessonID, EgressID: "eg-" + id,
		ObjectKey: "recordings/" + id + ".mp4",
		Status:    entity.RecordingStatusReady,
		StartedAt: ended.Add(-time.Hour), EndedAt: &ended, CreatedAt: ended,
	}))
	e.minio.Objects["recordings/"+id+".mp4"] = true
}

func (e *retentionEnv) status(t *testing.T, id string) string {
	t.Helper()
	rec, err := e.recRepo.GetByID(context.Background(), id)
	require.NoError(t, err)
	return rec.Status
}

// 30 kundan oshgan yozuv MinIO'dan o'chiriladi va `expired` deb belgilanadi;
// muddati o'tmagani TEGILMAYDI.
func TestRetention_DeletesOnlyExpired(t *testing.T) {
	e := newRetentionEnv(t)
	e.addRecording(t, "old", 31*24*time.Hour)
	e.addRecording(t, "fresh", 5*24*time.Hour)

	e.w.RunOnce(context.Background())

	require.Equal(t, entity.RecordingStatusExpired, e.status(t, "old"))
	require.False(t, e.minio.Objects["recordings/old.mp4"], "muddati o'tgan fayl MinIO'dan o'chishi kerak")

	require.Equal(t, entity.RecordingStatusReady, e.status(t, "fresh"))
	require.True(t, e.minio.Objects["recordings/fresh.mp4"], "muddati o'tmagan fayl qolishi kerak")
}

// Ogohlantirish o'chishga 3 kun qolganda YUBORILADI va faqat BIR MARTA
// (ishchi soatiga bir ishlaydi — dublikat bildirishnoma bo'lmasligi shart).
func TestRetention_WarnsOnceBeforeDeletion(t *testing.T) {
	ctx := context.Background()
	e := newRetentionEnv(t)
	// 28 kun oldin tugagan: o'chishiga 2 kun qoldi (3 kunlik oyna ichida).
	e.addRecording(t, "soon", 28*24*time.Hour)

	e.w.RunOnce(ctx)
	e.w.RunOnce(ctx)
	e.w.RunOnce(ctx)

	items, total, err := e.notifRep.ListByUser(ctx, "mentor1", &entity.NotificationFilter{})
	require.NoError(t, err)
	require.Equal(t, 1, total, "ogohlantirish faqat bir marta yuborilishi kerak")
	require.Equal(t, entity.NotificationTypeRecordingExpiring, items[0].Type)
	require.Contains(t, items[0].Body, "Algebra")
	// Ogohlantirilgan yozuv hali O'CHIRILMAYDI.
	require.Equal(t, entity.RecordingStatusReady, e.status(t, "soon"))
}

// Yangi yozuv uchun ogohlantirish ham, o'chirish ham bo'lmaydi.
func TestRetention_FreshRecordingUntouched(t *testing.T) {
	ctx := context.Background()
	e := newRetentionEnv(t)
	e.addRecording(t, "new", 24*time.Hour)

	e.w.RunOnce(ctx)

	_, total, err := e.notifRep.ListByUser(ctx, "mentor1", &entity.NotificationFilter{})
	require.NoError(t, err)
	require.Zero(t, total)
	require.Equal(t, entity.RecordingStatusReady, e.status(t, "new"))
}

// Retention o'chirilgan bo'lsa (RECORDING_RETENTION_DAYS=0) ishchi umuman
// ishlamaydi — eski o'rnatishlarda yozuvlar to'satdan yo'q bo'lib qolmasin.
func TestRetention_DisabledKeepsEverything(t *testing.T) {
	ctx := context.Background()
	lrepo := testutil.NewFakeLessonRepo()
	recRepo := testutil.NewFakeRecordingRepo()
	mc := testutil.NewFakeMinio()
	ended := time.Now().UTC().Add(-365 * 24 * time.Hour)
	require.NoError(t, recRepo.Create(ctx, &entity.Recording{
		ID: "ancient", LessonID: retLessonID, ObjectKey: "recordings/ancient.mp4",
		Status: entity.RecordingStatusReady, EndedAt: &ended, CreatedAt: ended,
	}))
	mc.Objects["recordings/ancient.mp4"] = true

	w := worker.NewRetentionWorker(recRepo, lrepo, mc,
		notification.New(testutil.NewFakeNotifRepo(), ws.NewHub(testutil.NewLogger()), testutil.NewLogger()),
		worker.RetentionConfig{Retention: 0}, testutil.NewLogger())

	// Run darhol qaytadi (Retention <= 0) — ctx'ni kutib qolmasligi ham tekshiriladi.
	done := make(chan struct{})
	go func() { defer close(done); w.Run(ctx) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("retention o'chirilganda Run darhol qaytishi kerak")
	}

	rec, err := recRepo.GetByID(ctx, "ancient")
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, rec.Status)
	require.True(t, mc.Objects["recordings/ancient.mp4"])
}

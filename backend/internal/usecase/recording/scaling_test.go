package recording_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/recording"
)

// setupWithCap — berilgan egress cap bilan usecase (system-design audit R1).
func setupWithCap(t *testing.T, egressMax int) (recording.UseCase, *testutil.FakeRecordingRepo, *testutil.FakeLessonRepo, *testutil.FakeLiveKit) {
	t.Helper()
	rrepo := testutil.NewFakeRecordingRepo()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive, IsRecordingEnabled: true}))
	lk := testutil.NewFakeLiveKit()
	uc := recording.New(rrepo, lrepo, lk, testutil.NewFakeMinio(), livekit.S3Config{}, testutil.NewFakeCache(),
		testRetention, testutil.NewFakeTelegram(), testCacheTTL, false, egressMax, testutil.NewLogger())
	return uc, rrepo, lrepo, lk
}

// TestEgressCap_BlocksAutoStart — cap to'lganda AVTOMATIK yozuv boshlanmaydi
// (dars o'zi to'xtamaydi), lekin xato ham qaytmaydi.
func TestEgressCap_BlocksAutoStart(t *testing.T) {
	uc, rrepo, _, lk := setupWithCap(t, 1)
	ctx := context.Background()

	// Boshqa darsda allaqachon bitta faol egress bor → cap (1) to'ldi.
	require.NoError(t, rrepo.Create(ctx, &entity.Recording{
		ID: "33333333-3333-4333-8333-333333333333", LessonID: "other-lesson",
		EgressID: "EG-other", Status: entity.RecordingStatusRecording,
	}))

	require.NoError(t, uc.EnsureRecording(ctx, testLessonID))
	require.Equal(t, 0, lk.Calls["StartRoomRecording"], "cap to'lganda avtomatik egress boshlanmasligi kerak")

	recs, _ := rrepo.ListByLesson(ctx, testLessonID)
	require.Empty(t, recs, "cap to'lganda bu dars uchun yozuv yaratilmaydi")
}

// TestEgressCap_ManualStartConflict — cap to'lganda qo'lda boshlash 409.
func TestEgressCap_ManualStartConflict(t *testing.T) {
	uc, rrepo, _, _ := setupWithCap(t, 1)
	ctx := context.Background()
	require.NoError(t, rrepo.Create(ctx, &entity.Recording{
		ID: "33333333-3333-4333-8333-333333333333", LessonID: "other-lesson",
		EgressID: "EG-other", Status: entity.RecordingStatusRecording,
	}))
	_, err := uc.StartRecording(ctx, "mentor1", testLessonID)
	require.True(t, apperr.IsConflict(err), "cap to'lganda qo'lda boshlash 409 bo'lishi kerak, oldi: %v", err)
}

// TestStartRecording_DoubleStartConflict — shu dars uchun faol yozuv bo'lsa
// qo'lda qayta boshlash 409 (audit topilma #4).
func TestStartRecording_DoubleStartConflict(t *testing.T) {
	uc, rrepo, _, lk := setupWithCap(t, 0)
	ctx := context.Background()

	_, err := uc.StartRecording(ctx, "mentor1", testLessonID)
	require.NoError(t, err)
	require.Equal(t, 1, lk.Calls["StartRoomRecording"])

	_, err = uc.StartRecording(ctx, "mentor1", testLessonID)
	require.True(t, apperr.IsConflict(err), "faol yozuv bor darsda qayta boshlash 409, oldi: %v", err)
	require.Equal(t, 1, lk.Calls["StartRoomRecording"], "ikkinchi egress ochilmasligi kerak")

	_ = rrepo
}

// TestHandleEgress_ReplayIdempotent — kech/takror `egress_ended` tayyor yozuv
// holatini buzmaydi va qayta navbatga qo'ymaydi (audit topilma #5).
func TestHandleEgress_ReplayIdempotent(t *testing.T) {
	uc, rrepo, _, _ := setupWithCap(t, 0)
	ctx := context.Background()
	seedRecording(t, rrepo, entity.RecordingStatusRecording)
	key := "recordings/" + testLessonID + "/" + testRecordingID

	// Birinchi COMPLETE → ready + transcode navbatiga.
	require.NoError(t, uc.HandleEgress(ctx, "EG1", true, key, 42, 1027))
	rec, _ := rrepo.GetByID(ctx, testRecordingID)
	require.Equal(t, entity.RecordingStatusReady, rec.Status)
	require.Equal(t, entity.TranscodePending, rrepo.TranscodeStatus(testRecordingID))

	// Yozuvni "arxivlangan" deb belgilaymiz va transcode holatini almashtiramiz —
	// keyin TAKROR COMPLETE kelsa bularni buzmasligi kerak.
	require.NoError(t, rrepo.UpdateStatus(ctx, testRecordingID, entity.RecordingStatusArchived))
	require.NoError(t, rrepo.FinishTranscode(ctx, testRecordingID, 18, 229, 40, 2))

	// Takror COMPLETE — status-guard tufayli 0 qator: hech narsa o'zgarmaydi.
	require.NoError(t, uc.HandleEgress(ctx, "EG1", true, key, 42, 1027))
	rec, _ = rrepo.GetByID(ctx, testRecordingID)
	require.Equal(t, entity.RecordingStatusArchived, rec.Status, "takror webhook arxivlangan yozuvni ready qilmasligi kerak")
	require.Equal(t, entity.TranscodeDone, rrepo.TranscodeStatus(testRecordingID), "takror webhook transcode'ni qayta navbatga qo'ymasligi kerak")
	require.Equal(t, 2, rec.ContentOffsetSec, "content_offset qayta jamlanmasligi kerak")
	_ = time.Now
}

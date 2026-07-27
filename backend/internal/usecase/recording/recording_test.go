package recording_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/recording"
)

// Haqiqiy UUID'lar: usecase ID formatini tekshiradi (shared.ValidateID).
const (
	testLessonID    = "11111111-1111-4111-8111-111111111111"
	testRecordingID = "22222222-2222-4222-8222-222222222222"
)

func setup(t *testing.T) (recording.UseCase, *testutil.FakeRecordingRepo, *testutil.FakeLessonRepo, *testutil.FakeLiveKit) {
	t.Helper()
	rrepo := testutil.NewFakeRecordingRepo()
	lrepo := testutil.NewFakeLessonRepo()
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive}))
	lk := testutil.NewFakeLiveKit() // enabled mock — egress chaqiruvlarini assert qilamiz
	uc := recording.New(rrepo, lrepo, lk, testutil.NewFakeMinio(), livekit.S3Config{}, testutil.NewLogger())
	return uc, rrepo, lrepo, lk
}

func seedRecording(t *testing.T, rrepo *testutil.FakeRecordingRepo, status string) *entity.Recording {
	t.Helper()
	rec := &entity.Recording{ID: testRecordingID, LessonID: testLessonID, EgressID: "EG1", ObjectKey: "recordings/" + testLessonID + "/" + testRecordingID, Status: status}
	require.NoError(t, rrepo.Create(context.Background(), rec))
	return rec
}

func TestStartRecording(t *testing.T) {
	uc, rrepo, _, lk := setup(t)
	rec, err := uc.StartRecording(context.Background(), "mentor1", testLessonID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusRecording, rec.Status)
	require.Equal(t, "egress-fake", rec.EgressID, "egress ID LiveKit'dan olinadi")
	require.GreaterOrEqual(t, lk.Calls["StartRoomRecording"], 1, "egress boshlanishi kerak")

	got, _ := rrepo.GetByID(context.Background(), rec.ID)
	require.NotNil(t, got)
}

func TestStartRecording_NotLive(t *testing.T) {
	uc, _, lrepo, _ := setup(t)
	l, _ := lrepo.GetByID(context.Background(), testLessonID)
	l.Status = entity.LessonStatusScheduled
	require.NoError(t, lrepo.Update(context.Background(), l))
	_, err := uc.StartRecording(context.Background(), "mentor1", testLessonID)
	require.Error(t, err, "jonli bo'lmagan darsni yozib bo'lmaydi")
}

func TestHandleEgress_Complete(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusProcessing)

	uc.HandleEgress(context.Background(), "EG1", true, "recordings/"+testLessonID+"/"+testRecordingID, 42, 1027)

	rec, err := rrepo.GetByID(context.Background(), testRecordingID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, rec.Status)
	require.Equal(t, 42, rec.DurationSec)
	require.Equal(t, int64(1027), rec.SizeBytes)
}

func TestHandleEgress_Failed(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusRecording)

	uc.HandleEgress(context.Background(), "EG1", false, "", 0, 0)

	rec, _ := rrepo.GetByID(context.Background(), testRecordingID)
	require.Equal(t, entity.RecordingStatusFailed, rec.Status)
}

func TestDownloadURL(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusReady)

	dl, err := uc.DownloadURL(context.Background(), "mentor1", testRecordingID)
	require.NoError(t, err)
	require.Contains(t, dl.URL, "recordings/"+testLessonID+"/"+testRecordingID)

	// Boshqa mentor → 403
	_, err = uc.DownloadURL(context.Background(), "intruder", testRecordingID)
	require.True(t, apperr.IsForbidden(err))
}

func TestDownloadURL_NotReady(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusProcessing)
	_, err := uc.DownloadURL(context.Background(), "mentor1", testRecordingID)
	require.Error(t, err, "tayyor bo'lmagan yozuvni yuklab bo'lmaydi")
}

// Yaroqsiz UUID DB'ga yetmasin: avval `22P02` → 500 INTERNAL_ERROR bo'lardi
// (har bir authed foydalanuvchi Sentry'ni to'ldirib real nosozlikni yashira olardi).
func TestRecording_InvalidUUID_NotFound(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusReady)
	ctx := context.Background()

	for _, id := range []string{"abc", "", "rec-1"} {
		// lessonID yaroqsiz (shared.OwnedLesson tekshiradi)
		_, err := uc.StartRecording(ctx, "mentor1", id)
		require.True(t, apperr.IsNotFound(err), "StartRecording(lesson=%q) → 404 kutilgan, oldi: %v", id, err)
		_, err = uc.ListByLesson(ctx, "mentor1", id)
		require.True(t, apperr.IsNotFound(err), "ListByLesson(%q) → 404 kutilgan, oldi: %v", id, err)

		// recordingID yaroqsiz
		require.True(t, apperr.IsNotFound(uc.StopRecording(ctx, "mentor1", id)), "StopRecording(%q) → 404 kutilgan", id)
		_, err = uc.DownloadURL(ctx, "mentor1", id)
		require.True(t, apperr.IsNotFound(err), "DownloadURL(%q) → 404 kutilgan, oldi: %v", id, err)
	}
}

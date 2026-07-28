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
	"github.com/zoom/darsly/internal/usecase/shared"
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
	// Default: yozib olish YONIQ (dars yaratishda shunday keladi).
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive, IsRecordingEnabled: true}))
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

// ─── Majburiy (avtomatik) yozib olish ────────────────────────────────────────

func TestEnsureRecording_StartsWhenLive(t *testing.T) {
	uc, rrepo, _, lk := setup(t)
	require.NoError(t, uc.EnsureRecording(context.Background(), testLessonID))
	require.Equal(t, 1, lk.Calls["StartRoomRecording"], "jonli darsda yozuv avtomatik boshlanadi")

	recs, _ := rrepo.ListByLesson(context.Background(), testLessonID)
	require.Len(t, recs, 1)
	require.Equal(t, entity.RecordingStatusRecording, recs[0].Status)
}

func TestEnsureRecording_Idempotent(t *testing.T) {
	// ⭐ ENG MUHIM MEZON. Ustoz dars davomida qayta ulanishi odatiy hol:
	// internet uzildi, ilova fon rejimidan qaytdi, telefon almashtirildi.
	// Har `HostToken` da yangi egress boshlansa, BITTA dars uchun bir nechta
	// parallel yozuv ketardi — har biri ~1-2 yadro yeydi va 4 yadroli server
	// yiqilardi (ya'ni "kechikish minimal" talabi ham buzilardi).
	uc, rrepo, _, lk := setup(t)

	require.NoError(t, uc.EnsureRecording(context.Background(), testLessonID))
	require.NoError(t, uc.EnsureRecording(context.Background(), testLessonID))
	require.NoError(t, uc.EnsureRecording(context.Background(), testLessonID))

	require.Equal(t, 1, lk.Calls["StartRoomRecording"], "uch marta chaqirilsa ham egress BITTA bo'lishi kerak")
	recs, _ := rrepo.ListByLesson(context.Background(), testLessonID)
	require.Len(t, recs, 1, "dublikat yozuv yaratilmasligi kerak")
}

func TestEnsureRecording_SkipsWhenNotLive(t *testing.T) {
	// Poyga: dars biz kelgunimizcha yakunlangan bo'lishi mumkin. Bunda yozuv
	// boshlanmasligi kerak, lekin XATO ham qaytmasligi kerak — bu normal holat.
	uc, _, lrepo, lk := setup(t)
	l, _ := lrepo.GetByID(context.Background(), testLessonID)
	l.Status = entity.LessonStatusEnded
	require.NoError(t, lrepo.Update(context.Background(), l))

	require.NoError(t, uc.EnsureRecording(context.Background(), testLessonID))
	require.Equal(t, 0, lk.Calls["StartRoomRecording"])
}

func TestEnsureRecording_RestartsAfterPreviousFinished(t *testing.T) {
	// Tugagan yozuv yangisini bloklamasligi kerak: ustoz darsni yakunlab,
	// keyin o'sha darsni qayta ochsa (status yana live bo'lsa) yozuv qayta
	// boshlanishi kerak. Idempotentlik faqat FAOL yozuvga tegishli.
	uc, rrepo, _, lk := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusReady) // eski, tugagan yozuv

	require.NoError(t, uc.EnsureRecording(context.Background(), testLessonID))
	require.Equal(t, 1, lk.Calls["StartRoomRecording"], "tugagan yozuv yangisini to'smasligi kerak")
}

func TestStopActiveForLesson(t *testing.T) {
	uc, rrepo, _, lk := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusRecording)

	require.NoError(t, uc.StopActiveForLesson(context.Background(), testLessonID))

	require.GreaterOrEqual(t, lk.Calls["StopRecording"], 1)
	got, _ := rrepo.GetByID(context.Background(), testRecordingID)
	require.Equal(t, entity.RecordingStatusProcessing, got.Status,
		"to'xtatilgach holat darhol processing bo'lsin — ustoz 'hali yozilmoqda' degan yolg'onni ko'rmasin")
}

func TestStopActiveForLesson_IgnoresFinished(t *testing.T) {
	uc, rrepo, _, lk := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusReady)

	require.NoError(t, uc.StopActiveForLesson(context.Background(), testLessonID))
	require.Equal(t, 0, lk.Calls["StopRecording"], "tayyor yozuvni qayta to'xtatishga urinilmasin")
}

func TestEnsureRecording_RespectsDisabledLesson(t *testing.T) {
	// ⭐ Yozib olish DEFAULT YONIQ, lekin MAJBURIY EMAS. Ustoz uni shu dars
	// uchun o'chirgan bo'lsa, avtomatik boshlash uni bekor qilmasligi kerak:
	// "o'chirdim, baribir yozildi" — bu maxfiylik buzilishi bo'lardi.
	uc, rrepo, lrepo, lk := setup(t)
	l, _ := lrepo.GetByID(context.Background(), testLessonID)
	l.IsRecordingEnabled = false
	require.NoError(t, lrepo.Update(context.Background(), l))

	require.NoError(t, uc.EnsureRecording(context.Background(), testLessonID))

	require.Equal(t, 0, lk.Calls["StartRoomRecording"], "o'chirilgan darsda yozuv boshlanmasin")
	recs, _ := rrepo.ListByLesson(context.Background(), testLessonID)
	require.Empty(t, recs)
}

// ─── Webhook orqali boshlash (5 daqiqalik tuzoq tuzatilishi) ─────────────────

func TestEnsureForRoom_StartsFromRoomName(t *testing.T) {
	// ⭐ Yozuv endi AYNAN shu yo'l bilan boshlanadi: LiveKit `track_published`
	// webhook'i faqat xona NOMINI beradi, dars ID sini emas.
	uc, rrepo, _, lk := setup(t)

	require.NoError(t, uc.EnsureForRoom(context.Background(), shared.RoomName(testLessonID)))

	require.Equal(t, 1, lk.Calls["StartRoomRecording"])
	recs, _ := rrepo.ListByLesson(context.Background(), testLessonID)
	require.Len(t, recs, 1)
}

func TestEnsureForRoom_IgnoresForeignRoom(t *testing.T) {
	// Begona xona nomi (boshqa tizim yoki LiveKit'ning o'z xonasi) — xato emas,
	// jimgina e'tiborsiz. Aks holda webhook 500 qaytarib, LiveKit hodisani
	// cheksiz qayta yuborardi.
	uc, _, _, lk := setup(t)

	require.NoError(t, uc.EnsureForRoom(context.Background(), "boshqa-xona"))
	require.NoError(t, uc.EnsureForRoom(context.Background(), ""))
	require.NoError(t, uc.EnsureForRoom(context.Background(), "lesson_"))

	require.Equal(t, 0, lk.Calls["StartRoomRecording"])
}

func TestEnsureForRoom_IdempotentAcrossManyTracks(t *testing.T) {
	// ⭐ Bir dars ichida ko'p trek e'lon qilinadi: mikrofon, kamera, ekran,
	// ekran audiosi — va HAR BIRI uchun webhook keladi. Ular yangi yozuv
	// boshlamasligi kerak, aks holda bitta darsdan to'rtta egress ketardi.
	uc, rrepo, _, lk := setup(t)
	room := shared.RoomName(testLessonID)

	for i := 0; i < 4; i++ {
		require.NoError(t, uc.EnsureForRoom(context.Background(), room))
	}

	require.Equal(t, 1, lk.Calls["StartRoomRecording"], "to'rt trek → BITTA yozuv")
	recs, _ := rrepo.ListByLesson(context.Background(), testLessonID)
	require.Len(t, recs, 1)
}

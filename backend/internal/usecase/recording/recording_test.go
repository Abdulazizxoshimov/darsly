package recording_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

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

// testRetention — yozuv saqlash muddati (`expires_at` hisoblanadigan maydon).
const testRetention = 30 * 24 * time.Hour

// testCacheTTL — Telegramdan tiklangan nusxa qancha turadi.
const testCacheTTL = 24 * time.Hour

func setup(t *testing.T) (recording.UseCase, *testutil.FakeRecordingRepo, *testutil.FakeLessonRepo, *testutil.FakeLiveKit) {
	uc, rrepo, lrepo, lk, _, _ := setupFull(t)
	return uc, rrepo, lrepo, lk
}

// setupFull — Telegram va MinIO faqe'lariga ham kirish beradi (arxiv/tiklash
// testlari uchun). `setup` — uning qisqa ko'rinishi, mavjud testlar
// o'zgarishsiz qolsin.
func setupFull(t *testing.T) (
	recording.UseCase, *testutil.FakeRecordingRepo, *testutil.FakeLessonRepo,
	*testutil.FakeLiveKit, *testutil.FakeTelegram, *testutil.FakeMinio,
) {
	t.Helper()
	rrepo := testutil.NewFakeRecordingRepo()
	lrepo := testutil.NewFakeLessonRepo()
	// Default: yozib olish YONIQ (dars yaratishda shunday keladi).
	require.NoError(t, lrepo.Create(context.Background(), &entity.Lesson{ID: testLessonID, MentorID: "mentor1", Status: entity.LessonStatusLive, IsRecordingEnabled: true}))
	lk := testutil.NewFakeLiveKit() // enabled mock — egress chaqiruvlarini assert qilamiz
	tgFake := testutil.NewFakeTelegram()
	mc := testutil.NewFakeMinio()
	uc := recording.New(rrepo, lrepo, lk, mc, livekit.S3Config{}, testutil.NewFakeCache(),
		testRetention, tgFake, testCacheTTL, false, 0, testutil.NewLogger())
	return uc, rrepo, lrepo, lk, tgFake, mc
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

// M1 — dublikat egress poygasi (TOCTOU).
//
// Auditda topilgan holat: `EnsureRecording` avval "faol yozuv bormi" deb
// tekshirar, keyin egress boshlar edi — ikkisi orasida qulf yo'q edi. Ustoz
// kamera va mikrofonni deyarli bir vaqtda yoqsa LiveKit IKKI `track_published`
// webhook'ini yuboradi, ikkalasi ham "yozuv yo'q" deb ko'radi va IKKI parallel
// egress boshlanadi: 2× CPU/disk va bitta darsdan ikkita fayl.
func TestEnsureRecording_NoDuplicateEgressUnderRace(t *testing.T) {
	uc, rrepo, _, lk := setup(t)
	ctx := context.Background()

	// Barcha gorutinalar bir vaqtda kirsin — poyga oynasi eng keng bo'lsin.
	const n = 8
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			<-start
			_ = uc.EnsureRecording(ctx, testLessonID)
		}()
	}
	close(start)
	wg.Wait()

	require.Equal(t, 1, lk.Calls["StartRoomRecording"],
		"bitta dars uchun FAQAT bitta egress boshlanishi kerak")

	recs, err := rrepo.ListByLesson(ctx, testLessonID)
	require.NoError(t, err)
	require.Len(t, recs, 1, "bitta darsdan ikkita yozuv qatori chiqmasin")
}

// Qulf yozuvni ABADIY bloklab qo'ymasligi kerak: egress boshlanmasa (LiveKit
// xatosi) keyingi `track_published` urinishi o'tishi shart, aks holda bitta
// vaqtinchalik xato butun darsni yozuvsiz qoldirardi.
func TestEnsureRecording_LockReleasedOnFailure(t *testing.T) {
	uc, rrepo, _, lk := setup(t)
	ctx := context.Background()

	lk.EgressErr = errors.New("livekit yetib bo'lmadi")
	require.Error(t, uc.EnsureRecording(ctx, testLessonID))

	// LiveKit tiklandi — keyingi webhook yozuvni boshlashi kerak. Qulf
	// bo'shatilmasa bu urinish TTL tugagunicha jimgina rad etilardi va dars
	// butunlay yozuvsiz qolardi.
	lk.EgressErr = nil
	require.NoError(t, uc.EnsureRecording(ctx, testLessonID))

	require.Equal(t, 2, lk.Calls["StartRoomRecording"], "qayta urinish LiveKit'ga yetib borishi kerak")
	recs, err := rrepo.ListByLesson(ctx, testLessonID)
	require.NoError(t, err)
	require.Len(t, recs, 1, "muvaffaqiyatsiz urinishdan yetim qator qolmasin")
	require.Equal(t, entity.RecordingStatusRecording, recs[0].Status)
}

// ─── Retention (PRODUCT.md №5) ───────────────────────────────────────────────

// Yozuvlar ro'yxatida `expires_at` qaytishi kerak — klient «X kundan keyin
// o'chadi» deb ko'rsatadi. Faqat `ready` yozuvlarda: hali yozilayotgan yoki
// yiqilgan yozuvda o'chadigan narsa yo'q.
func TestListByLesson_SetsExpiresAt(t *testing.T) {
	ctx := context.Background()
	uc, rrepo, _, _ := setup(t)
	ended := time.Now().UTC().Add(-2 * 24 * time.Hour)
	// `telegram_sent_at` TO'LDIRILGAN: Telegram arxivi yoqilganda faqat
	// arxivda tasdiqlangan yozuvning muddati bor. Tasdiqlanmagani esa
	// umuman o'chirilmaydi — `TestListByLesson_NoExpiryUntilArchived` ga qara.
	sent := ended.Add(time.Minute)
	require.NoError(t, rrepo.Create(ctx, &entity.Recording{
		ID: testRecordingID, LessonID: testLessonID, EgressID: "EG-ready",
		Status: entity.RecordingStatusReady, EndedAt: &ended, CreatedAt: ended,
		TelegramSentAt: &sent,
	}))
	require.NoError(t, rrepo.Create(ctx, &entity.Recording{
		ID: "33333333-3333-4333-8333-333333333333", LessonID: testLessonID, EgressID: "EG-live",
		Status: entity.RecordingStatusRecording, CreatedAt: time.Now().UTC(),
	}))

	recs, err := uc.ListByLesson(ctx, "mentor1", testLessonID)
	require.NoError(t, err)
	require.Len(t, recs, 2)

	for _, rec := range recs {
		switch rec.Status {
		case entity.RecordingStatusReady:
			require.NotNil(t, rec.ExpiresAt, "tayyor yozuvda expires_at bo'lishi kerak")
			require.WithinDuration(t, ended.Add(testRetention), *rec.ExpiresAt, time.Second)
		default:
			require.Nil(t, rec.ExpiresAt, "tayyor bo'lmagan yozuvda expires_at bo'lmasligi kerak")
		}
	}
}

// Muddati o'tgan yozuv uchun yuklab olish havolasi berilmaydi va sabab ANIQ
// aytiladi (fayl MinIO'da yo'q — "hali tayyor emas" degan yolg'on xabar emas).
func TestDownloadURL_ExpiredRecording(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusExpired)

	_, err := uc.DownloadURL(context.Background(), "mentor1", testRecordingID)
	require.Error(t, err)
	require.True(t, apperr.IsBadRequest(err))
	require.Contains(t, err.Error(), "expired")
}

// ─── Client-side (lokal) yozuv oqimi ────────────────────────────────────────

func TestLocalRecording_Flow(t *testing.T) {
	uc, rrepo, _, _, tg, mc := setupFull(t)
	tg.Disabled = false // Telegram yoqilgan
	ctx := context.Background()

	// 1. LocalStart → yozuv qatori (status=recording, sintetik egress_id).
	rec, err := uc.LocalStart(ctx, "mentor1", testLessonID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusRecording, rec.Status)
	require.True(t, strings.HasPrefix(rec.EgressID, "local:"), "egress_id sintetik bo'lishi kerak")
	require.NotEmpty(t, rec.ObjectKey)

	// 2. Upload URL (presigned PUT) — obyekt yo'lini o'z ichiga oladi.
	url, err := uc.LocalUploadURL(ctx, "mentor1", rec.ID)
	require.NoError(t, err)
	require.Contains(t, url, rec.ObjectKey)

	// 3. Telefon faylni yukladi (fake MinIO'ga qo'yamiz).
	mc.Objects[rec.ObjectKey] = true
	mc.Contents[rec.ObjectKey] = []byte("phone-recorded-video-bytes")

	// 4. LocalComplete → ready + o'lcham server tekshiruvidan.
	require.NoError(t, uc.LocalComplete(ctx, "mentor1", rec.ID, 120, time.Now().UTC()))
	got, err := rrepo.GetByID(ctx, rec.ID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, got.Status)
	require.Equal(t, int64(len("phone-recorded-video-bytes")), got.SizeBytes)
	require.Equal(t, 120, got.DurationSec)
}

func TestLocalStart_Ownership(t *testing.T) {
	uc, _, _, _, _, _ := setupFull(t)
	_, err := uc.LocalStart(context.Background(), "intruder", testLessonID)
	require.True(t, apperr.IsForbidden(err), "begona mentor rad etilsin")
}

func TestLocalComplete_NoFile_Rejected(t *testing.T) {
	uc, _, _, _, _, _ := setupFull(t)
	ctx := context.Background()
	rec, err := uc.LocalStart(ctx, "mentor1", testLessonID)
	require.NoError(t, err)
	// Fayl MinIO'ga yuklanmagan → "tayyor" deb belgilamaydi (BadRequest).
	err = uc.LocalComplete(ctx, "mentor1", rec.ID, 10, time.Now().UTC())
	require.True(t, apperr.IsBadRequest(err))
}

// ─── Lokal yozuv (client-side) ──────────────────────────────────────────────

func seedLocal(t *testing.T, rrepo *testutil.FakeRecordingRepo, mc *testutil.FakeMinio, status string) *entity.Recording {
	t.Helper()
	rec := &entity.Recording{ID: testRecordingID, LessonID: testLessonID, EgressID: "local:" + testRecordingID,
		ObjectKey: "recordings/" + testLessonID + "/" + testRecordingID + ".mp4", Status: status, StartedAt: time.Now().Add(-time.Hour)}
	require.NoError(t, rrepo.Create(context.Background(), rec))
	mc.Objects[rec.ObjectKey] = true
	return rec
}

func TestLocalStart_ReusesLocalButNotRealEgress(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	first, err := uc.LocalStart(context.Background(), "mentor1", testLessonID)
	require.NoError(t, err)
	again, err := uc.LocalStart(context.Background(), "mentor1", testLessonID)
	require.NoError(t, err)
	require.Equal(t, first.ID, again.ID)

	uc2, rrepo2, _, _ := setup(t)
	_ = rrepo
	seedRecording(t, rrepo2, entity.RecordingStatusRecording) // haqiqiy egress faol
	_, err = uc2.LocalStart(context.Background(), "mentor1", testLessonID)
	require.True(t, apperr.IsConflict(err), "server egress ustiga lokal yozuv ochilmaydi")
}

func TestLocalUploadURL_StatusPrecondition(t *testing.T) {
	uc, rrepo, _, _, _, mc := setupFull(t)
	seedLocal(t, rrepo, mc, entity.RecordingStatusReady)
	_, err := uc.LocalUploadURL(context.Background(), "mentor1", testRecordingID)
	require.True(t, apperr.IsBadRequest(err), "tayyor yozuvga yuklash URL'i berilmaydi")
}

func TestLocalUploadURL_RejectsRealEgress(t *testing.T) {
	uc, rrepo, _ := func() (recording.UseCase, *testutil.FakeRecordingRepo, int) {
		u, r, _, _ := setup(t)
		return u, r, 0
	}()
	seedRecording(t, rrepo, entity.RecordingStatusRecording)
	_, err := uc.LocalUploadURL(context.Background(), "mentor1", testRecordingID)
	require.True(t, apperr.IsBadRequest(err))
}

func TestLocalComplete_ClampsValues(t *testing.T) {
	uc, rrepo, _, _, _, mc := setupFull(t)
	seedLocal(t, rrepo, mc, entity.RecordingStatusRecording)
	far := time.Now().Add(24 * time.Hour)
	require.NoError(t, uc.LocalComplete(context.Background(), "mentor1", testRecordingID, -5, far))
	got, _ := rrepo.GetByID(context.Background(), testRecordingID)
	require.Equal(t, entity.RecordingStatusReady, got.Status)
	require.GreaterOrEqual(t, got.DurationSec, 0)
	require.NotNil(t, got.EndedAt)
	require.False(t, got.EndedAt.After(time.Now().Add(6*time.Minute)), "ended_at kelajakka surilmaydi")
}

func TestStopRecording_LocalRejected(t *testing.T) {
	uc, rrepo, _, lk, _, mc := setupFull(t)
	seedLocal(t, rrepo, mc, entity.RecordingStatusRecording)
	err := uc.StopRecording(context.Background(), "mentor1", testRecordingID)
	require.True(t, apperr.IsBadRequest(err))
	require.Equal(t, 0, lk.Calls["StopRecording"])
}

func TestStopActiveForLesson_SkipsLocal(t *testing.T) {
	uc, rrepo, _, lk, _, mc := setupFull(t)
	seedLocal(t, rrepo, mc, entity.RecordingStatusRecording)
	require.NoError(t, uc.StopActiveForLesson(context.Background(), testLessonID))
	require.Equal(t, 0, lk.Calls["StopRecording"], "lokal yozuvga egress stop chaqirilmaydi")
	got, _ := rrepo.GetByID(context.Background(), testRecordingID)
	require.Equal(t, entity.RecordingStatusRecording, got.Status)
}

func TestReapStaleRecordings(t *testing.T) {
	uc, rrepo, _, _, _, mc := setupFull(t)
	seedLocal(t, rrepo, mc, entity.RecordingStatusRecording) // started 1h ago
	require.Equal(t, 0, uc.ReapStaleRecordings(context.Background(), time.Now().Add(-2*time.Hour)))
	require.Equal(t, 1, uc.ReapStaleRecordings(context.Background(), time.Now()))
	got, _ := rrepo.GetByID(context.Background(), testRecordingID)
	require.Equal(t, entity.RecordingStatusFailed, got.Status)
}

func TestHandleEgress_FailedDeletesPartialObject(t *testing.T) {
	uc, rrepo, _, _, _, mc := setupFull(t)
	rec := seedRecording(t, rrepo, entity.RecordingStatusProcessing)
	mc.Objects[rec.ObjectKey] = true
	require.NoError(t, uc.HandleEgress(context.Background(), "EG1", false, "", 0, 0))
	require.False(t, mc.Objects[rec.ObjectKey], "qisman obyekt o'chirilishi kerak")
}

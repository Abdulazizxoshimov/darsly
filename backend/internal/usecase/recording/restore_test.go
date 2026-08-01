package recording_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
)

// seedArchived — Telegramda saqlangan, serverdan o'chirilgan yozuv.
func seedArchived(t *testing.T, rrepo *testutil.FakeRecordingRepo, fileID string) *entity.Recording {
	t.Helper()
	sent := time.Now().UTC().Add(-31 * 24 * time.Hour)
	rec := &entity.Recording{
		ID: testRecordingID, LessonID: testLessonID, EgressID: "EG-arch",
		ObjectKey: "recordings/" + testLessonID + "/" + testRecordingID + ".mp4",
		Status:    entity.RecordingStatusArchived,
		SizeBytes: 1024,
		CreatedAt: sent,
	}
	if fileID != "" {
		rec.TelegramFileID = &fileID
		rec.TelegramSentAt = &sent
	}
	require.NoError(t, rrepo.Create(context.Background(), rec))
	return rec
}

// Arxivlangan yozuvni yuklab olishga urinish ANIQ sabab beradi: u yo'qolmagan,
// avval tiklanishi kerak. "Hali tayyor emas" degan umumiy xabar mentorni
// kutishga majburlardi — aslida u tugmani bosishi kerak edi.
func TestDownloadURL_Archived(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedArchived(t, rrepo, "file-1")

	_, err := uc.DownloadURL(context.Background(), "mentor1", testRecordingID)
	require.True(t, apperr.IsBadRequest(err))
	require.Contains(t, err.Error(), "archived")
}

// Tiklash 202 + `restoring` qaytaradi (darhol, yuklab olishni kutmasdan).
func TestRestore_ClaimsAndReportsRestoring(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedArchived(t, rrepo, "file-1")

	res, err := uc.Restore(context.Background(), "mentor1", testRecordingID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusRestoring, res.Status)
	require.Positive(t, res.PollAfterS, "klientga poll oralig'i aytilishi kerak")

	got, err := rrepo.GetByID(context.Background(), testRecordingID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusRestoring, got.Status)
}

// ⭐ IDEMPOTENTLIK: ikki marta bosilgan tugma IKKINCHI yuklab olishni
// boshlamaydi. Aks holda ikkita parallel 1 GB yuklash ketardi.
func TestRestore_TwiceIsIdempotent(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedArchived(t, rrepo, "file-1")
	ctx := context.Background()

	first, err := uc.Restore(ctx, "mentor1", testRecordingID)
	require.NoError(t, err)
	second, err := uc.Restore(ctx, "mentor1", testRecordingID)
	require.NoError(t, err, "ikkinchi bosish ham xato emas")

	require.Equal(t, entity.RecordingStatusRestoring, first.Status)
	require.Equal(t, entity.RecordingStatusRestoring, second.Status,
		"ikkinchi so'rov ham 'restoring' beradi (ish allaqachon ketmoqda)")
}

// Allaqachon serverda bo'lgan yozuvni tiklash — xato emas, shunchaki `ready`.
func TestRestore_AlreadyReady(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusReady)

	res, err := uc.Restore(context.Background(), "mentor1", testRecordingID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, res.Status)
}

// Begona mentor tiklay olmaydi (IDOR).
func TestRestore_ForeignMentor(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedArchived(t, rrepo, "file-1")

	_, err := uc.Restore(context.Background(), "mentor2", testRecordingID)
	require.Error(t, err)
	require.False(t, apperr.IsBadRequest(err), "egalik xatosi bo'lishi kerak, so'rov xatosi emas")
}

// To'liq yo'l: `restoring` → Telegramdan yuklab olish → MinIO → `ready` + kesh.
func TestRunRestore_Success(t *testing.T) {
	uc, rrepo, _, _, _, mc := setupFull(t)
	rec := seedArchived(t, rrepo, "file-1")
	ctx := context.Background()

	_, err := uc.Restore(ctx, "mentor1", testRecordingID)
	require.NoError(t, err)
	require.NoError(t, uc.RunRestore(ctx, testRecordingID))

	got, err := rrepo.GetByID(ctx, testRecordingID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, got.Status)
	require.NotNil(t, got.CachedUntil, "tiklangan nusxa KESH — muddati bo'lishi shart")
	require.WithinDuration(t, time.Now().UTC().Add(testCacheTTL), *got.CachedUntil, time.Minute)
	require.True(t, mc.Objects[rec.ObjectKey], "fayl MinIO'ga qaytarilishi kerak")
}

// Tiklash yiqilsa yozuv `archived` ga QAYTADI (`restoring` da qotib qolmaydi)
// va Telegramdagi nusxa tegilmaydi — mentor qayta urina oladi.
func TestRunRestore_FailureReturnsToArchived(t *testing.T) {
	uc, rrepo, _, _, tgFake, _ := setupFull(t)
	seedArchived(t, rrepo, "file-1")
	tgFake.DownloadErr = fmt.Errorf("tarmoq uzildi")
	ctx := context.Background()

	_, err := uc.Restore(ctx, "mentor1", testRecordingID)
	require.NoError(t, err)
	require.Error(t, uc.RunRestore(ctx, testRecordingID))

	got, err := rrepo.GetByID(ctx, testRecordingID)
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusArchived, got.Status,
		"yiqilgan tiklash yozuvni 'restoring' da qoldirmasligi kerak")
	require.NotNil(t, got.TelegramError, "sabab yozilishi kerak")
}

// `file_id` bo'lmasa tiklash umuman boshlanmaydi: qaytaradigan narsa yo'q.
func TestRestore_NoTelegramCopy(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedArchived(t, rrepo, "") // file_id yo'q

	_, err := uc.Restore(context.Background(), "mentor1", testRecordingID)
	require.True(t, apperr.IsBadRequest(err))
}

// ⭐ Telegramda TASDIQLANMAGAN yozuvga `expires_at` qo'yilmaydi: uni retention
// o'chirmaydi, ya'ni "X kundan keyin o'chadi" degan ma'lumot yolg'on bo'lardi.
func TestListByLesson_NoExpiryUntilArchived(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	ctx := context.Background()
	ended := time.Now().UTC().Add(-2 * 24 * time.Hour)
	require.NoError(t, rrepo.Create(ctx, &entity.Recording{
		ID: testRecordingID, LessonID: testLessonID, EgressID: "EG-unconfirmed",
		Status: entity.RecordingStatusReady, EndedAt: &ended, CreatedAt: ended,
	}))

	recs, err := uc.ListByLesson(ctx, "mentor1", testLessonID)
	require.NoError(t, err)
	require.Len(t, recs, 1)
	require.Nil(t, recs[0].ExpiresAt,
		"Telegramga tushmagan yozuv o'chirilmaydi — muddat ko'rsatilmasin")
}

// `HandleEgress` yozuv tayyor bo'lgach uni Telegram navbatiga qo'yadi.
func TestHandleEgress_EnqueuesTelegram(t *testing.T) {
	uc, rrepo, _, _ := setup(t)
	seedRecording(t, rrepo, entity.RecordingStatusProcessing)
	ctx := context.Background()

	require.NoError(t, uc.HandleEgress(ctx, "EG1", true, "recordings/x.mp4", 60, 2048))

	// Navbatdagi yagona yozuv shu bo'lishi kerak.
	claimed, err := rrepo.ClaimTelegramUpload(ctx, time.Now().UTC(), 3)
	require.NoError(t, err)
	require.NotNil(t, claimed, "tayyor yozuv Telegram navbatiga tushishi kerak")
	require.Equal(t, testRecordingID, claimed.ID)
}

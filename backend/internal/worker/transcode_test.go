package worker

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/testutil"
)

func newRec(id string, size int64) *entity.Recording {
	return &entity.Recording{
		ID:        id,
		LessonID:  "l1",
		EgressID:  "eg-" + id,
		ObjectKey: "recordings/" + id + ".mp4",
		Status:    entity.RecordingStatusReady,
		SizeBytes: size,
		CreatedAt: time.Now().UTC(),
	}
}

// Navbat: faqat `ready` yozuv tushadi va bir marta olinadi.
func TestEnqueueAndClaim(t *testing.T) {
	repo := testutil.NewFakeRecordingRepo()
	ctx := context.Background()

	ready := newRec("a", 100)
	failed := newRec("b", 100)
	failed.Status = entity.RecordingStatusFailed

	require.NoError(t, repo.Create(ctx, ready))
	require.NoError(t, repo.Create(ctx, failed))
	require.NoError(t, repo.EnqueueTranscode(ctx, ready.EgressID))
	require.NoError(t, repo.EnqueueTranscode(ctx, failed.EgressID))

	got, err := repo.ClaimTranscode(ctx)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "a", got.ID, "yiqilgan yozuv navbatga tushmasligi kerak")

	// Ikkinchi chaqiruv bo'sh: bitta ish ikki marta bajarilmaydi.
	again, err := repo.ClaimTranscode(ctx)
	require.NoError(t, err)
	require.Nil(t, again)
}

// O'lib qolgan ish (deploy/crash) navbatga qaytadi — aks holda abadiy `running`.
func TestRequeueStale(t *testing.T) {
	repo := testutil.NewFakeRecordingRepo()
	ctx := context.Background()
	rec := newRec("a", 100)
	require.NoError(t, repo.Create(ctx, rec))
	require.NoError(t, repo.EnqueueTranscode(ctx, rec.EgressID))

	_, err := repo.ClaimTranscode(ctx)
	require.NoError(t, err)
	require.Equal(t, entity.TranscodeRunning, repo.TranscodeStatus("a"))

	n, err := repo.RequeueStaleTranscodes(ctx, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, int64(1), n)

	got, err := repo.ClaimTranscode(ctx)
	require.NoError(t, err)
	require.NotNil(t, got, "qaytarilgan ish yana olinishi kerak")
}

// ⭐ ENG MUHIM KAFOLAT: ffmpeg yiqilsa asl fayl TEGILMAYDI.
//
// Muvaffaqiyatsiz qayta kodlash yozuvni yo'q qilsa, dars butunlay yo'qolardi —
// bu hajmni tejashdan beqiyos qimmat.
func TestFailedTranscodeKeepsOriginal(t *testing.T) {
	repo := testutil.NewFakeRecordingRepo()
	mc := testutil.NewFakeMinio()
	ctx := context.Background()

	rec := newRec("a", 5_000_000)
	require.NoError(t, repo.Create(ctx, rec))
	require.NoError(t, repo.EnqueueTranscode(ctx, rec.EgressID))
	// MinIO'da obyekt YO'Q → yuklab olish yiqiladi (ffmpeg'gacha ham yetmaydi).

	w := NewTranscodeWorker(repo, mc, testutil.NewLogger(), DefaultTranscodeConfig())
	done, err := w.step(ctx)
	require.NoError(t, err)
	require.True(t, done)

	require.Equal(t, entity.TranscodeFailed, repo.TranscodeStatus("a"))
	got, err := repo.GetByID(ctx, "a")
	require.NoError(t, err)
	require.Equal(t, entity.RecordingStatusReady, got.Status, "yozuv baribir yuklab olinadigan qolsin")
	require.Equal(t, int64(5_000_000), got.SizeBytes, "hajm o'zgarmasin")
	require.False(t, mc.Objects[rec.ObjectKey], "asl obyekt o'chirilmasin")
}

// Bo'sh navbatda ishchi hech narsa qilmaydi (va yiqilmaydi).
func TestEmptyQueue(t *testing.T) {
	w := NewTranscodeWorker(testutil.NewFakeRecordingRepo(), testutil.NewFakeMinio(), testutil.NewLogger(), DefaultTranscodeConfig())
	done, err := w.step(context.Background())
	require.NoError(t, err)
	require.False(t, done)
}

// ffmpeg haqiqatan chaqiriladi va faylni kichraytiradi.
// ffmpeg yo'q muhitda (CI konteyneri) o'tkazib yuboriladi.
func TestFFmpegShrinksRealFile(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg yo'q — integratsiya testi o'tkazib yuborildi")
	}
	dir := t.TempDir()
	src := dir + "/src.mp4"
	dst := dir + "/out.mp4"

	// 3 soniyalik shovqinli test videosi: shovqin ATAYLAB — bir xil rangli
	// kadr har qanday sozlamada kichkina chiqadi va test hech narsani isbotlamasdi.
	gen := exec.Command("ffmpeg", "-y", "-nostdin",
		"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=30:duration=3",
		"-c:v", "libx264", "-b:v", "3000k", "-preset", "ultrafast", src)
	out, err := gen.CombinedOutput()
	require.NoError(t, err, string(out))

	w := NewTranscodeWorker(nil, nil, testutil.NewLogger(), DefaultTranscodeConfig())
	require.NoError(t, w.runFFmpeg(context.Background(), src, dst))

	srcInfo, err := os.Stat(src)
	require.NoError(t, err)
	dstInfo, err := os.Stat(dst)
	require.NoError(t, err)
	require.Less(t, dstInfo.Size(), srcInfo.Size(), "CRF natijasi kichikroq bo'lishi kerak")
}

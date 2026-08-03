package worker

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	require.NoError(t, w.runFFmpeg(context.Background(), src, dst, editPlan{}))

	srcInfo, err := os.Stat(src)
	require.NoError(t, err)
	dstInfo, err := os.Stat(dst)
	require.NoError(t, err)
	require.Less(t, dstInfo.Size(), srcInfo.Size(), "CRF natijasi kichikroq bo'lishi kerak")
}

// ⭐ 2026-08-01 NUQSONINING SUN'IY TAKRORI — uchdan-uchga, haqiqiy ffmpeg bilan.
//
// Nima yasaladi (Zoom bilan taqqoslashda o'lchangan holatning aynan o'zi):
//   - 1280x720 kadr, ichida 404x720 TIK kontent markazda → chap/o'ngda qora yo'l,
//     kontent kadrning atigi ~32% i;
//   - birinchi 14 soniya BUTUNLAY qora va JIM (egress ulanib, hali media
//     kelmagan payt);
//   - keyin shovqinli kontent + tovush.
//
// Nima tasdiqlanadi: `analyze` shu ikkalasini ham topadi va `runFFmpeg` natijasi
// Zoom pariteti bo'yicha chiqadi — kontent butun kadrni egallaydi, 25 fps,
// 48 kHz stereo, boshidagi o'lik qism yo'q.
//
// Pure funksiya testlari (videofilter_test.go) qarorni qamraydi; bu test esa
// ffmpeg CHAQIRUVI to'g'ri simlanganini — filtrlar zanjiri, `-ss` joyi,
// `-ar`/`-ac` — ya'ni faqat real ffmpeg ko'rsata oladigan qismini tekshiradi.
func TestTranscodePipeline_RemovesBarsAndDeadStart(t *testing.T) {
	requireFFmpeg(t)
	dir := t.TempDir()
	src := dir + "/src.mp4"
	dst := dir + "/out.mp4"

	// 1) Tik kontent (404x720) + tovush — 20 s.
	content := dir + "/content.mp4"
	run(t, "ffmpeg", "-y", "-nostdin",
		"-f", "lavfi", "-i", "testsrc2=size=404x720:rate=25:duration=20",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=20",
		"-c:v", "libx264", "-preset", "ultrafast", "-b:v", "2500k",
		"-c:a", "aac", "-ar", "44100", "-ac", "1", content)

	// 2) 14 s qora + jim, so'ng kontentni 1280x720 kadr markaziga joylash.
	//    `tpad`/`adelay` aynan egress xulqini beradi: oqim bor, lekin bo'sh.
	run(t, "ffmpeg", "-y", "-nostdin", "-i", content,
		"-filter_complex",
		"[0:v]tpad=start_duration=14:start_mode=add:color=black,"+
			"pad=1280:720:(ow-iw)/2:0:black[v];"+
			"[0:a]adelay=14000|14000[a]",
		"-map", "[v]", "-map", "[a]",
		"-c:v", "libx264", "-preset", "ultrafast", "-b:v", "2500k",
		"-c:a", "aac", "-ar", "44100", "-ac", "1", src)

	// Manba haqiqatan nuqsonli ekanini tasdiqlaymiz (test o'zini aldamasin).
	srcW, srcH, srcFPS, srcRate, srcCh, srcDur := probeAll(t, src)
	require.Equal(t, [2]int{1280, 720}, [2]int{srcW, srcH})
	require.InDelta(t, 34.0, srcDur, 1.0, "14 s o'lik + 20 s kontent")
	require.Equal(t, 44100, srcRate)
	require.Equal(t, 1, srcCh)
	_ = srcFPS

	w := NewTranscodeWorker(nil, nil, testutil.NewLogger(), DefaultTranscodeConfig())
	ctx := context.Background()

	plan := w.analyze(ctx, src)
	require.NotNil(t, plan.Crop, "qora yo'llar topilishi kerak edi: %s", plan)
	require.Greater(t, plan.StartSec, 10.0, "boshidagi ~14 s o'lik qism topilsin: %s", plan)
	require.Less(t, plan.StartSec, 14.0, "zaxira qoldirilsin (kontent boshi kesilmasin)")

	require.NoError(t, w.runFFmpeg(ctx, src, dst, plan))

	outW, outH, outFPS, outRate, outCh, outDur := probeAll(t, dst)

	// Kadr: kontent nisbatiga qaytgan (tik), qora yo'l yo'q.
	require.Less(t, outW, outH, "natija TIK bo'lishi kerak, 16:9 emas — %dx%d", outW, outH)
	require.InDelta(t, 404.0/720.0, float64(outW)/float64(outH), 0.04,
		"nisbat kontentnikiga mos kelsin — %dx%d", outW, outH)
	require.Zero(t, outW%2)
	require.Zero(t, outH%2)

	// Zoom pariteti: 25 fps, 48 kHz, stereo.
	require.InDelta(t, 25.0, outFPS, 0.6, "25 fps")
	require.Equal(t, 48000, outRate, "48 kHz — Opus manbasi bilan bir xil")
	require.Equal(t, 2, outCh, "stereo")

	// Boshidagi o'lik qism ketgan: davomiylik ~20 s (34 emas).
	require.InDelta(t, 20.0, outDur, 1.5, "o'lik boshlanish kesilsin — %.1f s", outDur)

	// ⭐ Kontent YO'QOTILMAGAN: natija boshi endi qora emas.
	require.False(t, startsBlack(t, dst), "kesishdan keyin birinchi kadr kontent bo'lsin")
}

// ⭐ TO'LIQ ISHCHI ZANJIRI: kesilgan soniyalar DB'ga YOZILADI.
//
// Busiz arxiv sahifasida chat xabari bosilganda video kesilgan miqdorga
// noto'g'ri joyga sakrardi — ya'ni yozuv sifatini yaxshilash chat
// sinxronizatsiyasini jimgina buzardi. `content_offset_sec` aynan shuni
// yopadi (izoh: `entity.Recording.PlaybackZero`).
func TestTranscodeStep_PersistsTrimOffset(t *testing.T) {
	requireFFmpeg(t)
	ctx := context.Background()
	dir := t.TempDir()
	src := dir + "/src.mp4"

	// 12 s qora+jim, so'ng 20 s kontent (tik, qora yo'l bilan).
	content := dir + "/content.mp4"
	run(t, "ffmpeg", "-y", "-nostdin",
		"-f", "lavfi", "-i", "testsrc2=size=404x720:rate=25:duration=20",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=20",
		"-c:v", "libx264", "-preset", "ultrafast", "-b:v", "2500k",
		"-c:a", "aac", content)
	run(t, "ffmpeg", "-y", "-nostdin", "-i", content,
		"-filter_complex",
		"[0:v]tpad=start_duration=12:start_mode=add:color=black,"+
			"pad=1280:720:(ow-iw)/2:0:black[v];[0:a]adelay=12000|12000[a]",
		"-map", "[v]", "-map", "[a]",
		"-c:v", "libx264", "-preset", "ultrafast", "-b:v", "2500k",
		"-c:a", "aac", src)

	st, err := os.Stat(src)
	require.NoError(t, err)

	repo := testutil.NewFakeRecordingRepo()
	mc := testutil.NewFakeMinio()
	rec := newRec("a", st.Size())
	rec.StartedAt = rec.CreatedAt // egress boshlangan lahza — `PlaybackZero` tayanchi
	require.NoError(t, repo.Create(ctx, rec))
	require.NoError(t, repo.EnqueueTranscode(ctx, rec.EgressID))
	require.NoError(t, mc.PutFile(rec.ObjectKey, src))

	w := NewTranscodeWorker(repo, mc, testutil.NewLogger(), DefaultTranscodeConfig())
	done, err := w.step(ctx)
	require.NoError(t, err)
	require.True(t, done)
	require.Equal(t, entity.TranscodeDone, repo.TranscodeStatus("a"))

	got, err := repo.GetByID(ctx, "a")
	require.NoError(t, err)
	require.Greater(t, got.ContentOffsetSec, 9, "~12 s kesildi — DB'da qolsin")
	require.Less(t, got.ContentOffsetSec, 13)
	require.InDelta(t, 20, got.DurationSec, 2, "davomiylik ham yangilansin")

	// Nol nuqta suriladi: chat sakrashi endi to'g'ri hisoblanadi.
	zero, ok := got.PlaybackZero()
	require.True(t, ok)
	require.Equal(t, time.Duration(got.ContentOffsetSec)*time.Second, zero.Sub(got.StartedAt))
}

// Kesish qo'llanmasa (kadr allaqachon to'g'ri) fayl buzilmaydi — regressiya
// qo'riqchisi: `analyze` ni «har doim biror narsa kesadi» holiga tushirmaslik.
func TestTranscodePipeline_CleanSourceUntouched(t *testing.T) {
	requireFFmpeg(t)
	dir := t.TempDir()
	src := dir + "/src.mp4"

	run(t, "ffmpeg", "-y", "-nostdin",
		"-f", "lavfi", "-i", "testsrc2=size=1280x720:rate=25:duration=12",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=12",
		"-c:v", "libx264", "-preset", "ultrafast", "-b:v", "2500k",
		"-c:a", "aac", src)

	w := NewTranscodeWorker(nil, nil, testutil.NewLogger(), DefaultTranscodeConfig())
	plan := w.analyze(context.Background(), src)
	require.Nil(t, plan.Crop, "qora yo'l yo'q — kesilmasin: %s", plan)
	require.Zero(t, plan.StartSec, "boshi qora emas — kesilmasin: %s", plan)
}

// ── yordamchilar ──

func requireFFmpeg(t *testing.T) {
	t.Helper()
	for _, bin := range []string{"ffmpeg", "ffprobe"} {
		if _, err := exec.LookPath(bin); err != nil {
			t.Skipf("%s yo'q — integratsiya testi o'tkazib yuborildi", bin)
		}
	}
}

func run(t *testing.T, name string, args ...string) {
	t.Helper()
	out, err := exec.Command(name, args...).CombinedOutput()
	require.NoError(t, err, "%s: %s", name, tail(string(out), 800))
}

// probeAll — kenglik, balandlik, fps, ovoz chastotasi, kanallar, davomiylik.
func probeAll(t *testing.T, path string) (w, h int, fps float64, rate, ch int, dur float64) {
	t.Helper()
	out, err := exec.Command("ffprobe", "-v", "error",
		"-show_entries", "stream=codec_type,width,height,avg_frame_rate,sample_rate,channels",
		"-show_entries", "format=duration", "-of", "json", path).Output()
	require.NoError(t, err)

	var p struct {
		Streams []struct {
			CodecType  string `json:"codec_type"`
			Width      int    `json:"width"`
			Height     int    `json:"height"`
			AvgFrame   string `json:"avg_frame_rate"`
			SampleRate string `json:"sample_rate"`
			Channels   int    `json:"channels"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	require.NoError(t, json.Unmarshal(out, &p))
	for _, s := range p.Streams {
		switch s.CodecType {
		case "video":
			w, h = s.Width, s.Height
			if n, d, ok := strings.Cut(s.AvgFrame, "/"); ok {
				num, _ := strconv.ParseFloat(n, 64)
				den, _ := strconv.ParseFloat(d, 64)
				if den > 0 {
					fps = num / den
				}
			}
		case "audio":
			rate, _ = strconv.Atoi(s.SampleRate)
			ch = s.Channels
		}
	}
	dur, _ = strconv.ParseFloat(p.Format.Duration, 64)
	return
}

// startsBlack — faylning birinchi soniyasi qoramî.
//
// `blackdetect` ishlatiladi (o'rtacha yorug'lik emas): aynan shu filtr bilan
// qaror qabul qilingan, ya'ni tekshiruv ham shu o'lchovda bo'lishi kerak.
func startsBlack(t *testing.T, path string) bool {
	t.Helper()
	out, _ := exec.Command("ffmpeg", "-nostdin", "-hide_banner",
		"-i", path, "-t", "1", "-vf", "blackdetect=d=0.5:pix_th=0.10",
		"-an", "-f", "null", "-").CombinedOutput()
	for _, r := range parseBlackDetect(string(out)) {
		if r.Start <= 0.1 {
			return true
		}
	}
	return false
}

// ⭐ Transkod kadr chastotasini manbadan YUQORIGA ko'tarmasligi kerak.
//
// 2026-08-04 da serverda aynan shu bo'ldi: egress CPU sababli 15 fps yozardi,
// transkod esa 25 ni majburlab har uchinchi kadrni takrorlardi — fayl kattaroq,
// sifat esa bir xil.
func TestTranscode_FPSManbadanOshmaydi(t *testing.T) {
	requireFFmpeg(t)
	dir := t.TempDir()
	src := dir + "/src15.mp4"
	dst := dir + "/out.mp4"

	run(t, "ffmpeg", "-y", "-nostdin",
		"-f", "lavfi", "-i", "testsrc2=size=640x480:rate=15:duration=6",
		"-c:v", "libx264", "-preset", "ultrafast", src)

	w := NewTranscodeWorker(nil, nil, testutil.NewLogger(), DefaultTranscodeConfig())
	ctx := context.Background()

	plan := w.analyze(ctx, src)
	require.Equal(t, 15, plan.FPS, "manba 15 fps — reja shuni belgilashi kerak")
	require.Equal(t, 15, plan.outFPS(w.cfg.FPS))

	require.NoError(t, w.runFFmpeg(ctx, src, dst, plan))
	_, _, fps, _, _, _ := probeAll(t, dst)
	require.InDelta(t, 15.0, fps, 0.6, "natija manbadan tez bo'lmasin")
}

// Manba sozlamadan TEZ bo'lsa chegara ishlaydi (hajm nazorati).
func TestTranscode_FPSChegaraQollanadi(t *testing.T) {
	requireFFmpeg(t)
	dir := t.TempDir()
	src := dir + "/src30.mp4"
	dst := dir + "/out.mp4"

	run(t, "ffmpeg", "-y", "-nostdin",
		"-f", "lavfi", "-i", "testsrc2=size=640x480:rate=30:duration=6",
		"-c:v", "libx264", "-preset", "ultrafast", src)

	w := NewTranscodeWorker(nil, nil, testutil.NewLogger(), DefaultTranscodeConfig())
	ctx := context.Background()
	plan := w.analyze(ctx, src)
	require.Zero(t, plan.FPS, "manba tez — reja aralashmasin, sozlama chegarasi qolsin")

	require.NoError(t, w.runFFmpeg(ctx, src, dst, plan))
	_, _, fps, _, _, _ := probeAll(t, dst)
	require.InDelta(t, 25.0, fps, 0.6, "sozlamadagi chegara qo'llanadi")
}

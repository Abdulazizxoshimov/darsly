package worker

import (
	"context"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// TranscodeConfig — ffmpeg parametrlari.
//
// ## Nega CRF, bitrate emas (Zoom naqshi)
// Egress faylni JONLI yozadi va unga aniq bitrate beriladi: statik slaydga ham,
// harakatli videoga ham bir xil oqim ketadi. CRF esa SIFATNI qat'iy ushlaydi va
// bitrate'ni mazmunga qarab o'zgartiradi — dars slaydlari deyarli bepul bo'ladi.
// Zoom ham xuddi shunday qiladi: uchrashuv tugagach yozuvni qayta kodlaydi.
//
// CRF 26: 720p ekran/slayd uchun matn ravshan qoladi (23 — deyarli vizual
// yo'qotishsiz, 28 — kamera tasvirida yumshoqlik sezila boshlaydi).
// `veryfast` preset: 4 yadroli VPS'da real vaqtdan tez ishlaydi, ya'ni
// 2.5 soatlik dars ~20-30 daqiqada qayta kodlanadi va navbat to'planmaydi.
type TranscodeConfig struct {
	Enabled bool
	CRF     int
	Preset  string
	FPS     int
	// AudioKbps — nutq uchun; 64k AAC dars yozuvida yetarli.
	AudioKbps int
	// AudioRateHz / AudioChannels — Zoom pariteti (docs/PRODUCT.md «Yozuv sifati»):
	// Zoom yozuvi 48 kHz stereo. WebRTC/Opus manbasi ham 48 kHz, ya'ni 44.1 ga
	// tushirish faqat zarar qilardi.
	//
	// Stereo mono manbaga BEPUL yaqin: AAC kanallarni mid/side kodlaydi va
	// bir xil ikki kanalda side deyarli nol bit oladi. Ya'ni bu "pariteti
	// uchun hajm to'lash" emas.
	AudioRateHz   int
	AudioChannels int
	// CropBars — qora yo'llarni (pillarbox/letterbox) `cropdetect` bilan
	// aniqlab kesish. Qaror `decideCrop` da (videofilter.go).
	CropBars bool
	// TrimLead — yozuv boshidagi qora VA jim qismni kesish (`leadingDeadSeconds`).
	TrimLead bool
	// TempDir — vaqtinchalik fayllar. Bo'sh bo'lsa OS temp'i.
	TempDir string
	// Threads — ffmpeg necha yadro ishlatsin (0 = cheksiz, hammasini oladi).
	//
	// 4-yadroli VPS'da 2 → jonli LiveKit SFU va API uchun kamida 2 yadro DOIM
	// bo'sh qoladi. Shu bilan 100+ ishtirokchili dars transkod ketayotganda ham
	// sekinlashmaydi (transkod jonli sig'imni tushirmaydi — foydalanuvchi talabi).
	Threads int
	// NiceLevel — ffmpeg jarayoni ustuvorligi (0..19). 19 = eng past: Linux
	// rejalashtiruvchisi jonli trafikka (LiveKit/API) ustunlik beradi, ffmpeg
	// faqat BO'SH CPU'ni oladi. Server band bo'lsa transkod sekinlashadi, lekin
	// darsni sekinlashtirmaydi. `nice`/`ionice` bo'lmasa e'tiborsiz qolinadi.
	NiceLevel int
	// Interval — navbatni tekshirish davri.
	Interval time.Duration
	// StaleAfter — `running` da qotib qolgan ish shu muddatdan keyin navbatga qaytadi.
	StaleAfter time.Duration
}

func DefaultTranscodeConfig() TranscodeConfig {
	return TranscodeConfig{
		Enabled: true,
		CRF:     26,
		Preset:  "veryfast",
		// FPS — bu YUQORI CHEGARA, majburiy qiymat emas.
		//
		// Manba (egress) undan past bo'lsa `analyze` chiqishni manbaga
		// tenglashtiradi (`editPlan.FPS`). Aks holda ffmpeg kadrlarni
		// TAKRORLAB 25 ga yetkazardi: fayl kattaroq, sifat esa bir xil —
		// 2026-08-04 da serverda aynan shunday bo'lgan (egress 15, natija 25).
		//
		// 25 chegara sifatida qoladi: kuchli serverda `RECORDING_FPS=25`
		// qo'yilsa transkod ham avtomatik 25 chiqaradi, kod o'zgarmaydi.
		FPS:           25,
		AudioKbps:     64,
		AudioRateHz:   48000,
		AudioChannels: 2,
		CropBars:      true,
		TrimLead:      true,
		// Jonli sig'imni himoya qilish: transkod eng ko'pi 2 yadro + eng past
		// ustuvorlik — 100+ ishtirokchi transkod paytida ham sekinlashmaydi.
		Threads:    2,
		NiceLevel:  19,
		Interval:   30 * time.Second,
		StaleAfter: 3 * time.Hour,
	}
}

// TranscodeWorker tayyor yozuvlarni ffmpeg bilan qayta kodlab, MinIO'dagi
// faylni almashtiradi.
//
// ## Kafolat: yozuv HECH QACHON yo'qolmaydi
// Yangi fayl faqat ffmpeg muvaffaqiyatli tugagach va u haqiqatan kichikroq
// bo'lsagina yuklanadi. Har qanday xatoda asl fayl MinIO'da tegilmagan qoladi
// va ustoz uni baribir yuklab oladi — faqat `transcode_status` `failed` bo'ladi.
type TranscodeWorker struct {
	repo  repository.RecordingRepository
	minio minio.Client
	log   logger.Logger
	cfg   TranscodeConfig
}

func NewTranscodeWorker(
	repo repository.RecordingRepository,
	mc minio.Client,
	log logger.Logger,
	cfg TranscodeConfig,
) *TranscodeWorker {
	return &TranscodeWorker{repo: repo, minio: mc, log: log, cfg: cfg}
}

// FFmpegAvailable — ffmpeg PATH'da bormi. Yo'q bo'lsa ishchi ishga tushmaydi
// (har 30 soniyada yiqilib log to'ldirgandan ko'ra, boshida bir marta aytish).
func FFmpegAvailable() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func (w *TranscodeWorker) Run(ctx context.Context) {
	if !w.cfg.Enabled {
		w.log.Info(ctx, "transcode worker disabled")
		return
	}
	if !FFmpegAvailable() {
		w.log.Warn(ctx, "transcode worker: ffmpeg not found in PATH — yozuvlar qayta kodlanmaydi")
		return
	}
	w.log.Info(ctx, "transcode worker started",
		logger.String("crf", fmt.Sprint(w.cfg.CRF)),
		logger.String("preset", w.cfg.Preset),
	)

	// Ishga tushishda: oldingi nusxa o'lganда `running` da qolgan ishlarni qaytaramiz.
	if n, err := w.repo.RequeueStaleTranscodes(ctx, time.Now().UTC().Add(-w.cfg.StaleAfter)); err != nil {
		w.log.Warn(ctx, "transcode: requeue stale failed", logger.SafeString("err", err.Error()))
	} else if n > 0 {
		w.log.Info(ctx, "transcode: stale jobs requeued", logger.String("count", fmt.Sprint(n)))
	}

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Navbat bo'shaguncha ketma-ket ishlaymiz, LEKIN bittalab:
			// ffmpeg CPU'ni to'liq yeydi va parallel ish API'ni sekinlashtirardi.
			for {
				done, err := w.step(ctx)
				if err != nil || !done || ctx.Err() != nil {
					break
				}
			}
		}
	}
}

// step navbatdan bitta ishni oladi va bajaradi. `false` — navbat bo'sh.
func (w *TranscodeWorker) step(ctx context.Context) (bool, error) {
	rec, err := w.repo.ClaimTranscode(ctx)
	if err != nil {
		w.log.Warn(ctx, "transcode: claim failed", logger.SafeString("err", err.Error()))
		return false, err
	}
	if rec == nil {
		return false, nil
	}

	start := time.Now()
	res, err := w.process(ctx, rec)
	if err != nil {
		w.log.Warn(ctx, "transcode failed — asl fayl saqlanib qoldi",
			logger.String("recording_id", rec.ID),
			logger.SafeString("err", err.Error()),
		)
		// Bekor qilingan kontekstda DB yozuvi ham o'tmaydi — alohida kontekst.
		fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if e := w.repo.FailTranscode(fctx, rec.ID); e != nil {
			w.log.Error(fctx, "transcode: mark failed error", logger.SafeString("err", e.Error()))
		}
		return true, nil
	}

	if err := w.repo.FinishTranscode(ctx, rec.ID, res.Size, rec.SizeBytes, res.DurationSec, res.TrimmedSec); err != nil {
		w.log.Error(ctx, "transcode: finish write failed", logger.SafeString("err", err.Error()))
		return true, nil
	}
	w.log.Info(ctx, "recording transcoded",
		logger.String("recording_id", rec.ID),
		logger.String("before", humanMB(rec.SizeBytes)),
		logger.String("after", humanMB(res.Size)),
		logger.String("edits", res.Note),
		logger.String("took", time.Since(start).Round(time.Second).String()),
	)
	return true, nil
}

// transcodeResult — bajarilgan ish natijasi.
type transcodeResult struct {
	Size int64
	// DurationSec — natija davomiyligi. 0 → DB'dagi qiymat o'zgarmaydi
	// (boshi kesilmagan, ya'ni eski davomiylik hamon to'g'ri).
	DurationSec int
	// TrimmedSec — fayl boshidan kesilgan soniyalar. Arxivdagi chat sakrashi
	// shu qadar surilishi kerak ([entity.Recording.PlaybackZero]).
	TrimmedSec int
	// Note — jurnal uchun qisqa izoh ("crop=720x1280+280+0 trim=13.4s").
	Note string
}

// process: MinIO → vaqtinchalik fayl → tahlil → ffmpeg → MinIO (ustiga).
func (w *TranscodeWorker) process(ctx context.Context, rec *entity.Recording) (transcodeResult, error) {
	var res transcodeResult
	dir, err := os.MkdirTemp(w.cfg.TempDir, "darsly-transcode-")
	if err != nil {
		return res, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(dir) // katta fayllar diskda qolib ketmasin

	src := filepath.Join(dir, "src.mp4")
	dst := filepath.Join(dir, "out.mp4")

	if err := w.download(ctx, rec.ObjectKey, src); err != nil {
		return res, err
	}

	// Tahlil MAJBURIY EMAS: yiqilsa oddiy (kesishsiz) qayta kodlash ketadi.
	// Yozuvni qayta kodlamay qoldirishdan ko'ra qora yo'lli qoldirgan yaxshi.
	plan := w.analyze(ctx, src)
	res.Note = plan.String()

	if err := w.runFFmpeg(ctx, src, dst, plan); err != nil {
		return res, err
	}

	st, err := os.Stat(dst)
	if err != nil {
		return res, fmt.Errorf("stat result: %w", err)
	}
	// Natija kattaroq bo'lsa almashtirish odatda ZARAR: fayl ham o'sadi, sifat
	// ham tushadi (ikki marta kodlangan bo'ladi).
	//
	// ISTISNO — kesish qo'llangan hol. Unda qayta kodlashning maqsadi hajm emas,
	// KOMPOZITSIYA: qora yo'llar olib tashlanadi va kontent butun kadrni
	// egallaydi. Bunda natija bir oz kattarishi mumkin (endi bitlar qora
	// piksellarga emas, matnga ketadi) va bu KUTILGAN. Faqat portlash
	// (`cropGrowthLimit` dan ortiq o'sish) rad etiladi.
	limit := rec.SizeBytes
	if plan.edits() {
		limit = int64(float64(rec.SizeBytes) * cropGrowthLimit)
	}
	if rec.SizeBytes > 0 && st.Size() >= limit {
		return res, fmt.Errorf("natija kichrayamadi (%s → %s)", humanMB(rec.SizeBytes), humanMB(st.Size()))
	}

	f, err := os.Open(dst)
	if err != nil {
		return res, fmt.Errorf("open result: %w", err)
	}
	defer f.Close()

	// Ustiga yozamiz: havola (`object_key`) o'zgarmaydi, ya'ni allaqachon
	// berilgan presigned URL ham ishlayveradi.
	if _, err := w.minio.Upload(ctx, rec.ObjectKey, "video/mp4", f, st.Size()); err != nil {
		return res, fmt.Errorf("upload result: %w", err)
	}
	res.Size = st.Size()
	// Davomiylik va siljish FAQAT boshi kesilganda o'zgaradi — aks holda
	// DB'dagi (webhook'dan kelgan) qiymatlar tegilmaydi.
	//
	// Siljish AYNAN shu yerda, yuklashdan KEYIN yoziladi: yuklash yiqilsa
	// MinIO'da hamon ESKI (kesilmagan) fayl turadi va siljishni saqlash
	// arxivdagi chatni buzardi.
	if plan.StartSec > 0 {
		res.TrimmedSec = int(math.Round(plan.StartSec))
		if d := probeDuration(ctx, dst); d > 0 {
			res.DurationSec = int(d)
		}
	}
	return res, nil
}

// cropGrowthLimit — kesish qo'llanganda natija asl hajmning shuncha barobaridan
// oshmasligi kerak. 1.25 — qora yo'llar o'rniga kontentga ketgan bitlar uchun
// yetarli zaxira, ammo ffmpeg butunlay noto'g'ri ishlagan holni ushlaydi.
const cropGrowthLimit = 1.25

func (w *TranscodeWorker) download(ctx context.Context, key, path string) error {
	obj, err := w.minio.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer obj.Close()

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	defer f.Close()

	if _, err := copyCtx(ctx, f, obj); err != nil {
		return fmt.Errorf("download copy: %w", err)
	}
	return f.Sync()
}

func (w *TranscodeWorker) runFFmpeg(ctx context.Context, src, dst string, plan editPlan) error {
	args := []string{"-y", "-nostdin"}
	// `-ss` INPUT'dan OLDIN: ffmpeg faylni shu nuqtaga tez o'tkazadi va chiqish
	// vaqt belgilari noldan boshlanadi (baribir qayta kodlanadi, ya'ni kesish
	// kalit kadrga qadalib qolmaydi).
	if plan.StartSec > 0 {
		args = append(args, "-ss", strconv.FormatFloat(plan.StartSec, 'f', 2, 64))
	}
	args = append(args, "-i", src)

	vf := ""
	if plan.Crop != nil {
		vf = fmt.Sprintf("crop=%d:%d:%d:%d,", plan.Crop.W, plan.Crop.H, plan.Crop.X, plan.Crop.Y)
	}
	vf += fmt.Sprintf("fps=%d", plan.outFPS(w.cfg.FPS))

	if w.cfg.Threads > 0 {
		// Yadro chegarasi: jonli LiveKit/API uchun yadro doim bo'sh qolsin.
		args = append(args, "-threads", strconv.Itoa(w.cfg.Threads))
	}
	args = append(args,
		"-c:v", "libx264",
		"-crf", fmt.Sprint(w.cfg.CRF),
		"-preset", w.cfg.Preset,
		"-vf", vf,
		"-pix_fmt", "yuv420p", // eski pleyerlar va Safari uchun
		"-c:a", "aac",
		"-b:a", fmt.Sprintf("%dk", w.cfg.AudioKbps),
		"-ar", fmt.Sprint(w.audioRate()),
		"-ac", fmt.Sprint(w.audioChannels()),
		"-movflags", "+faststart", // brauzerda darhol o'ynasin (moov boshda)
		dst,
	)
	cmd := w.lowPriorityFFmpeg(ctx, args)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, tail(string(out), 500))
	}
	return nil
}

// lowPriorityFFmpeg ffmpeg'ni PAST ustuvorlikda ishga tushiradi: `nice` (CPU) +
// `ionice -c 3` (disk I/O idle). Shunda jonli dars trafigi (LiveKit SFU, API)
// har doim ustun bo'ladi va transkod faqat bo'sh resursni oladi — 100+ jonli
// ishtirokchi transkod paytida ham sekinlashmaydi.
//
// `nice`/`ionice` bo'lmasa (masalan lokal dev/macOS) e'tiborsiz — oddiy ffmpeg.
// Har biri o'zini keyingi buyruq bilan almashtiradi (exec), ya'ni PID bir xil
// qoladi va `CommandContext` bekor qilinsa ffmpeg ham to'xtaydi.
func (w *TranscodeWorker) lowPriorityFFmpeg(ctx context.Context, ffArgs []string) *exec.Cmd {
	full := append([]string{"ffmpeg"}, ffArgs...)
	if w.cfg.NiceLevel > 0 {
		if _, err := exec.LookPath("ionice"); err == nil {
			full = append([]string{"ionice", "-c", "3"}, full...)
		}
		if _, err := exec.LookPath("nice"); err == nil {
			full = append([]string{"nice", "-n", strconv.Itoa(w.cfg.NiceLevel)}, full...)
		}
	}
	return exec.CommandContext(ctx, full[0], full[1:]...)
}

// audioRate / audioChannels — nolinchi konfiguratsiyada ham yaroqli qiymat
// (test yoki qisman to'ldirilgan config ffmpeg'ni `-ar 0` bilan yiqitmasin).
func (w *TranscodeWorker) audioRate() int {
	if w.cfg.AudioRateHz <= 0 {
		return 48000
	}
	return w.cfg.AudioRateHz
}

func (w *TranscodeWorker) audioChannels() int {
	if w.cfg.AudioChannels != 1 && w.cfg.AudioChannels != 2 {
		return 2
	}
	return w.cfg.AudioChannels
}

func humanMB(b int64) string { return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024)) }

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

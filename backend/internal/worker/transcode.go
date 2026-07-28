package worker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	// TempDir — vaqtinchalik fayllar. Bo'sh bo'lsa OS temp'i.
	TempDir string
	// Interval — navbatni tekshirish davri.
	Interval time.Duration
	// StaleAfter — `running` da qotib qolgan ish shu muddatdan keyin navbatga qaytadi.
	StaleAfter time.Duration
}

func DefaultTranscodeConfig() TranscodeConfig {
	return TranscodeConfig{
		Enabled:    true,
		CRF:        26,
		Preset:     "veryfast",
		FPS:        15,
		AudioKbps:  64,
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
	newSize, err := w.process(ctx, rec)
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

	if err := w.repo.FinishTranscode(ctx, rec.ID, newSize, rec.SizeBytes); err != nil {
		w.log.Error(ctx, "transcode: finish write failed", logger.SafeString("err", err.Error()))
		return true, nil
	}
	w.log.Info(ctx, "recording transcoded",
		logger.String("recording_id", rec.ID),
		logger.String("before", humanMB(rec.SizeBytes)),
		logger.String("after", humanMB(newSize)),
		logger.String("took", time.Since(start).Round(time.Second).String()),
	)
	return true, nil
}

// process: MinIO → vaqtinchalik fayl → ffmpeg → MinIO (ustiga).
func (w *TranscodeWorker) process(ctx context.Context, rec *entity.Recording) (int64, error) {
	dir, err := os.MkdirTemp(w.cfg.TempDir, "darsly-transcode-")
	if err != nil {
		return 0, fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(dir) // katta fayllar diskda qolib ketmasin

	src := filepath.Join(dir, "src.mp4")
	dst := filepath.Join(dir, "out.mp4")

	if err := w.download(ctx, rec.ObjectKey, src); err != nil {
		return 0, err
	}
	if err := w.runFFmpeg(ctx, src, dst); err != nil {
		return 0, err
	}

	st, err := os.Stat(dst)
	if err != nil {
		return 0, fmt.Errorf("stat result: %w", err)
	}
	// Natija kattaroq bo'lsa almashtirish ZARAR: fayl ham o'sadi, sifat ham
	// tushadi (ikki marta kodlangan bo'ladi). Bunday hol harakatli video yoki
	// juda qisqa yozuvda bo'lishi mumkin.
	if st.Size() >= rec.SizeBytes {
		return 0, fmt.Errorf("natija kichrayamadi (%s → %s)", humanMB(rec.SizeBytes), humanMB(st.Size()))
	}

	f, err := os.Open(dst)
	if err != nil {
		return 0, fmt.Errorf("open result: %w", err)
	}
	defer f.Close()

	// Ustiga yozamiz: havola (`object_key`) o'zgarmaydi, ya'ni allaqachon
	// berilgan presigned URL ham ishlayveradi.
	if _, err := w.minio.Upload(ctx, rec.ObjectKey, "video/mp4", f, st.Size()); err != nil {
		return 0, fmt.Errorf("upload result: %w", err)
	}
	return st.Size(), nil
}

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

func (w *TranscodeWorker) runFFmpeg(ctx context.Context, src, dst string) error {
	args := []string{
		"-y", "-nostdin",
		"-i", src,
		"-c:v", "libx264",
		"-crf", fmt.Sprint(w.cfg.CRF),
		"-preset", w.cfg.Preset,
		"-vf", fmt.Sprintf("fps=%d", w.cfg.FPS),
		"-pix_fmt", "yuv420p", // eski pleyerlar va Safari uchun
		"-c:a", "aac",
		"-b:a", fmt.Sprintf("%dk", w.cfg.AudioKbps),
		"-ac", "1", // nutq — mono; stereo ikki barobar joy yeydi, foyda yo'q
		"-movflags", "+faststart", // brauzerda darhol o'ynasin (moov boshda)
		dst,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, tail(string(out), 500))
	}
	return nil
}

func humanMB(b int64) string { return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024)) }

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}

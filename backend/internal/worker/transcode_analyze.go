package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zoom/darsly/internal/pkg/logger"
)

// Bu fayl — transkoddan OLDINGI tahlil: ffprobe/ffmpeg'dan o'lchov olib,
// `videofilter.go` dagi sof funksiyalarga uzatadi va tahrir rejasini yasaydi.
//
// Qoida: bu yerdagi HAR QANDAY xato jimgina yutiladi va bo'sh reja qaytadi.
// Tahlil yiqilsa yozuv oddiy (kesishsiz) qayta kodlanadi — bu yomonlashish
// emas, avvalgi xulq. Tahlil xatosi tufayli yozuvni umuman qayta kodlamaslik
// yoki, undan battari, noto'g'ri kesish esa mazmun yo'qotish bo'lardi.

// editPlan — ffmpeg'ga beriladigan tahrirlar.
type editPlan struct {
	// Crop — qora yo'llarni kesish (nil → kesilmaydi).
	Crop *CropRect
	// StartSec — boshidan necha soniya tashlanadi (0 → tashlanmaydi).
	StartSec float64
}

func (p editPlan) edits() bool { return p.Crop != nil || p.StartSec > 0 }

func (p editPlan) String() string {
	var parts []string
	if p.Crop != nil {
		parts = append(parts, fmt.Sprintf("crop=%dx%d+%d+%d", p.Crop.W, p.Crop.H, p.Crop.X, p.Crop.Y))
	}
	if p.StartSec > 0 {
		parts = append(parts, fmt.Sprintf("trim=%.1fs", p.StartSec))
	}
	if len(parts) == 0 {
		return "yo'q"
	}
	return strings.Join(parts, " ")
}

// FFprobeAvailable — ffprobe PATH'da bormi. Yo'q bo'lsa tahlil o'tkazib
// yuboriladi (kesish yo'q), qayta kodlash esa ishlayveradi.
func FFprobeAvailable() bool {
	_, err := exec.LookPath("ffprobe")
	return err == nil
}

// mediaInfo — ffprobe'dan olingan minimal ma'lumot.
type mediaInfo struct {
	Width, Height int
	Duration      float64
	HasAudio      bool
}

// analyze manba faylni o'lchab tahrir rejasini qaytaradi.
func (w *TranscodeWorker) analyze(ctx context.Context, src string) editPlan {
	var plan editPlan
	if !w.cfg.CropBars && !w.cfg.TrimLead {
		return plan
	}
	if !FFprobeAvailable() {
		w.log.Warn(ctx, "transcode: ffprobe yo'q — kesish tahlili o'tkazib yuborildi")
		return plan
	}
	info, err := probeMedia(ctx, src)
	if err != nil {
		w.log.Warn(ctx, "transcode: ffprobe yiqildi — kesishsiz davom etamiz",
			logger.SafeString("err", err.Error()))
		return plan
	}

	if w.cfg.CropBars {
		samples := cropSamples(ctx, src, info.Duration)
		if rect, ok := decideCrop(samples, info.Width, info.Height); ok {
			plan.Crop = &rect
		}
	}
	if w.cfg.TrimLead {
		black, silence := detectLead(ctx, src, info.Duration)
		plan.StartSec = leadingDeadSeconds(black, silence, info.Duration, info.HasAudio)
	}
	return plan
}

// probeMedia — kadr o'lchami, davomiylik va ovoz oqimi bormi.
func probeMedia(ctx context.Context, path string) (mediaInfo, error) {
	out, err := exec.CommandContext(ctx, "ffprobe",
		"-v", "error",
		"-show_entries", "stream=codec_type,width,height",
		"-show_entries", "format=duration",
		"-of", "json", path,
	).Output()
	if err != nil {
		return mediaInfo{}, fmt.Errorf("ffprobe: %w", err)
	}
	var parsed struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			Width     int    `json:"width"`
			Height    int    `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return mediaInfo{}, fmt.Errorf("ffprobe json: %w", err)
	}
	var info mediaInfo
	for _, s := range parsed.Streams {
		switch s.CodecType {
		case "video":
			// Maydoni kattasi — asosiy oqim (MP4 da muqova rasmi ham "video"
			// oqim bo'lib ko'rinishi mumkin).
			if s.Width*s.Height > info.Width*info.Height {
				info.Width, info.Height = s.Width, s.Height
			}
		case "audio":
			info.HasAudio = true
		}
	}
	info.Duration, _ = strconv.ParseFloat(parsed.Format.Duration, 64)
	if info.Width <= 0 || info.Height <= 0 {
		return info, fmt.Errorf("ffprobe: video oqimi topilmadi")
	}
	return info, nil
}

// probeDuration — natija faylining davomiyligi (soniya). Xatoda 0.
func probeDuration(ctx context.Context, path string) float64 {
	info, err := probeMedia(ctx, path)
	if err != nil {
		return 0
	}
	return info.Duration
}

// cropSamples — bir NECHTA nuqtadan `cropdetect` o'lchovi.
//
// ⚠️ Bitta nuqta YETARLI EMAS: dars boshida ekran hali ulashilmagan bo'lishi
// yoki slayd bir lahza butunlay qora bo'lishi mumkin — o'sha bitta o'lchov
// butun videoni noto'g'ri kesib qo'yardi. Uch nuqta (10%, 50%, 90%) olinadi
// va `decideCrop` ularning BIRLASHMASINI hisoblaydi.
func cropSamples(ctx context.Context, src string, duration float64) []CropRect {
	var out []CropRect
	for _, off := range sampleOffsets(duration) {
		args := []string{"-nostdin", "-hide_banner"}
		if off > 0 {
			args = append(args, "-ss", strconv.FormatFloat(off, 'f', 2, 64))
		}
		args = append(args,
			"-i", src,
			"-t", strconv.Itoa(cropSampleSecs),
			// limit=24: LiveKit shabloni to'q (dark) fon chizadi, sof qora emas —
			// 24/255 chegara uni ham "bo'sh" deb ko'radi.
			// round=2: natija JUFT (H.264 talabi).
			// reset=0: namuna ichidagi barcha kadrlar birga hisoblanadi.
			"-vf", "cropdetect=limit=24:round=2:reset=0",
			"-an", "-f", "null", "-",
		)
		// Xato bo'lsa ham chiqishni o'qiymiz: ffmpeg qisman ishlab, keyin
		// yiqilgan bo'lishi mumkin va o'sha qismdagi o'lchov ham qimmatli.
		res, _ := exec.CommandContext(ctx, "ffmpeg", args...).CombinedOutput()
		rects := parseCropDetect(string(res))
		if len(rects) == 0 {
			continue
		}
		// Namuna ichidagi OXIRGI qator — o'sha oynadagi to'plangan natija.
		out = append(out, rects[len(rects)-1])
	}
	return out
}

// cropSampleSecs — har bir namuna necha soniya.
const cropSampleSecs = 3

// sampleOffsets — o'lchov nuqtalari (soniya).
func sampleOffsets(duration float64) []float64 {
	if duration <= 0 {
		return []float64{0}
	}
	if duration < 3*cropSampleSecs {
		return []float64{0}
	}
	var out []float64
	for _, frac := range []float64{0.10, 0.50, 0.90} {
		off := duration * frac
		if max := duration - cropSampleSecs; off > max {
			off = max
		}
		if off < 0 {
			off = 0
		}
		out = append(out, off)
	}
	return out
}

// detectLead — yozuv BOSHIDAGI qora va jim oraliqlar.
//
// Faqat birinchi `leadScanSecs` soniya ko'riladi: bizni faqat boshlanish
// qiziqtiradi va 2.5 soatlik faylni to'liq skanerlash bekorga CPU.
func detectLead(ctx context.Context, src string, duration float64) (black, silence []timeRange) {
	res, _ := exec.CommandContext(ctx, "ffmpeg",
		"-nostdin", "-hide_banner",
		"-i", src,
		"-t", strconv.Itoa(leadScanSecs),
		"-vf", "blackdetect=d=0.5:pix_th=0.10",
		"-af", "silencedetect=n=-45dB:d=0.5",
		"-f", "null", "-",
	).CombinedOutput()
	log := string(res)
	scanned := duration
	if scanned > leadScanSecs {
		scanned = leadScanSecs
	}
	return parseBlackDetect(log), parseSilenceDetect(log, scanned)
}

// leadScanSecs — boshlanishni qidirish oynasi. `leadMaxTrim` (60 s) dan
// yetarlicha katta, ya'ni chegaraga yaqin holatlar ham to'liq ko'rinadi.
const leadScanSecs = 120

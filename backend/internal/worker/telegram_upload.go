package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	tg "github.com/zoom/darsly/internal/infrastructure/telegram"
	"github.com/zoom/darsly/internal/pkg/logger"
)

// TelegramUploadConfig — arxivga yuklash ishchisining sozlamalari.
type TelegramUploadConfig struct {
	// Interval — navbatni tekshirish davri.
	Interval time.Duration
	// Backoff — qayta urinish oraliqlari. Uzunligi = MAKSIMAL urinishlar soni.
	//
	// PRODUCT.md: «3 marta ortib boruvchi oraliq». 1m — tarmoqning bir
	// lahzalik uzilishi uchun; 5m — Telegram tomonidagi qisqa nosozlik;
	// 15m — jiddiyroq uzilish. Undan keyin qayta urinishning ma'nosi yo'q:
	// muammo konfiguratsiyada (bot guruhda emas, chat ID noto'g'ri) va uni
	// mentor tuzatishi kerak.
	Backoff []time.Duration
	// TempDir — vaqtinchalik fayllar (bo'sh → OS temp).
	TempDir string
}

func DefaultTelegramUploadConfig() TelegramUploadConfig {
	return TelegramUploadConfig{
		Interval: time.Minute,
		Backoff:  []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute},
	}
}

// Notifier — mentorga ilova ichida bildirishnoma (tor interfeys).
type Notifier interface {
	Notify(ctx context.Context, userID, ntype, title, body string, lessonID *string) (*entity.Notification, error)
}

// TelegramPrompter — yuklash tugagach mentordan «qaysi guruhga?» so'raydi.
//
// Alohida interfeys, chunki bu ishchining ASOSIY vazifasi emas: prompt
// yuborilmasa ham video arxiv guruhida turadi va retention to'g'ri ishlaydi.
// nil bo'lsa jimgina o'tkazib yuboriladi.
type TelegramPrompter interface {
	PromptShare(ctx context.Context, rec *entity.Recording, lesson *entity.Lesson) error
}

// TelegramUploadWorker — tayyor yozuvlarni Telegram arxiv guruhiga yuboradi.
//
// # Nega alohida ishchi (webhook ichida emas)
//
// Yuklash 1 GB gacha fayl va o'nlab daqiqa. Uni LiveKit webhook'i ichida
// bajarish ikki narsani buzardi: webhook 30 s timeout bilan uziladi va
// LiveKit hodisani QAYTA yuboradi — natijada bir yozuv bir necha marta
// yuklanib, guruhda dublikatlar paydo bo'lardi.
//
// # Kafolat
//
// Yuklash muvaffaqiyatsiz bo'lsa server nusxasi HECH QACHON o'chirilmaydi:
// retention `telegram_sent_at IS NOT NULL` shartiga bog'langan (SQL'da).
// Ya'ni bu ishchi butunlay ishlamay qolsa ham hech qanday video yo'qolmaydi —
// faqat disk to'ladi va mentor ogohlantiriladi.
type TelegramUploadWorker struct {
	repo       repository.RecordingRepository
	lessonRepo repository.LessonRepository
	userRepo   repository.UserRepository
	minio      minio.Client
	bot        tg.Client
	notif      Notifier
	prompter   TelegramPrompter
	cfg        TelegramUploadConfig
	log        logger.Logger
}

func NewTelegramUploadWorker(
	repo repository.RecordingRepository,
	lessonRepo repository.LessonRepository,
	userRepo repository.UserRepository,
	mc minio.Client,
	bot tg.Client,
	notif Notifier,
	prompter TelegramPrompter,
	cfg TelegramUploadConfig,
	log logger.Logger,
) *TelegramUploadWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Minute
	}
	if len(cfg.Backoff) == 0 {
		cfg.Backoff = DefaultTelegramUploadConfig().Backoff
	}
	return &TelegramUploadWorker{
		repo: repo, lessonRepo: lessonRepo, userRepo: userRepo, minio: mc, bot: bot,
		notif: notif, prompter: prompter, cfg: cfg, log: log,
	}
}

func (w *TelegramUploadWorker) Run(ctx context.Context) {
	if w.bot == nil || !w.bot.Enabled() {
		w.log.Info(ctx, "telegram upload worker disabled (TELEGRAM_BOT_TOKEN bo'sh)")
		return
	}
	if w.bot.ArchiveChatID() == 0 {
		// Bot bor, lekin arxiv guruhi ko'rsatilmagan. Bu NOSOZLIK emas:
		// asoschi tokenni qo'shib, guruhni keyinroq ulashi mumkin. Lekin
		// ogohlantirish shart — aks holda "yozuvlar arxivlanmayapti" degan
		// holat jimgina davom etardi.
		w.log.Warn(ctx, "telegram upload worker: TELEGRAM_ARCHIVE_CHAT_ID bo'sh — avtomatik arxivlash o'chiq")
		return
	}
	w.log.Info(ctx, "telegram upload worker started",
		logger.String("local_api", strconv.FormatBool(w.bot.Local())),
		logger.String("max_upload_mb", fmt.Sprintf("%.0f", float64(w.bot.MaxUploadBytes())/(1024*1024))),
	)

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Navbat bo'shaguncha BITTALAB: parallel yuklash VPS'ning yuqoriga
			// yo'nalgan kanalini to'ldirib, jonli darsning media oqimini
			// buzardi (ular bir xil tarmoq interfeysida).
			for {
				more, err := w.step(ctx)
				if err != nil || !more || ctx.Err() != nil {
					break
				}
			}
		}
	}
}

// RunOnce — navbatdan BITTA yozuvni oladi va bir urinish qiladi.
//
// Eksport qilingan, chunki qayta urinish siyosati (1m/5m/15m va 3 urinishdan
// keyin taslim) aynan KETMA-KET chaqiruvlar bilan sinaladi; `Run` esa
// cheksiz ticker va uni testda boshqarib bo'lmaydi.
func (w *TelegramUploadWorker) RunOnce(ctx context.Context) {
	_, _ = w.step(ctx)
}

// step — navbatdan bitta yozuvni oladi va yuboradi. `false` — navbat bo'sh.
func (w *TelegramUploadWorker) step(ctx context.Context) (bool, error) {
	maxAttempts := len(w.cfg.Backoff)
	rec, err := w.repo.ClaimTelegramUpload(ctx, time.Now().UTC(), maxAttempts)
	if err != nil {
		w.log.Warn(ctx, "telegram upload: claim failed", logger.SafeString("err", err.Error()))
		return false, err
	}
	if rec == nil {
		return false, nil
	}

	l, lerr := w.lessonRepo.GetByID(ctx, rec.LessonID)
	if lerr != nil {
		// Dars o'chirilgan bo'lsa yozuv ham CASCADE bilan ketardi — bu yerga
		// tushish kutilmagan holat. Yuklashni davom ettiramiz (video muhim),
		// faqat sarlavhasiz.
		w.log.Warn(ctx, "telegram upload: dars topilmadi",
			logger.String("recording_id", rec.ID), logger.SafeString("err", lerr.Error()))
	}

	if err := w.upload(ctx, rec, l); err != nil {
		w.handleFailure(ctx, rec, l, err)
		return true, nil
	}
	return true, nil
}

// upload — MinIO → vaqtinchalik fayl → Telegram arxiv guruhi.
func (w *TelegramUploadWorker) upload(ctx context.Context, rec *entity.Recording, l *entity.Lesson) error {
	if rec.SizeBytes > 0 && rec.SizeBytes > w.bot.MaxUploadBytes() {
		// Yuklab olishdan OLDIN tekshiramiz: 1.5 GB faylni diskka tushirib,
		// keyin "juda katta" deb tashlash bir necha daqiqa va gigabaytlarni
		// bekorga sarflardi.
		return &tg.APIError{
			Code:   400,
			Method: "sendVideo",
			Description: fmt.Sprintf("fayl chegaradan katta: %.0f MB > %.0f MB",
				float64(rec.SizeBytes)/(1024*1024), float64(w.bot.MaxUploadBytes())/(1024*1024)),
		}
	}

	dir, err := os.MkdirTemp(w.cfg.TempDir, "darsly-tg-")
	if err != nil {
		return fmt.Errorf("temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	local := filepath.Join(dir, "lesson.mp4")
	if err := w.download(ctx, rec.ObjectKey, local); err != nil {
		return err
	}

	// Mentor nomi — umumiy jamoa arxivida qaysi ustoz o'tgani ajralib tursin
	// (bir guruhga bir necha mentor darsi tushadi). Xato JIM: nom kosmetik,
	// uni deb yozuvni yubormay qo'yish noto'g'ri bo'lardi.
	mentorName := ""
	if l != nil {
		if u, uerr := w.userRepo.GetByID(ctx, l.MentorID); uerr == nil && u != nil {
			mentorName = strings.TrimSpace(u.FullName)
		}
	}

	msg, err := w.bot.SendVideo(ctx, w.bot.ArchiveChatID(), local, archiveCaption(rec, l, mentorName), rec.DurationSec)
	if err != nil {
		return err
	}

	fileID := ""
	if msg.Video != nil {
		fileID = msg.Video.FileID
	} else if msg.Document != nil {
		// Telegram katta yoki nostandart videoni hujjat sifatida qabul
		// qilishi mumkin — `file_id` baribir keladi va tiklash ishlayveradi.
		fileID = msg.Document.FileID
	}
	if fileID == "" {
		// `file_id` siz yozuvni QAYTARIB BO'LMAYDI. Bu holda "yuborildi" deb
		// belgilash server nusxasini o'chirishga ruxsat berardi va video
		// butunlay yo'qolardi. Shuning uchun bu XATO.
		return fmt.Errorf("telegram javobida file_id yo'q — yozuv tasdiqlanmadi")
	}

	if err := w.repo.MarkTelegramSent(ctx, rec.ID, msg.Chat.ID, msg.MessageID, fileID); err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	w.log.Info(ctx, "recording archived to telegram",
		logger.String("recording_id", rec.ID),
		logger.String("lesson_id", rec.LessonID),
		logger.String("size", humanMB(rec.SizeBytes)),
	)

	// Mentordan «o'quvchilar guruhiga ham yuboraymi?» deb so'raymiz.
	// Xato JIM: video allaqachon arxivda va bu asosiy kafolat bajarildi.
	if w.prompter != nil && l != nil {
		rec.TelegramFileID = &fileID
		if err := w.prompter.PromptShare(ctx, rec, l); err != nil {
			w.log.Warn(ctx, "telegram upload: mentorga so'rov yuborilmadi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		}
	}
	return nil
}

func (w *TelegramUploadWorker) download(ctx context.Context, key, path string) error {
	obj, err := w.minio.Get(ctx, key)
	if err != nil {
		return fmt.Errorf("minio get: %w", err)
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

// handleFailure — qayta urinish rejasini yozadi yoki taslim bo'ladi.
//
// Uch xil xato uch xil qaraladi:
//   - 429 (`retry_after`): urinish SARFLANMAYDI (hisob orqaga qaytariladi
//     emas — buning o'rniga keyingi urinish Telegram aytgan vaqtga suriladi;
//     hisobni kamaytirish DB'da poyga ochardi). Bu vaqtinchalik va bizning
//     aybimiz emas;
//   - doimiy xato (400/403 — bot guruhda emas, fayl katta): darhol taslim,
//     chunki 15 daqiqadan keyin ham natija bir xil bo'ladi;
//   - qolganlari: 1m → 5m → 15m.
func (w *TelegramUploadWorker) handleFailure(ctx context.Context, rec *entity.Recording, l *entity.Lesson, cause error) {
	// Kontekst bekor bo'lgan bo'lishi mumkin (shutdown) — holat baribir yozilsin.
	fctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()

	msg := truncateErr(cause.Error(), 480)

	if after, ok := tg.IsRetryAfter(cause); ok {
		next := time.Now().UTC().Add(after + 5*time.Second)
		w.log.Warn(fctx, "telegram upload: rate limit",
			logger.String("recording_id", rec.ID), logger.String("retry_after", after.String()))
		if err := w.repo.MarkTelegramFailed(fctx, rec.ID, msg, &next); err != nil {
			w.log.Error(fctx, "telegram upload: holatni yozib bo'lmadi", logger.SafeString("err", err.Error()))
		}
		return
	}

	permanent := tg.IsPermanent(cause)
	// `ClaimTelegramUpload` hisobni allaqachon oshirgan, ya'ni `rec.TelegramAttempts`
	// SO'ROVDAN OLDINGI qiymat: hozirgi urinish tartibi = +1.
	attempt := rec.TelegramAttempts + 1
	exhausted := permanent || attempt >= len(w.cfg.Backoff)

	if exhausted {
		w.log.Error(fctx, "telegram upload: urinishlar tugadi — server nusxasi SAQLANADI",
			logger.String("recording_id", rec.ID),
			logger.String("attempts", strconv.Itoa(attempt)),
			logger.SafeString("err", msg))
		if err := w.repo.MarkTelegramFailed(fctx, rec.ID, msg, nil); err != nil {
			w.log.Error(fctx, "telegram upload: holatni yozib bo'lmadi", logger.SafeString("err", err.Error()))
		}
		w.notifyMentor(fctx, rec, l)
		return
	}

	next := time.Now().UTC().Add(w.cfg.Backoff[attempt-1])
	w.log.Warn(fctx, "telegram upload: qayta urinamiz",
		logger.String("recording_id", rec.ID),
		logger.String("attempt", strconv.Itoa(attempt)),
		logger.String("next_in", w.cfg.Backoff[attempt-1].String()),
		logger.SafeString("err", msg))
	if err := w.repo.MarkTelegramFailed(fctx, rec.ID, msg, &next); err != nil {
		w.log.Error(fctx, "telegram upload: holatni yozib bo'lmadi", logger.SafeString("err", err.Error()))
	}
}

// notifyMentor — ilova ichida bildirishnoma (PRODUCT.md: «mentorga bildirishnoma»).
func (w *TelegramUploadWorker) notifyMentor(ctx context.Context, rec *entity.Recording, l *entity.Lesson) {
	if w.notif == nil || l == nil {
		return
	}
	title := "Yozuv Telegramga yuborilmadi"
	body := fmt.Sprintf(
		"«%s» darsining yozuvini Telegram arxiviga yuborib bo'lmadi. Video serverda SAQLANIB QOLDI va o'chirilmaydi — uni ilovadan yuklab olishingiz mumkin.",
		l.Title)
	lessonID := l.ID
	if _, err := w.notif.Notify(ctx, l.MentorID, entity.NotificationTypeTelegramFailed, title, body, &lessonID); err != nil {
		w.log.Warn(ctx, "telegram upload: bildirishnoma yuborilmadi",
			logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
	}
}

// archiveCaption — arxiv guruhidagi xabar matni.
//
// Sana MAJBURIY: arxiv guruhida yuzlab video to'planadi va ularni faqat
// sarlavha bo'yicha ajratib bo'lmaydi (bir dars har hafta takrorlanadi).
func archiveCaption(rec *entity.Recording, l *entity.Lesson, mentorName string) string {
	title := "Dars yozuvi"
	if l != nil && strings.TrimSpace(l.Title) != "" {
		title = l.Title
	}
	when := rec.CreatedAt
	if rec.EndedAt != nil {
		when = *rec.EndedAt
	}
	parts := []string{title, when.Format("2006-01-02 15:04")}
	if rec.DurationSec > 0 {
		parts = append(parts, fmt.Sprintf("%d daq", rec.DurationSec/60))
	}
	// Mentor nomi — umumiy arxivda kim o'tgani ko'rinsin (bo'sh bo'lsa tashlanadi).
	if mentorName != "" {
		parts = append(parts, "👤 "+mentorName)
	}
	return strings.Join(parts, " · ")
}

// truncateErr — xato matnini DB ustuni va Telegram xabari uchun qisqartiradi.
func truncateErr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

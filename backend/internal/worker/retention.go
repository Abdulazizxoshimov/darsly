package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/notification"
)

// RetentionConfig — yozuvlarni saqlash siyosati (PRODUCT.md №5).
type RetentionConfig struct {
	// Retention — yozuv qancha saqlanadi (RECORDING_RETENTION_DAYS, default 30 kun).
	// 0 yoki manfiy → o'chirish BUTUNLAY o'chirilgan (ishchi ishga tushmaydi).
	Retention time.Duration
	// WarnBefore — o'chishdan qancha oldin mentor ogohlantiriladi (default 3 kun).
	// 0 → ogohlantirish yo'q.
	WarnBefore time.Duration
	// Interval — tekshirish oralig'i. Retention kunlar bilan o'lchanadi, ya'ni
	// soatiga bir marta ham yetarli; tez-tez tekshirish faqat DB'ni bezovta qiladi.
	Interval time.Duration
	// BatchLimit — bitta tick'da nechta yozuv qayta ishlanadi. Cheklov bor,
	// chunki har o'chirish MinIO chaqiruvi: eski o'rnatishda birdaniga minglab
	// obyekt o'chirish diskni va tarmoqni cho'ktirardi. Qolganlari keyingi
	// tick'da — retention shoshilinch amal emas.
	BatchLimit uint64
	// TelegramArchive — Telegram arxivi yoqilganmi.
	//
	// ⭐ BU BAYROQ RETENTION MA'NOSINI TUBDAN O'ZGARTIRADI:
	//   false → eski xulq: muddat o'tsa fayl o'chadi va yozuv `expired`
	//           (butunlay yo'qoladi);
	//   true  → yangi qoida: fayl FAQAT Telegramda tasdiqlangan bo'lsa
	//           o'chadi va yozuv `archived` bo'ladi (Telegramdan qaytariladi).
	//           Tasdiqlanmagani esa DISKDA QOLADI va mentor ogohlantiriladi.
	//
	// Ya'ni Telegram sozlanmagan serverda hech narsa o'zgarmaydi, sozlanganida
	// esa «video hech qachon yo'qolmaydi» kafolati kuchga kiradi.
	TelegramArchive bool
}

// DefaultRetentionConfig — PRODUCT.md dagi qiymatlar.
func DefaultRetentionConfig() RetentionConfig {
	return RetentionConfig{
		Retention:  30 * 24 * time.Hour,
		WarnBefore: 3 * 24 * time.Hour,
		Interval:   time.Hour,
		BatchLimit: 100,
	}
}

// RetentionWorker — muddati o'tgan yozuvlarni MinIO'dan o'chiradi va o'chishga
// yaqin yozuvlar haqida mentorni ogohlantiradi.
//
// `ReminderWorker` bilan bir xil naqsh: ro'yxatni repo beradi, har element
// ATOMIK claim bilan band qilinadi (ko'p instansda dublikat yo'q), keyin ish
// bajariladi.
type RetentionWorker struct {
	recRepo    repository.RecordingRepository
	lessonRepo repository.LessonRepository
	minio      minio.Client
	notif      notification.UseCase
	cfg        RetentionConfig
	log        logger.Logger
}

func NewRetentionWorker(
	recRepo repository.RecordingRepository,
	lessonRepo repository.LessonRepository,
	mc minio.Client,
	notif notification.UseCase,
	cfg RetentionConfig,
	log logger.Logger,
) *RetentionWorker {
	if cfg.Interval <= 0 {
		cfg.Interval = time.Hour
	}
	if cfg.BatchLimit == 0 {
		cfg.BatchLimit = 100
	}
	return &RetentionWorker{recRepo: recRepo, lessonRepo: lessonRepo, minio: mc, notif: notif, cfg: cfg, log: log}
}

// Run davriy ravishda retention siyosatini qo'llaydi.
//
// Birinchi tick DARHOL bajariladi: server qayta ishga tushganda muddati o'tgan
// yozuvlar bir soat kutib turmasin (uzoq to'xtab qolgandan keyin ular ko'p
// bo'lishi mumkin).
func (w *RetentionWorker) Run(ctx context.Context) {
	if w.cfg.Retention <= 0 {
		w.log.Info(ctx, "retention worker disabled (RECORDING_RETENTION_DAYS=0)")
		return
	}
	w.log.Info(ctx, "retention worker started",
		logger.String("retention", w.cfg.Retention.String()),
		logger.String("warn_before", w.cfg.WarnBefore.String()),
		logger.String("interval", w.cfg.Interval.String()))

	w.RunOnce(ctx)
	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.RunOnce(ctx)
		}
	}
}

// RunOnce — retention siyosatining BIR tsikli. Eksport qilingan, chunki
// idempotentlik (ogohlantirish bir marta, o'chirish takrorlanmasin) aynan
// KETMA-KET chaqiruvlar bilan sinaladi; `Run` esa cheksiz ticker.
func (w *RetentionWorker) RunOnce(ctx context.Context) {
	now := time.Now().UTC()
	// TARTIB MUHIM: avval ogohlantirish, keyin o'chirish. Teskarisi bo'lsa
	// chegaraviy yozuv (aynan shu tick'da muddati o'tgan) o'chirilib, keyin
	// "3 kundan keyin o'chadi" degan bildirishnoma yuborilardi.
	w.warnExpiring(ctx, now)
	w.deleteExpired(ctx, now)
	if w.cfg.TelegramArchive {
		// Telegramga tushmagan, lekin muddati o'tgan yozuvlar: o'chirilmaydi,
		// mentor ogohlantiriladi (aks holda disk jimgina to'lardi).
		w.warnUnconfirmed(ctx, now)
		// Tiklangan (kesh) nusxalarni muddati bilan tozalash.
		w.evictCache(ctx, now)
	}
}

// recordingEnd — retention hisobining tayanch vaqti.
//
// `COALESCE(ended_at, created_at)` bilan AYNI mantiq (postgres
// `retentionQuery` izohiga qara): `ended_at` bo'sh bo'lishi mumkin va u
// holda yozuv hisobdan butunlay tushib qolmasligi kerak.
func recordingEnd(rec *entity.Recording) time.Time {
	if rec.EndedAt != nil {
		return *rec.EndedAt
	}
	return rec.CreatedAt
}

// warnExpiring — o'chishiga `WarnBefore` qolgan yozuvlar uchun mentorga
// bildirishnoma. `ClaimRetentionWarning` tufayli har yozuv uchun BIR MARTA.
func (w *RetentionWorker) warnExpiring(ctx context.Context, now time.Time) {
	if w.cfg.WarnBefore <= 0 {
		return
	}
	// Muddati `WarnBefore` ichida tugaydiganlar: ended_at <= now - (retention - warn).
	cutoff := now.Add(-(w.cfg.Retention - w.cfg.WarnBefore))
	recs, err := w.recRepo.ListExpiringUnwarned(ctx, cutoff, w.cfg.BatchLimit)
	if err != nil {
		w.log.Warn(ctx, "retention: ogohlantirish ro'yxatini o'qib bo'lmadi", logger.SafeString("err", err.Error()))
		return
	}
	// ⚠️ Muddati ALLAQACHON o'tganlar bu ro'yxatdan chiqariladi.
	//
	// `ListExpiringUnwarned` faqat YUQORI chegara bilan filtrlaydi, ya'ni unga
	// 40 kunlik yozuv ham tushadi. Ogohlantirishsiz qoldirilsa mentor AYNI SHU
	// tick'da ikki xabar olardi: «3 kundan keyin o'chadi» va darhol keyin
	// «o'chirildi/arxivga ko'chdi». Bundan tashqari `ClaimRetentionWarning`
	// belgisi sarflanib, HAQIQIY sabab (masalan «Telegramga tushmagan»)
	// haqidagi ogohlantirish endi yuborilmasdi.
	expiredBefore := now.Add(-w.cfg.Retention)
	for _, rec := range recs {
		if !recordingEnd(rec).After(expiredBefore) {
			continue // bu yozuv bilan `deleteExpired`/`warnUnconfirmed` shug'ullanadi
		}
		l, err := w.lessonRepo.GetByID(ctx, rec.LessonID)
		if err != nil {
			// Dars o'chirilgan bo'lsa yozuv ham CASCADE bilan ketgan bo'lardi;
			// bu yerga tushish — kutilmagan holat, lekin ogohlantirishni
			// kimga yuborishni bilmaymiz. Belgilab qo'yamiz, aks holda har
			// tick'da qayta urinilardi.
			if claimed, cErr := w.recRepo.ClaimRetentionWarning(ctx, rec.ID); cErr != nil || !claimed {
				w.log.Warn(ctx, "retention: egasi topilmagan yozuvni belgilab bo'lmadi",
					logger.String("recording_id", rec.ID))
			}
			continue
		}
		claimed, err := w.recRepo.ClaimRetentionWarning(ctx, rec.ID)
		if err != nil {
			w.log.Warn(ctx, "retention: ogohlantirishni band qilib bo'lmadi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
			continue
		}
		if !claimed {
			continue // boshqa instans allaqachon yuborgan
		}
		days := int(w.cfg.WarnBefore.Hours() / 24)
		if days < 1 {
			days = 1
		}
		body := fmt.Sprintf("«%s» darsining yozuvi %d kundan keyin avtomatik o'chiriladi. Kerak bo'lsa yuklab oling.", l.Title, days)
		lessonID := l.ID
		if _, err := w.notif.Notify(ctx, l.MentorID, entity.NotificationTypeRecordingExpiring,
			"Yozuv o'chirilmoqda", body, &lessonID); err != nil {
			w.log.Warn(ctx, "retention: ogohlantirish yuborilmadi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		}
	}
}

// deleteExpired — muddati o'tgan yozuvlarni MinIO'dan o'chiradi.
//
// TARTIB: avval DB'da ATOMIK `expired` deb belgilanadi, keyin obyekt
// o'chiriladi. Sabab `ClaimExpire` izohida — teskarisi "holat `ready`, lekin
// fayl yo'q" degan buzuq holatni qoldirishi mumkin.
func (w *RetentionWorker) deleteExpired(ctx context.Context, now time.Time) {
	cutoff := now.Add(-w.cfg.Retention)

	// ⭐ Telegram arxivi yoqilganda RO'YXAT ham, YAKUNIY HOLAT ham boshqa:
	//   · ro'yxat — faqat Telegramda tasdiqlanganlar (`ListArchivable`);
	//   · holat — `archived` (yozuv yo'qolmadi, joyi o'zgardi).
	// O'chiq bo'lsa eski xulq to'liq saqlanadi.
	list := w.recRepo.ListExpired
	claim := w.recRepo.ClaimExpire
	final := entity.RecordingStatusExpired
	if w.cfg.TelegramArchive {
		list = w.recRepo.ListArchivable
		claim = w.recRepo.ClaimArchive
		final = entity.RecordingStatusArchived
	}

	recs, err := list(ctx, cutoff, w.cfg.BatchLimit)
	if err != nil {
		w.log.Warn(ctx, "retention: muddati o'tganlar ro'yxatini o'qib bo'lmadi", logger.SafeString("err", err.Error()))
		return
	}
	for _, rec := range recs {
		// `ClaimArchive` ning WHERE sharti `telegram_sent_at IS NOT NULL` ni
		// YANA tekshiradi: ro'yxat olingandan keyingi oynada holat o'zgargan
		// bo'lsa ham fayl o'chmaydi.
		claimed, err := claim(ctx, rec.ID, now)
		if err != nil {
			w.log.Warn(ctx, "retention: o'chirishni band qilib bo'lmadi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
			continue
		}
		if !claimed {
			continue // boshqa instans oldinroq ulgurdi
		}
		if err := w.minio.Delete(ctx, rec.ObjectKey); err != nil {
			// Qator allaqachon yangi holatda — mentor uchun holat to'g'ri.
			// Obyekt esa diskda qolib ketdi: bu OPERATSION nuqson, shuning
			// uchun Error darajasida (monitoring ko'rsin), lekin holatni
			// orqaga qaytarmaymiz — aks holda cheksiz qayta urinish sikliga
			// tushardi.
			w.log.Error(ctx, "retention: MinIO obyektini o'chirib bo'lmadi (diskda qoldi)",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
			continue
		}
		w.log.Info(ctx, "retention: server nusxasi olib tashlandi",
			logger.String("recording_id", rec.ID),
			logger.String("lesson_id", rec.LessonID),
			logger.String("status", final))
		if w.cfg.TelegramArchive {
			w.notifyArchived(ctx, rec)
		}
	}
}

// notifyArchived — «yozuv arxivga ko'chdi» bildirishnomasi.
//
// Kerak, chunki mentor uchun ro'yxatdagi holat kutilmaganda o'zgaradi.
// Xabarsiz bu «yozuvim yo'qoldi» deb tushunilardi — aslida u Telegramda va
// bir bosishda qaytadi.
func (w *RetentionWorker) notifyArchived(ctx context.Context, rec *entity.Recording) {
	l, err := w.lessonRepo.GetByID(ctx, rec.LessonID)
	if err != nil {
		return
	}
	lessonID := l.ID
	body := fmt.Sprintf(
		"«%s» darsining yozuvi serverdan Telegram arxiviga ko'chirildi. Yo'qolgani yo'q — ochganingizda qaytarib olinadi (30-60 soniya).",
		l.Title)
	if _, err := w.notif.Notify(ctx, l.MentorID, entity.NotificationTypeRecordingArchived,
		"Yozuv arxivga ko'chdi", body, &lessonID); err != nil {
		w.log.Warn(ctx, "retention: arxiv bildirishnomasi yuborilmadi",
			logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
	}
}

// warnUnconfirmed — muddati o'tgan, LEKIN Telegramga tushmagan yozuvlar.
//
// Ular O'CHIRILMAYDI (fayl diskda qoladi). Bu ataylab: mahsulot qarori
// bo'yicha ma'lumot yo'qotishdan ko'ra to'lgan disk yaxshiroq. Lekin bu
// holat JIM qolmasligi kerak — mentor bilsin va yuklab olib qo'ysin.
//
// `ClaimRetentionWarning` tufayli har yozuv uchun bir marta (ogohlantirish
// bilan bir xil belgidan foydalanadi — ikki xil ogohlantirish bir yozuv uchun
// baribir keraksiz).
func (w *RetentionWorker) warnUnconfirmed(ctx context.Context, now time.Time) {
	cutoff := now.Add(-w.cfg.Retention)
	recs, err := w.recRepo.ListExpiredUnconfirmed(ctx, cutoff, w.cfg.BatchLimit)
	if err != nil {
		w.log.Warn(ctx, "retention: tasdiqlanmaganlar ro'yxati o'qilmadi", logger.SafeString("err", err.Error()))
		return
	}
	for _, rec := range recs {
		l, err := w.lessonRepo.GetByID(ctx, rec.LessonID)
		if err != nil {
			continue
		}
		claimed, err := w.recRepo.ClaimRetentionWarning(ctx, rec.ID)
		if err != nil || !claimed {
			continue
		}
		lessonID := l.ID
		body := fmt.Sprintf(
			"«%s» darsining yozuvi saqlash muddatidan oshdi, lekin Telegram arxiviga hali tushmagan. "+
				"Shuning uchun u serverda SAQLANIB TURIBDI (o'chirilmaydi). Telegram sozlamalarini tekshiring yoki yozuvni yuklab oling.",
			l.Title)
		if _, err := w.notif.Notify(ctx, l.MentorID, entity.NotificationTypeTelegramFailed,
			"Yozuv arxivlanmagan", body, &lessonID); err != nil {
			w.log.Warn(ctx, "retention: tasdiqlanmagan ogohlantirishi ketmadi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		}
	}
}

// evictCache — Telegramdan tiklangan nusxalarni muddati tugagach o'chiradi.
//
// Kesh bo'lmasa har ochilgan eski dars 1 GB ni diskda abadiy qoldirardi va
// 30 kunlik retention'ning butun ma'nosi yo'qolardi.
func (w *RetentionWorker) evictCache(ctx context.Context, now time.Time) {
	recs, err := w.recRepo.ListCacheExpired(ctx, now, w.cfg.BatchLimit)
	if err != nil {
		w.log.Warn(ctx, "retention: kesh ro'yxati o'qilmadi", logger.SafeString("err", err.Error()))
		return
	}
	for _, rec := range recs {
		// Avval DB (atomik), keyin fayl — `deleteExpired` bilan bir xil
		// sabab: teskarisi "holat ready, fayl yo'q" degan yolg'on qoldirardi.
		claimed, err := w.recRepo.ClaimCacheEvict(ctx, rec.ID)
		if err != nil {
			w.log.Warn(ctx, "retention: kesh o'chirishni band qilib bo'lmadi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
			continue
		}
		if !claimed {
			continue
		}
		if err := w.minio.Delete(ctx, rec.ObjectKey); err != nil {
			w.log.Error(ctx, "retention: kesh obyektini o'chirib bo'lmadi (diskda qoldi)",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
			continue
		}
		w.log.Info(ctx, "retention: kesh nusxasi o'chirildi",
			logger.String("recording_id", rec.ID))
	}
}

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
	for _, rec := range recs {
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
	recs, err := w.recRepo.ListExpired(ctx, cutoff, w.cfg.BatchLimit)
	if err != nil {
		w.log.Warn(ctx, "retention: muddati o'tganlar ro'yxatini o'qib bo'lmadi", logger.SafeString("err", err.Error()))
		return
	}
	for _, rec := range recs {
		claimed, err := w.recRepo.ClaimExpire(ctx, rec.ID, now)
		if err != nil {
			w.log.Warn(ctx, "retention: o'chirishni band qilib bo'lmadi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
			continue
		}
		if !claimed {
			continue // boshqa instans oldinroq ulgurdi
		}
		if err := w.minio.Delete(ctx, rec.ObjectKey); err != nil {
			// Qator allaqachon `expired` — mentor uchun holat to'g'ri. Obyekt
			// esa diskda qolib ketdi: bu OPERATSION nuqson, shuning uchun
			// Error darajasida (monitoring ko'rsin), lekin holatni orqaga
			// qaytarmaymiz — aks holda cheksiz qayta urinish sikliga tushardi.
			w.log.Error(ctx, "retention: MinIO obyektini o'chirib bo'lmadi (diskda qoldi)",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
			continue
		}
		w.log.Info(ctx, "retention: yozuv o'chirildi",
			logger.String("recording_id", rec.ID), logger.String("lesson_id", rec.LessonID))
	}
}

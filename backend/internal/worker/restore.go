package worker

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/recording"
)

// RestoreWorker — `restoring` holatidagi yozuvlarni Telegramdan qaytarib oladi.
//
// # Nega ishchi, HTTP so'rovi ichidagi gorutina emas
//
// Birinchi variant "endpoint 202 qaytaradi va `go uc.RunRestore(...)`"
// bo'lardi. Ikki jiddiy kamchiligi bor:
//
//  1. Jarayon qayta ishga tushsa (deploy, crash) gorutina yo'qoladi va yozuv
//     ABADIY `restoring` da qoladi — mentor uchun bu "tugma ishlamaydi"
//     degani va o'zi tuzata olmaydi.
//  2. Gorutinalar soni cheklanmaydi: bir necha mentor bir vaqtda tiklashni
//     bossa serverga bir necha gigabaytlik parallel yuklab olish tushardi.
//
// Holatning O'ZI navbat bo'lgani uchun bu ishchi ikkalasini ham hal qiladi:
// u `restoring` yozuvlarni skanlaydi (ya'ni "yo'qolgan" ish o'zi qaytadi) va
// ularni BITTALAB bajaradi.
type RestoreWorker struct {
	repo repository.RecordingRepository
	uc   recording.UseCase
	// Interval — skanlash oralig'i. Tiklashning o'zi 30-60 s, shuning uchun
	// 5 s qo'shimcha kechikish sezilmaydi, so'rov esa arzon (status indeksi).
	Interval time.Duration
	// BatchLimit — bitta tsiklda nechta yozuv (parallel emas, ketma-ket).
	BatchLimit uint64
	log        logger.Logger
}

func NewRestoreWorker(repo repository.RecordingRepository, uc recording.UseCase, log logger.Logger) *RestoreWorker {
	return &RestoreWorker{repo: repo, uc: uc, Interval: 5 * time.Second, BatchLimit: 5, log: log}
}

func (w *RestoreWorker) Run(ctx context.Context) {
	if w.Interval <= 0 {
		w.Interval = 5 * time.Second
	}
	w.log.Info(ctx, "restore worker started")
	ticker := time.NewTicker(w.Interval)
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

// RunOnce — bir tsikl. Eksport qilingan: testda ketma-ket chaqirib
// idempotentlikni (ikki marta tiklash urinishi) tekshirish uchun.
func (w *RestoreWorker) RunOnce(ctx context.Context) {
	recs, err := w.repo.ListByStatus(ctx, entity.RecordingStatusRestoring, w.BatchLimit)
	if err != nil {
		w.log.Warn(ctx, "restore: ro'yxatni o'qib bo'lmadi", logger.SafeString("err", err.Error()))
		return
	}
	for _, rec := range recs {
		if ctx.Err() != nil {
			return
		}
		// Xato `RunRestore` ichida qayd etiladi va holat `archived` ga
		// qaytariladi — bu yerda faqat tsiklni davom ettiramiz, chunki bitta
		// yiqilgan tiklash qolganlarini to'xtatmasligi kerak.
		if err := w.uc.RunRestore(ctx, rec.ID); err != nil {
			w.log.Warn(ctx, "restore: tiklash yiqildi",
				logger.String("recording_id", rec.ID), logger.SafeString("err", err.Error()))
		}
	}
}

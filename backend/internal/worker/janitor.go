package worker

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/logger"
)

const janitorBatch = 5000

// JanitorWorker — davriy tozalash: muddati o'tgan token/reset'lar, eski hal qilingan
// kutish-xonasi so'rovlari va o'qilgan bildirishnomalar (aks holda jadvallar cheksiz o'sadi).
type JanitorWorker struct {
	auth repository.AuthRepository
	jan  repository.JanitorRepository
	log  logger.Logger

	WaitingRetention      time.Duration
	NotificationRetention time.Duration
}

func NewJanitorWorker(auth repository.AuthRepository, jan repository.JanitorRepository, log logger.Logger) *JanitorWorker {
	return &JanitorWorker{
		auth: auth, jan: jan, log: log,
		WaitingRetention:      30 * 24 * time.Hour,
		NotificationRetention: 90 * 24 * time.Hour,
	}
}

func (w *JanitorWorker) Run(ctx context.Context, interval time.Duration) {
	w.log.Info(ctx, "janitor worker started", logger.String("interval", interval.String()))
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			w.tick(ctx)
		}
	}
}

func (w *JanitorWorker) tick(ctx context.Context) {
	if err := w.auth.DeleteExpiredTokens(ctx); err != nil {
		w.log.Warn(ctx, "janitor: expired tokens", logger.SafeString("err", err.Error()))
	}
	if err := w.auth.DeleteExpiredPasswordResets(ctx); err != nil {
		w.log.Warn(ctx, "janitor: expired password resets", logger.SafeString("err", err.Error()))
	}
	now := time.Now().UTC()
	w.drain(ctx, "waiting_room_requests", func() (int64, error) {
		return w.jan.DeleteDecidedWaitingRequests(ctx, now.Add(-w.WaitingRetention), janitorBatch)
	})
	w.drain(ctx, "notifications", func() (int64, error) {
		return w.jan.DeleteOldReadNotifications(ctx, now.Add(-w.NotificationRetention), janitorBatch)
	})
}

// drain batch tugaguncha (n < limit) yoki xato/ctx bekor bo'lguncha takrorlaydi.
func (w *JanitorWorker) drain(ctx context.Context, table string, del func() (int64, error)) {
	var total int64
	for ctx.Err() == nil {
		n, err := del()
		if err != nil {
			w.log.Warn(ctx, "janitor: batch failed", logger.String("table", table), logger.SafeString("err", err.Error()))
			break
		}
		total += n
		if n < janitorBatch {
			break
		}
	}
	if total > 0 {
		w.log.Info(ctx, "janitor: cleaned", logger.String("table", table), logger.Int64("rows", total))
	}
}

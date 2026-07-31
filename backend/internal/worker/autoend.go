package worker

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/room"
)

// AutoEndWorker — darsni avtomatik yakunlaydigan fon ishchisi (PRODUCT.md №2).
//
// Ikki qoidani qo'llaydi: 4 soatlik texnik limit va bo'shagan xona grace'i.
// Qoidalarning O'ZI usecase'da (`room.SweepAutoEnd`) — bu yerda faqat tick.
// Sabab CLAUDE.md qoidasi: biznes mantiqi usecase qatlamida bo'lishi kerak,
// ishchi esa `ReminderWorker` kabi shunchaki vaqt manbayi.
type AutoEndWorker struct {
	rooms room.UseCase
	log   logger.Logger
}

func NewAutoEndWorker(rooms room.UseCase, log logger.Logger) *AutoEndWorker {
	return &AutoEndWorker{rooms: rooms, log: log}
}

// Run har `interval` da bir marta jonli darslarni ko'rib chiqadi.
// ctx bekor qilinganda toza chiqadi (graceful shutdown).
func (w *AutoEndWorker) Run(ctx context.Context, interval, maxDuration, emptyGrace time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.log.Info(ctx, "auto-end worker started",
		logger.String("interval", interval.String()),
		logger.String("max_duration", maxDuration.String()),
		logger.String("empty_grace", emptyGrace.String()))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if n := w.rooms.SweepAutoEnd(ctx, maxDuration, emptyGrace); n > 0 {
				w.log.Info(ctx, "auto-end: darslar yakunlandi", logger.Int("count", n))
			}
		}
	}
}

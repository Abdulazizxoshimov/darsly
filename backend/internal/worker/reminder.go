// Package worker — fon (background) vazifalari.
package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/notification"
)

// ReminderWorker rejalashtirilgan darslar uchun mentorga eslatma yuboradi.
type ReminderWorker struct {
	lessonRepo repository.LessonRepository
	notif      notification.UseCase
	log        logger.Logger
}

func NewReminderWorker(lessonRepo repository.LessonRepository, notif notification.UseCase, log logger.Logger) *ReminderWorker {
	return &ReminderWorker{lessonRepo: lessonRepo, notif: notif, log: log}
}

// Run har `interval` da yaqinlashayotgan (keyingi `lead` ichida boshlanadigan) darslarni tekshiradi.
func (w *ReminderWorker) Run(ctx context.Context, interval, lead time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	w.log.Info(ctx, "reminder worker started", logger.String("interval", interval.String()), logger.String("lead", lead.String()))
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.tick(ctx, lead)
		}
	}
}

func (w *ReminderWorker) tick(ctx context.Context, lead time.Duration) {
	now := time.Now().UTC()
	lessons, err := w.lessonRepo.ListUpcomingUnreminded(ctx, now, now.Add(lead))
	if err != nil {
		w.log.Warn(ctx, "reminder tick: query failed", logger.SafeString("err", err.Error()))
		return
	}
	for _, l := range lessons {
		// Avval atomik claim — faqat biz band qilsak eslatma yuboramiz (dublikatsiz, distributed-safe).
		claimed, err := w.lessonRepo.ClaimReminder(ctx, l.ID)
		if err != nil {
			w.log.Warn(ctx, "reminder tick: claim failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
			continue
		}
		if !claimed {
			continue // boshqa instans allaqachon yuborgan
		}
		body := fmt.Sprintf("«%s» darsi tez orada boshlanadi.", l.Title)
		lessonID := l.ID
		if _, err := w.notif.Notify(ctx, l.MentorID, entity.NotificationTypeLessonReminder, "Dars eslatmasi", body, &lessonID); err != nil {
			w.log.Warn(ctx, "reminder tick: notify failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		}
	}
}

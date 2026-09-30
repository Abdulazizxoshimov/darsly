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
	// retryBackoff — Notify qayta urinishlari orasidagi boshlang'ich kutish (0 = default; testlar uchun).
	retryBackoff time.Duration
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
		if err := w.notifyWithRetry(ctx, l.MentorID, body, &lessonID); err != nil {
			// Qayta urinishlar ham yiqildi — claim'ni QAYTARAMIZ, shunda keyingi
			// tick qayta uriladi (audit realtime #11). Notify muvaffaqiyatsiz =
			// bildirishnoma yaratilmagan, ya'ni qayta yuborish dublikat yasamaydi.
			w.log.Error(ctx, "reminder tick: notify failed after retries — unclaiming",
				logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
			if uerr := w.lessonRepo.UnclaimReminder(ctx, l.ID); uerr != nil {
				w.log.Warn(ctx, "reminder tick: unclaim failed (eslatma yo'qolishi mumkin)",
					logger.String("lesson_id", l.ID), logger.SafeString("err", uerr.Error()))
			}
		}
	}
}

// notifyAttempts / retryBackoff — claim-before-send tufayli bir martalik DB/Redis
// uzilishi eslatmani abadiy yo'qotmasligi uchun qisqa qayta urinish.
const notifyAttempts = 3

// notifyWithRetry — Notify'ni eksponensial kutish bilan qayta urinadi (ctx bekor bo'lsa to'xtaydi).
func (w *ReminderWorker) notifyWithRetry(ctx context.Context, mentorID, body string, lessonID *string) error {
	backoff := w.retryBackoff
	if backoff == 0 {
		backoff = 300 * time.Millisecond
	}
	var err error
	for attempt := 1; attempt <= notifyAttempts; attempt++ {
		if _, err = w.notif.Notify(ctx, mentorID, entity.NotificationTypeLessonReminder, "Dars eslatmasi", body, lessonID); err == nil {
			return nil
		}
		w.log.Warn(ctx, "reminder tick: notify failed", logger.Int("attempt", attempt), logger.SafeString("err", err.Error()))
		if attempt == notifyAttempts {
			break
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(backoff):
			backoff *= 2
		}
	}
	return err
}

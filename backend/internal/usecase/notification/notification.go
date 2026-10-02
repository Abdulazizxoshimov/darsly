package notification

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// Notifier — real-time WS push uchun minimal port (DIP; consumer-side, audit R7).
//
// Avval bu yerda konkret `*ws.Hub` turardi va har test HAQIQIY hub qurishga
// majbur edi. Endi interfeys: `*ws.Hub` uni strukturaviy qondiradi, testlar esa
// yengil fake bera oladi. WS fan-out (Redis) tufayli worker-only jarayonda ham
// bu Send ishlaydi — xabar Redis orqali API jarayoniga o'tadi.
type Notifier interface {
	Send(userID string, msg ws.Message)
}

type useCase struct {
	repo repository.NotificationRepository
	hub  Notifier
	log  logger.Logger
}

func New(repo repository.NotificationRepository, hub Notifier, log logger.Logger) UseCase {
	return &useCase{repo: repo, hub: hub, log: log}
}

func (uc *useCase) Notify(ctx context.Context, userID, ntype, title, body string, lessonID *string) (*entity.Notification, error) {
	n := &entity.Notification{
		ID:        uuid.NewString(),
		UserID:    userID,
		Type:      ntype,
		Title:     title,
		Body:      body,
		LessonID:  lessonID,
		CreatedAt: time.Now().UTC(),
	}
	if err := uc.repo.Create(ctx, n); err != nil {
		uc.log.Error(ctx, "notification.Notify: db error", logger.String("user_id", userID), logger.SafeString("err", err.Error()))
		return nil, err
	}

	// Real-time push (foydalanuvchi ulangan bo'lsa).
	uc.hub.Send(userID, ws.NewNotificationMsg(userID, n))
	metrics.NotificationsSent.Inc()

	uc.log.Info(ctx, "notification sent", logger.String("user_id", userID), logger.String("type", ntype))
	return n, nil
}

func (uc *useCase) List(ctx context.Context, userID string, filter *entity.NotificationFilter) ([]*entity.Notification, int, error) {
	return uc.repo.ListByUser(ctx, userID, filter)
}

func (uc *useCase) UnreadCount(ctx context.Context, userID string) (int, error) {
	return uc.repo.UnreadCount(ctx, userID)
}

func (uc *useCase) MarkRead(ctx context.Context, userID, id string) error {
	// Yaroqsiz UUID Postgres'ga yetmasin (22P02 → 500); NotFound qaytadi.
	if err := shared.ValidateID(id, "notification"); err != nil {
		return err
	}
	return uc.repo.MarkRead(ctx, id, userID)
}

func (uc *useCase) MarkAllRead(ctx context.Context, userID string) error {
	return uc.repo.MarkAllRead(ctx, userID)
}

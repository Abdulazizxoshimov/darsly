package notification

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UseCase interface {
	// Notify — bildirishnoma yaratadi va WS orqali foydalanuvchiga yuboradi.
	Notify(ctx context.Context, userID, ntype, title, body string, lessonID *string) (*entity.Notification, error)
	List(ctx context.Context, userID string, filter *entity.NotificationFilter) ([]*entity.Notification, int, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, userID, id string) error
	MarkAllRead(ctx context.Context, userID string) error
}

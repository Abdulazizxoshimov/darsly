package repository

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *entity.Notification) error
	ListByUser(ctx context.Context, userID string, filter *entity.NotificationFilter) ([]*entity.Notification, int, error)
	UnreadCount(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
}

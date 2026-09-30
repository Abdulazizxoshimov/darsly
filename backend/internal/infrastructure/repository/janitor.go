package repository

import (
	"context"
	"time"
)

// JanitorRepository — cheksiz o'suvchi jadvallarni BATCH'lab tozalash.
// Har metod bitta batch'da o'chirilgan qatorlar sonini qaytaradi (limit'gacha).
type JanitorRepository interface {
	DeleteDecidedWaitingRequests(ctx context.Context, olderThan time.Time, limit int) (int64, error)
	DeleteOldReadNotifications(ctx context.Context, olderThan time.Time, limit int) (int64, error)
}

package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/infrastructure/repository"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type janitorRepo struct{ db *pgxpool.Pool }

func NewJanitorRepo(p *pg.Postgres) repository.JanitorRepository {
	return &janitorRepo{db: p.DB}
}

// DeleteDecidedWaitingRequests — qarori chiqqan (pending emas) eski so'rovlar.
func (r *janitorRepo) DeleteDecidedWaitingRequests(ctx context.Context, olderThan time.Time, limit int) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM waiting_room_requests WHERE id IN (
			SELECT id FROM waiting_room_requests
			WHERE status <> 'pending' AND created_at < $1
			LIMIT $2)`, olderThan, limit)
	if err != nil {
		return 0, fmt.Errorf("janitorRepo.DeleteDecidedWaitingRequests: %w", err)
	}
	return tag.RowsAffected(), nil
}

// DeleteOldReadNotifications — o'qilgan va eski bildirishnomalar.
func (r *janitorRepo) DeleteOldReadNotifications(ctx context.Context, olderThan time.Time, limit int) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM notifications WHERE id IN (
			SELECT id FROM notifications
			WHERE read_at IS NOT NULL AND created_at < $1
			LIMIT $2)`, olderThan, limit)
	if err != nil {
		return 0, fmt.Errorf("janitorRepo.DeleteOldReadNotifications: %w", err)
	}
	return tag.RowsAffected(), nil
}

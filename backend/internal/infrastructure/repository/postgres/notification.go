package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type notificationRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewNotificationRepo(p *pg.Postgres) repository.NotificationRepository {
	return &notificationRepo{db: p.DB, builder: p.Builder}
}

const notificationCols = "id, user_id, type, title, body, lesson_id, read_at, created_at"

func scanNotification(row pgx.Row) (*entity.Notification, error) {
	n := &entity.Notification{}
	err := row.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.LessonID, &n.ReadAt, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (r *notificationRepo) Create(ctx context.Context, n *entity.Notification) error {
	sql, args, err := r.builder.
		Insert("notifications").
		Columns("id", "user_id", "type", "title", "body", "lesson_id", "created_at").
		Values(n.ID, n.UserID, n.Type, n.Title, n.Body, n.LessonID, n.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("notificationRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("notificationRepo.Create: %w", err)
	}
	return nil
}

func (r *notificationRepo) ListByUser(ctx context.Context, userID string, filter *entity.NotificationFilter) ([]*entity.Notification, int, error) {
	where := sq.And{sq.Eq{"user_id": userID}}
	if filter.UnreadOnly {
		where = append(where, sq.Eq{"read_at": nil})
	}

	var total int
	cntSQL, cntArgs, _ := r.builder.Select("COUNT(*)").From("notifications").Where(where).ToSql()
	if err := r.db.QueryRow(ctx, cntSQL, cntArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("notificationRepo.ListByUser count: %w", err)
	}

	dataSQL, dataArgs, _ := r.builder.
		Select(notificationCols).From("notifications").Where(where).
		OrderBy("created_at DESC").
		Limit(uint64(filter.GetLimit())).Offset(uint64(filter.Offset())).ToSql()

	rows, err := r.db.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("notificationRepo.ListByUser: %w", err)
	}
	defer rows.Close()

	var out []*entity.Notification
	for rows.Next() {
		n, err := scanNotification(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, n)
	}
	return out, total, rows.Err()
}

func (r *notificationRepo) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("notificationRepo.UnreadCount: %w", err)
	}
	return count, nil
}

func (r *notificationRepo) MarkRead(ctx context.Context, id, userID string) error {
	sql, args, _ := r.builder.
		Update("notifications").Set("read_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"user_id": userID}, sq.Eq{"read_at": nil}}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

func (r *notificationRepo) MarkAllRead(ctx context.Context, userID string) error {
	sql, args, _ := r.builder.
		Update("notifications").Set("read_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"user_id": userID}, sq.Eq{"read_at": nil}}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

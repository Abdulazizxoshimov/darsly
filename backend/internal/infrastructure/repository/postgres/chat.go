package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type chatRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewChatRepo(p *pg.Postgres) repository.ChatRepository {
	return &chatRepo{db: p.DB, builder: p.Builder}
}

func (r *chatRepo) Create(ctx context.Context, m *entity.ChatMessage) error {
	sql, args, err := r.builder.
		Insert("chat_messages").
		Columns("id", "lesson_id", "sender_identity", "sender_name", "body", "created_at").
		Values(m.ID, m.LessonID, m.SenderIdentity, m.SenderName, m.Body, m.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("chatRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("chatRepo.Create: %w", err)
	}
	return nil
}

func (r *chatRepo) ListByLesson(ctx context.Context, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := r.builder.
		Select("id", "lesson_id", "sender_identity", "sender_name", "body", "created_at").
		From("chat_messages").Where(sq.Eq{"lesson_id": lessonID})
	if before != nil {
		q = q.Where(sq.Lt{"created_at": *before}) // kursor: faqat undan eski xabarlar
	}
	// Eng yangidan eskiga — uzun darsda host oxirgi xabarlarni ko'radi (offset yo'q, arzon).
	sql, args, _ := q.OrderBy("created_at DESC").Limit(uint64(limit)).ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("chatRepo.ListByLesson: %w", err)
	}
	defer rows.Close()

	var out []*entity.ChatMessage
	for rows.Next() {
		m := &entity.ChatMessage{}
		if err := rows.Scan(&m.ID, &m.LessonID, &m.SenderIdentity, &m.SenderName, &m.Body, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

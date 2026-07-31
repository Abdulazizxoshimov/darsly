package postgres

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type blocklistRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewBlocklistRepo(p *pg.Postgres) repository.BlocklistRepository {
	return &blocklistRepo{db: p.DB, builder: p.Builder}
}

func (r *blocklistRepo) Add(ctx context.Context, e *entity.BlocklistEntry) error {
	sql, args, err := r.builder.
		Insert("mentor_blocklist").
		Columns("id", "mentor_id", "identity", "display_name").
		Values(e.ID, e.MentorID, e.Identity, strings.TrimSpace(e.DisplayName)).
		// Bir mentor uchun bir ism bir marta — takror kick xato emas, no-op.
		Suffix("ON CONFLICT DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("blocklistRepo.Add: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("blocklistRepo.Add: %w", err)
	}
	return nil
}

func (r *blocklistRepo) IsBlocked(ctx context.Context, mentorID, displayName string) (bool, error) {
	name := strings.TrimSpace(displayName)
	if name == "" {
		return false, nil // bo'sh ism hech kimni bloklamaydi
	}
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(
		    SELECT 1 FROM mentor_blocklist
		    WHERE mentor_id = $1 AND lower(display_name) = lower($2)
		 )`, mentorID, name,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("blocklistRepo.IsBlocked: %w", err)
	}
	return exists, nil
}

func (r *blocklistRepo) ListByMentor(ctx context.Context, mentorID string) ([]*entity.BlocklistEntry, error) {
	sql, args, err := r.builder.
		Select("id", "mentor_id", "identity", "display_name", "created_at").
		From("mentor_blocklist").
		Where(sq.Eq{"mentor_id": mentorID}).
		OrderBy("created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("blocklistRepo.ListByMentor: %w", err)
	}
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("blocklistRepo.ListByMentor: %w", err)
	}
	defer rows.Close()

	var out []*entity.BlocklistEntry
	for rows.Next() {
		e := &entity.BlocklistEntry{}
		if err := rows.Scan(&e.ID, &e.MentorID, &e.Identity, &e.DisplayName, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("blocklistRepo.ListByMentor scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *blocklistRepo) Delete(ctx context.Context, mentorID, id string) error {
	// mentor_id ham shartda — begona mentor yozuvini o'chirib bo'lmaydi (IDOR emas).
	sql, args, err := r.builder.
		Delete("mentor_blocklist").
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"mentor_id": mentorID}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("blocklistRepo.Delete: %w", err)
	}
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("blocklistRepo.Delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("blocklist entry")
	}
	return nil
}

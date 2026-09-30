package postgres

import (
	"context"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/infrastructure/repository"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type banRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewBanRepo(p *pg.Postgres) repository.BanRepository {
	return &banRepo{db: p.DB, builder: p.Builder}
}

func (r *banRepo) AddBan(ctx context.Context, lessonID, identity, displayName string) error {
	sql, args, err := r.builder.
		Insert("lesson_bans").
		Columns("lesson_id", "identity", "display_name").
		Values(lessonID, identity, strings.TrimSpace(displayName)).
		// Takror kick — no-op (PK: lesson_id + identity).
		Suffix("ON CONFLICT (lesson_id, identity) DO NOTHING").
		ToSql()
	if err != nil {
		return fmt.Errorf("banRepo.AddBan: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("banRepo.AddBan: %w", err)
	}
	return nil
}

func (r *banRepo) IsBanned(ctx context.Context, lessonID, identity string) (bool, error) {
	if identity == "" {
		return false, nil
	}
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM lesson_bans WHERE lesson_id = $1 AND identity = $2)`,
		lessonID, identity,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("banRepo.IsBanned: %w", err)
	}
	return exists, nil
}

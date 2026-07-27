package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type pollRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewPollRepo(p *pg.Postgres) repository.PollRepository {
	return &pollRepo{db: p.DB, builder: p.Builder}
}

func (r *pollRepo) Create(ctx context.Context, p *entity.Poll) error {
	sql, args, err := r.builder.
		Insert("polls").
		Columns("id", "lesson_id", "question", "options", "is_active", "created_at").
		Values(p.ID, p.LessonID, p.Question, p.Options, p.IsActive, p.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("pollRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("pollRepo.Create: %w", err)
	}
	return nil
}

func (r *pollRepo) GetByID(ctx context.Context, id string) (*entity.Poll, error) {
	sql, args, _ := r.builder.
		Select("id", "lesson_id", "question", "options", "is_active", "created_at", "closed_at").
		From("polls").Where(sq.Eq{"id": id}).ToSql()
	p := &entity.Poll{}
	err := r.db.QueryRow(ctx, sql, args...).Scan(&p.ID, &p.LessonID, &p.Question, &p.Options, &p.IsActive, &p.CreatedAt, &p.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("poll")
	}
	return p, err
}

func (r *pollRepo) ListByLesson(ctx context.Context, lessonID string) ([]*entity.Poll, error) {
	sql, args, _ := r.builder.
		Select("id", "lesson_id", "question", "options", "is_active", "created_at", "closed_at").
		From("polls").Where(sq.Eq{"lesson_id": lessonID}).OrderBy("created_at DESC").Limit(200).ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("pollRepo.ListByLesson: %w", err)
	}
	defer rows.Close()

	var out []*entity.Poll
	for rows.Next() {
		p := &entity.Poll{}
		if err := rows.Scan(&p.ID, &p.LessonID, &p.Question, &p.Options, &p.IsActive, &p.CreatedAt, &p.ClosedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *pollRepo) Close(ctx context.Context, id string) error {
	sql, args, _ := r.builder.Update("polls").
		Set("is_active", false).Set("closed_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

func (r *pollRepo) Vote(ctx context.Context, pollID, voterIdentity string, optionIndex int) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO poll_votes (poll_id, voter_identity, option_index) VALUES ($1,$2,$3)
		 ON CONFLICT (poll_id, voter_identity) DO UPDATE SET option_index = EXCLUDED.option_index, created_at = NOW()`,
		pollID, voterIdentity, optionIndex)
	if err != nil {
		return fmt.Errorf("pollRepo.Vote: %w", err)
	}
	return nil
}

func (r *pollRepo) Counts(ctx context.Context, pollID string, numOptions int) ([]int, error) {
	counts := make([]int, numOptions)
	rows, err := r.db.Query(ctx,
		`SELECT option_index, COUNT(*) FROM poll_votes WHERE poll_id = $1 GROUP BY option_index`, pollID)
	if err != nil {
		return nil, fmt.Errorf("pollRepo.Counts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var idx, n int
		if err := rows.Scan(&idx, &n); err != nil {
			return nil, err
		}
		if idx >= 0 && idx < numOptions {
			counts[idx] = n
		}
	}
	return counts, rows.Err()
}

package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

// pollColumns — o'qish ustunlari (Select/Scan/RETURNING bitta manbadan).
var pollColumns = []string{
	"id", "lesson_id", "question", "options", "is_active", "created_at", "closed_at",
	"results_visibility", "results_published_at",
}

func scanPoll(row pgx.Row) (*entity.Poll, error) {
	p := &entity.Poll{}
	if err := row.Scan(&p.ID, &p.LessonID, &p.Question, &p.Options, &p.IsActive,
		&p.CreatedAt, &p.ClosedAt, &p.ResultsVisibility, &p.ResultsPublishedAt); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *pollRepo) Create(ctx context.Context, p *entity.Poll) error {
	sql, args, err := r.builder.
		Insert("polls").
		Columns("id", "lesson_id", "question", "options", "is_active", "created_at", "results_visibility").
		Values(p.ID, p.LessonID, p.Question, p.Options, p.IsActive, p.CreatedAt, p.ResultsVisibility).
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
	sql, args, _ := r.builder.Select(pollColumns...).From("polls").Where(sq.Eq{"id": id}).ToSql()
	p, err := scanPoll(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("poll")
	}
	return p, err
}

func (r *pollRepo) ListByLesson(ctx context.Context, lessonID string) ([]*entity.Poll, error) {
	sql, args, _ := r.builder.Select(pollColumns...).
		From("polls").Where(sq.Eq{"lesson_id": lessonID}).OrderBy("created_at DESC").Limit(200).ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("pollRepo.ListByLesson: %w", err)
	}
	defer rows.Close()

	var out []*entity.Poll
	for rows.Next() {
		p, err := scanPoll(rows)
		if err != nil {
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

func (r *pollRepo) Publish(ctx context.Context, pollID string) (*entity.Poll, error) {
	sql, args, err := r.builder.Update("polls").
		// COALESCE — idempotentlik (interfeys izohiga qara).
		Set("results_published_at", sq.Expr("COALESCE(results_published_at, NOW())")).
		Where(sq.Eq{"id": pollID}).
		Suffix("RETURNING " + strings.Join(pollColumns, ", ")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("pollRepo.Publish: %w", err)
	}
	p, err := scanPoll(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("poll")
	}
	if err != nil {
		return nil, fmt.Errorf("pollRepo.Publish: %w", err)
	}
	return p, nil
}

// Vote — upsert SHARTLI: faqat poll hali faol bo'lsa yoziladi (usecase'dagi
// is_active o'qish bilan yozish orasidagi Close poygasini yopadi). Poll yopilgan
// (yoki yo'q) bo'lsa RowsAffected==0 → "poll is closed".
func (r *pollRepo) Vote(ctx context.Context, pollID, voterIdentity string, optionIndex int) error {
	tag, err := r.db.Exec(ctx,
		`INSERT INTO poll_votes (poll_id, voter_identity, option_index)
		 SELECT $1::uuid, $2, $3
		 WHERE EXISTS (SELECT 1 FROM polls WHERE id = $1::uuid AND is_active)
		 ON CONFLICT (poll_id, voter_identity) DO UPDATE SET option_index = EXCLUDED.option_index, created_at = NOW()`,
		pollID, voterIdentity, optionIndex)
	if err != nil {
		return fmt.Errorf("pollRepo.Vote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.BadRequest("poll is closed")
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

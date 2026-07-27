package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type recordingRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewRecordingRepo(p *pg.Postgres) repository.RecordingRepository {
	return &recordingRepo{db: p.DB, builder: p.Builder}
}

const recordingCols = "id, lesson_id, egress_id, object_key, status, duration_sec, size_bytes, started_at, ended_at, created_at"

func scanRecording(row pgx.Row) (*entity.Recording, error) {
	r := &entity.Recording{}
	err := row.Scan(&r.ID, &r.LessonID, &r.EgressID, &r.ObjectKey, &r.Status,
		&r.DurationSec, &r.SizeBytes, &r.StartedAt, &r.EndedAt, &r.CreatedAt)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *recordingRepo) Create(ctx context.Context, rec *entity.Recording) error {
	sql, args, err := r.builder.
		Insert("recordings").
		Columns("id", "lesson_id", "egress_id", "object_key", "status", "duration_sec", "size_bytes", "started_at", "created_at").
		Values(rec.ID, rec.LessonID, rec.EgressID, rec.ObjectKey, rec.Status, rec.DurationSec, rec.SizeBytes, rec.StartedAt, rec.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("recordingRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("recordingRepo.Create: %w", err)
	}
	return nil
}

func (r *recordingRepo) GetByID(ctx context.Context, id string) (*entity.Recording, error) {
	sql, args, _ := r.builder.Select(recordingCols).From("recordings").Where(sq.Eq{"id": id}).ToSql()
	rec, err := scanRecording(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("recording")
	}
	return rec, err
}

func (r *recordingRepo) GetByEgressID(ctx context.Context, egressID string) (*entity.Recording, error) {
	sql, args, _ := r.builder.Select(recordingCols).From("recordings").Where(sq.Eq{"egress_id": egressID}).ToSql()
	rec, err := scanRecording(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("recording")
	}
	return rec, err
}

func (r *recordingRepo) ListByLesson(ctx context.Context, lessonID string) ([]*entity.Recording, error) {
	sql, args, _ := r.builder.Select(recordingCols).From("recordings").
		Where(sq.Eq{"lesson_id": lessonID}).OrderBy("created_at DESC").Limit(200).ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListByLesson: %w", err)
	}
	defer rows.Close()

	var out []*entity.Recording
	for rows.Next() {
		rec, err := scanRecording(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

func (r *recordingRepo) UpdateStatus(ctx context.Context, id, status string) error {
	sql, args, _ := r.builder.Update("recordings").Set("status", status).Where(sq.Eq{"id": id}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

func (r *recordingRepo) MarkReady(ctx context.Context, egressID, objectKey string, durationSec int, sizeBytes int64, endedAt time.Time) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusReady).
		Set("object_key", objectKey).
		Set("duration_sec", durationSec).
		Set("size_bytes", sizeBytes).
		Set("ended_at", endedAt).
		Where(sq.Eq{"egress_id": egressID}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

func (r *recordingRepo) MarkFailed(ctx context.Context, egressID string, endedAt time.Time) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusFailed).
		Set("ended_at", endedAt).
		Where(sq.Eq{"egress_id": egressID}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

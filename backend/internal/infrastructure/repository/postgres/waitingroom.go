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

type waitingRoomRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewWaitingRoomRepo(p *pg.Postgres) repository.WaitingRoomRepository {
	return &waitingRoomRepo{db: p.DB, builder: p.Builder}
}

const waitingCols = "id, lesson_id, requester_name, guest_identity, status, created_at, decided_at"

func scanWaiting(row pgx.Row) (*entity.WaitingRoomRequest, error) {
	w := &entity.WaitingRoomRequest{}
	err := row.Scan(&w.ID, &w.LessonID, &w.RequesterName, &w.GuestIdentity, &w.Status, &w.CreatedAt, &w.DecidedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (r *waitingRoomRepo) Create(ctx context.Context, w *entity.WaitingRoomRequest) error {
	sql, args, err := r.builder.
		Insert("waiting_room_requests").
		Columns("id", "lesson_id", "requester_name", "guest_identity", "status", "created_at").
		Values(w.ID, w.LessonID, w.RequesterName, w.GuestIdentity, w.Status, w.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("waitingRoomRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("waitingRoomRepo.Create: %w", err)
	}
	return nil
}

func (r *waitingRoomRepo) GetByID(ctx context.Context, id string) (*entity.WaitingRoomRequest, error) {
	sql, args, err := r.builder.
		Select(waitingCols).From("waiting_room_requests").
		Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("waitingRoomRepo.GetByID: %w", err)
	}
	w, err := scanWaiting(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("waiting room request")
	}
	return w, err
}

func (r *waitingRoomRepo) ListPending(ctx context.Context, lessonID string) ([]*entity.WaitingRoomRequest, error) {
	sql, args, err := r.builder.
		Select(waitingCols).From("waiting_room_requests").
		Where(sq.And{sq.Eq{"lesson_id": lessonID}, sq.Eq{"status": entity.WaitingStatusPending}}).
		OrderBy("created_at ASC").ToSql()
	if err != nil {
		return nil, fmt.Errorf("waitingRoomRepo.ListPending: %w", err)
	}
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("waitingRoomRepo.ListPending: %w", err)
	}
	defer rows.Close()

	var out []*entity.WaitingRoomRequest
	for rows.Next() {
		w, err := scanWaiting(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *waitingRoomRepo) CountPendingByLesson(ctx context.Context, lessonID string) (int, error) {
	sql, args, err := r.builder.
		Select("COUNT(*)").From("waiting_room_requests").
		Where(sq.And{sq.Eq{"lesson_id": lessonID}, sq.Eq{"status": entity.WaitingStatusPending}}).ToSql()
	if err != nil {
		return 0, fmt.Errorf("waitingRoomRepo.CountPendingByLesson: %w", err)
	}
	var n int
	if err := r.db.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("waitingRoomRepo.CountPendingByLesson: %w", err)
	}
	return n, nil
}

// maxPendingSnapshot — mentor WS reconnect'ida bir martada yetkaziladigan pending'lar chegarasi.
const maxPendingSnapshot = 200

func (r *waitingRoomRepo) ListPendingByMentor(ctx context.Context, mentorID string) ([]*entity.WaitingRoomRequest, error) {
	sql, args, err := r.builder.
		Select("w.id, w.lesson_id, w.requester_name, w.guest_identity, w.status, w.created_at, w.decided_at").
		From("waiting_room_requests w").
		Join("lessons l ON l.id = w.lesson_id").
		Where(sq.And{
			sq.Eq{"l.mentor_id": mentorID},
			sq.Eq{"w.status": entity.WaitingStatusPending},
			sq.Eq{"l.deleted_at": nil},
			// Faqat hozirgi/kelgusi darslar: tugagan (ended) darslardan "ghost" push bo'lmasin.
			sq.Eq{"l.status": []string{entity.LessonStatusScheduled, entity.LessonStatusLive}},
		}).
		OrderBy("w.created_at ASC").
		Limit(maxPendingSnapshot).ToSql()
	if err != nil {
		return nil, fmt.Errorf("waitingRoomRepo.ListPendingByMentor: %w", err)
	}
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("waitingRoomRepo.ListPendingByMentor: %w", err)
	}
	defer rows.Close()

	var out []*entity.WaitingRoomRequest
	for rows.Next() {
		w, err := scanWaiting(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *waitingRoomRepo) TransitionFromPending(ctx context.Context, id, newStatus string, decidedAt time.Time) (bool, error) {
	sql, args, err := r.builder.
		Update("waiting_room_requests").
		Set("status", newStatus).
		Set("decided_at", decidedAt).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"status": entity.WaitingStatusPending}}).ToSql()
	if err != nil {
		return false, fmt.Errorf("waitingRoomRepo.TransitionFromPending: %w", err)
	}
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("waitingRoomRepo.TransitionFromPending: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

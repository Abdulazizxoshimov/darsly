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

var allowedLessonSortCols = map[string]bool{
	"created_at": true, "updated_at": true, "scheduled_at": true, "title": true,
}

type lessonRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewLessonRepo(p *pg.Postgres) repository.LessonRepository {
	return &lessonRepo{db: p.DB, builder: p.Builder}
}

const lessonCols = "id, mentor_id, title, description, scheduled_at, duration_min, recurrence_rule, " +
	"join_slug, passcode_hash, is_locked, is_recording_enabled, is_waiting_room_enabled, " +
	"mute_on_entry, allow_self_unmute, " +
	"status, started_at, ended_at, created_at, updated_at, deleted_at"

func scanLesson(row pgx.Row) (*entity.Lesson, error) {
	l := &entity.Lesson{}
	err := row.Scan(
		&l.ID, &l.MentorID, &l.Title, &l.Description, &l.ScheduledAt, &l.DurationMin, &l.RecurrenceRule,
		&l.JoinSlug, &l.PasscodeHash, &l.IsLocked, &l.IsRecordingEnabled, &l.IsWaitingRoomEnabled,
		&l.MuteOnEntry, &l.AllowSelfUnmute,
		&l.Status, &l.StartedAt, &l.EndedAt, &l.CreatedAt, &l.UpdatedAt, &l.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	l.HasPasscode = l.PasscodeHash != nil
	return l, nil
}

func (r *lessonRepo) Create(ctx context.Context, l *entity.Lesson) error {
	sql, args, err := r.builder.
		Insert("lessons").
		Columns("id", "mentor_id", "title", "description", "scheduled_at", "duration_min", "recurrence_rule",
			"join_slug", "passcode_hash", "is_locked", "is_recording_enabled", "is_waiting_room_enabled",
			"mute_on_entry", "allow_self_unmute",
			"status", "created_at", "updated_at").
		Values(l.ID, l.MentorID, l.Title, l.Description, l.ScheduledAt, l.DurationMin, l.RecurrenceRule,
			l.JoinSlug, l.PasscodeHash, l.IsLocked, l.IsRecordingEnabled, l.IsWaitingRoomEnabled,
			l.MuteOnEntry, l.AllowSelfUnmute,
			l.Status, l.CreatedAt, l.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("lessonRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("join_slug already exists")
		}
		return fmt.Errorf("lessonRepo.Create: %w", err)
	}
	return nil
}

func (r *lessonRepo) GetByID(ctx context.Context, id string) (*entity.Lesson, error) {
	sql, args, err := r.builder.
		Select(lessonCols).From("lessons").
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("lessonRepo.GetByID: %w", err)
	}
	l, err := scanLesson(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("lesson")
	}
	return l, err
}

func (r *lessonRepo) GetBySlug(ctx context.Context, slug string) (*entity.Lesson, error) {
	sql, args, err := r.builder.
		Select(lessonCols).From("lessons").
		Where(sq.And{sq.Eq{"join_slug": slug}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("lessonRepo.GetBySlug: %w", err)
	}
	l, err := scanLesson(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("lesson")
	}
	return l, err
}

func (r *lessonRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM lessons WHERE join_slug = $1)`, slug,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("lessonRepo.SlugExists: %w", err)
	}
	return exists, nil
}

func (r *lessonRepo) ListByMentor(ctx context.Context, mentorID string, filter *entity.LessonFilter) ([]*entity.Lesson, int, error) {
	where := sq.And{sq.Eq{"mentor_id": mentorID}, sq.Eq{"deleted_at": nil}}
	if filter.Search != "" {
		where = append(where, sq.ILike{"title": SearchPattern(filter.Search)})
	}
	if filter.Status != "" {
		where = append(where, sq.Eq{"status": filter.Status})
	}
	if filter.CreatedFrom != nil {
		where = append(where, sq.GtOrEq{"created_at": *filter.CreatedFrom})
	}
	if filter.CreatedTo != nil {
		where = append(where, sq.LtOrEq{"created_at": *filter.CreatedTo})
	}

	var total int
	cntSQL, cntArgs, _ := r.builder.Select("COUNT(*)").From("lessons").Where(where).ToSql()
	if err := r.db.QueryRow(ctx, cntSQL, cntArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("lessonRepo.ListByMentor count: %w", err)
	}

	sortBy := "created_at"
	if allowedLessonSortCols[filter.SortBy] {
		sortBy = filter.SortBy
	}
	order := "DESC"
	if filter.GetSortOrder() == entity.SortAsc {
		order = "ASC"
	}

	dataSQL, dataArgs, _ := r.builder.
		Select(lessonCols).From("lessons").Where(where).
		OrderBy(sortBy + " " + order).
		Limit(uint64(filter.GetLimit())).Offset(uint64(filter.Offset())).
		ToSql()

	rows, err := r.db.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("lessonRepo.ListByMentor query: %w", err)
	}
	defer rows.Close()

	var lessons []*entity.Lesson
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, 0, err
		}
		lessons = append(lessons, l)
	}
	return lessons, total, rows.Err()
}

func (r *lessonRepo) Update(ctx context.Context, l *entity.Lesson) error {
	sql, args, err := r.builder.
		Update("lessons").
		Set("title", l.Title).
		Set("description", l.Description).
		Set("scheduled_at", l.ScheduledAt).
		Set("duration_min", l.DurationMin).
		Set("recurrence_rule", l.RecurrenceRule).
		Set("passcode_hash", l.PasscodeHash).
		Set("is_locked", l.IsLocked).
		Set("is_recording_enabled", l.IsRecordingEnabled).
		Set("is_waiting_room_enabled", l.IsWaitingRoomEnabled).
		Set("mute_on_entry", l.MuteOnEntry).
		Set("allow_self_unmute", l.AllowSelfUnmute).
		Set("status", l.Status).
		Set("started_at", l.StartedAt).
		Set("ended_at", l.EndedAt).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"id": l.ID}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("lessonRepo.Update: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *lessonRepo) ListUpcomingUnreminded(ctx context.Context, from, to time.Time) ([]*entity.Lesson, error) {
	sql, args, _ := r.builder.
		Select(lessonCols).From("lessons").
		Where(sq.And{
			sq.Eq{"deleted_at": nil},
			sq.Eq{"reminder_sent_at": nil},
			sq.Eq{"status": entity.LessonStatusScheduled},
			sq.GtOrEq{"scheduled_at": from},
			sq.LtOrEq{"scheduled_at": to},
		}).ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("lessonRepo.ListUpcomingUnreminded: %w", err)
	}
	defer rows.Close()

	var out []*entity.Lesson
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *lessonRepo) ClaimReminder(ctx context.Context, id string) (bool, error) {
	sql, args, _ := r.builder.
		Update("lessons").Set("reminder_sent_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"reminder_sent_at": nil}}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("lessonRepo.ClaimReminder: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// ListLive — hozir jonli darslar. Avto-yakun ishchisi (4 soatlik limit va bo'sh
// xona grace'i) shu ro'yxat ustida ishlaydi; ro'yxat kichik bo'ladi (bir vaqtda
// jonli darslar soni cheklangan), shuning uchun sahifalash kerak emas.
func (r *lessonRepo) ListLive(ctx context.Context) ([]*entity.Lesson, error) {
	sql, args, _ := r.builder.
		Select(lessonCols).From("lessons").
		Where(sq.And{
			sq.Eq{"deleted_at": nil},
			sq.Eq{"status": entity.LessonStatusLive},
		}).
		OrderBy("started_at ASC").ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("lessonRepo.ListLive: %w", err)
	}
	defer rows.Close()

	var out []*entity.Lesson
	for rows.Next() {
		l, err := scanLesson(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ClaimEnd — darsni ATOMIK yakunlaydi (faqat hamon `live` bo'lsa).
//
// `WHERE status='live'` + RowsAffected — `ClaimReminder` bilan bir xil naqsh:
// ikki instans (yoki ishchi va mentorning "Yakunlash" tugmasi) bir vaqtda
// kelganda faqat bittasi g'olib bo'ladi va yozuvni to'xtatish / xonani
// o'chirish bir marta bajariladi.
func (r *lessonRepo) ClaimEnd(ctx context.Context, id string, endedAt time.Time) (bool, error) {
	sql, args, _ := r.builder.
		Update("lessons").
		Set("status", entity.LessonStatusEnded).
		Set("ended_at", endedAt).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.And{
			sq.Eq{"id": id},
			sq.Eq{"status": entity.LessonStatusLive},
			sq.Eq{"deleted_at": nil},
		}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("lessonRepo.ClaimEnd: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *lessonRepo) SoftDelete(ctx context.Context, id string) error {
	sql, args, err := r.builder.
		Update("lessons").
		Set("deleted_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("lessonRepo.SoftDelete: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

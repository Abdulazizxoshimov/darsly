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

// ── Retention (PRODUCT.md №5) ────────────────────────────────────────────────

// retentionQuery — retention so'rovlarining umumiy o'zagi: hali tirik (`ready`)
// yozuvlar, tugash vaqti bo'yicha eng eskisidan.
//
// `COALESCE(ended_at, created_at)`: `ended_at` nazariy jihatdan bo'sh bo'lishi
// mumkin (webhook yo'qolgan, holat qo'lda qo'yilgan). U holda yozuv retention
// hisobidan BUTUNLAY tushib qolardi va diskda abadiy qolardi — aynan
// oldini olmoqchi bo'lgan holatimiz.
func (r *recordingRepo) retentionQuery(endedBefore time.Time, limit uint64) sq.SelectBuilder {
	if limit == 0 {
		limit = 200
	}
	return r.builder.Select(recordingCols).From("recordings").
		Where(sq.Eq{"status": entity.RecordingStatusReady}).
		Where(sq.Expr("COALESCE(ended_at, created_at) <= ?", endedBefore)).
		OrderBy("COALESCE(ended_at, created_at) ASC").
		Limit(limit)
}

func (r *recordingRepo) scanRecordings(ctx context.Context, sql string, args []any, op string) ([]*entity.Recording, error) {
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.%s: %w", op, err)
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

func (r *recordingRepo) ListExpired(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error) {
	sql, args, err := r.retentionQuery(endedBefore, limit).ToSql()
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListExpired: %w", err)
	}
	return r.scanRecordings(ctx, sql, args, "ListExpired")
}

func (r *recordingRepo) ListExpiringUnwarned(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error) {
	sql, args, err := r.retentionQuery(endedBefore, limit).
		Where(sq.Eq{"retention_warned_at": nil}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListExpiringUnwarned: %w", err)
	}
	return r.scanRecordings(ctx, sql, args, "ListExpiringUnwarned")
}

// ClaimExpire — yozuvni ATOMIK ravishda `expired` deb belgilaydi.
//
// Faylni MinIO'dan o'chirishdan OLDIN chaqiriladi: shunda ikki instans bir
// vaqtda ishlaganda faqat bittasi o'chirish ishini bajaradi. Teskari tartib
// (avval o'chirish, keyin belgilash) DB uzilganda "obyekt yo'q, lekin holat
// `ready`" degan yolg'on qoldirardi — mentor yuklab olmoqchi bo'lib xato olardi.
func (r *recordingRepo) ClaimExpire(ctx context.Context, id string, deletedAt time.Time) (bool, error) {
	sql, args, _ := r.builder.
		Update("recordings").
		Set("status", entity.RecordingStatusExpired).
		Set("deleted_at", deletedAt).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"status": entity.RecordingStatusReady}}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("recordingRepo.ClaimExpire: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *recordingRepo) ClaimRetentionWarning(ctx context.Context, id string) (bool, error) {
	sql, args, _ := r.builder.
		Update("recordings").
		Set("retention_warned_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"retention_warned_at": nil}}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("recordingRepo.ClaimRetentionWarning: %w", err)
	}
	return tag.RowsAffected() == 1, nil
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

// ─── Qayta kodlash navbati (CRF) ─────────────────────────────────────────────

// EnqueueTranscode yozuvni navbatga qo'yadi. FAQAT `ready` yozuv navbatga
// tushadi: yiqilgan egress fayli yo'q va uni qayta kodlash ma'nosiz.
func (r *recordingRepo) EnqueueTranscode(ctx context.Context, egressID string) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("transcode_status", entity.TranscodePending).
		Where(sq.Eq{"egress_id": egressID, "status": entity.RecordingStatusReady}).
		ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

// ClaimTranscode navbatdan BITTA yozuvni atomik ravishda oladi.
//
// `FOR UPDATE SKIP LOCKED`: bir nechta backend nusxasi ishlayotganda ham bitta
// yozuv ikki marta kodlanmaydi (ikkalasi bir vaqtda tanlab, bir-birining
// natijasini ustiga yozardi — MinIO'da yarim yozilgan fayl qolardi).
// Navbat bo'sh bo'lsa `nil, nil` qaytadi.
func (r *recordingRepo) ClaimTranscode(ctx context.Context) (*entity.Recording, error) {
	const q = `
		UPDATE recordings SET transcode_status = $1, transcode_started_at = now()
		WHERE id = (
			SELECT id FROM recordings
			WHERE transcode_status = $2
			ORDER BY created_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING ` + recordingCols
	rec, err := scanRecording(r.db.QueryRow(ctx, q, entity.TranscodeRunning, entity.TranscodePending))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // navbat bo'sh — xato emas
	}
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ClaimTranscode: %w", err)
	}
	return rec, nil
}

// FinishTranscode muvaffaqiyatli natijani yozadi: yangi hajm + eskisi (taqqoslash uchun).
func (r *recordingRepo) FinishTranscode(ctx context.Context, id string, newSize, originalSize int64) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("transcode_status", entity.TranscodeDone).
		Set("size_bytes", newSize).
		Set("original_size_bytes", originalSize).
		Where(sq.Eq{"id": id}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

// FailTranscode — qayta kodlash yiqildi. Yozuvning O'ZI tegilmaydi: MinIO'dagi
// asl fayl joyida qoladi va ustoz uni baribir yuklab ola oladi.
func (r *recordingRepo) FailTranscode(ctx context.Context, id string) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("transcode_status", entity.TranscodeFailed).
		Where(sq.Eq{"id": id}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

// RequeueStaleTranscodes — ishlab turgan paytda o'lgan (deploy, crash) ishlarni
// navbatga qaytaradi. Usiz bunday yozuv abadiy `running` bo'lib qolardi.
func (r *recordingRepo) RequeueStaleTranscodes(ctx context.Context, olderThan time.Time) (int64, error) {
	sql, args, _ := r.builder.Update("recordings").
		Set("transcode_status", entity.TranscodePending).
		Where(sq.And{
			sq.Eq{"transcode_status": entity.TranscodeRunning},
			sq.Lt{"transcode_started_at": olderThan},
		}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func (r *recordingRepo) MarkFailed(ctx context.Context, egressID string, endedAt time.Time) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusFailed).
		Set("ended_at", endedAt).
		Where(sq.Eq{"egress_id": egressID}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	return err
}

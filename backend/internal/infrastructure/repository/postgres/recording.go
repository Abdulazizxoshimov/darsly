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

const recordingCols = "id, lesson_id, egress_id, object_key, status, duration_sec, size_bytes, started_at, ended_at, created_at, " +
	"telegram_file_id, telegram_message_id, telegram_chat_id, telegram_sent_at, telegram_error, telegram_attempts, cached_until, " +
	"content_offset_sec"

func scanRecording(row pgx.Row) (*entity.Recording, error) {
	r := &entity.Recording{}
	err := row.Scan(&r.ID, &r.LessonID, &r.EgressID, &r.ObjectKey, &r.Status,
		&r.DurationSec, &r.SizeBytes, &r.StartedAt, &r.EndedAt, &r.CreatedAt,
		&r.TelegramFileID, &r.TelegramMessageID, &r.TelegramChatID, &r.TelegramSentAt,
		&r.TelegramError, &r.TelegramAttempts, &r.CachedUntil, &r.ContentOffsetSec)
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
		if isUniqueViolation(err) {
			return apperr.Conflict("recording already exists for this egress")
		}
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

// ListExpired — Telegram arxivi O'CHIQ bo'lgan o'rnatish uchun (eski xulq).
func (r *recordingRepo) ListExpired(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error) {
	sql, args, err := r.retentionQuery(endedBefore, limit).ToSql()
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListExpired: %w", err)
	}
	return r.scanRecordings(ctx, sql, args, "ListExpired")
}

// ListArchivable — muddati o'tgan VA Telegramda tasdiqlangan yozuvlar.
//
// ⭐ `telegram_sent_at IS NOT NULL` — mahsulotning eng muhim kafolati
// («video hech qachon yo'qolmaydi») aynan shu bitta shartda. U SQL'da,
// chunki bu yerdan qaytgan har qator MinIO'dan o'chirilishga nomzod: filtrni
// unutish yoki noto'g'ri joyga qo'yish qaytarib bo'lmaydigan yo'qotish
// bo'lardi.
func (r *recordingRepo) ListArchivable(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error) {
	sql, args, err := r.retentionQuery(endedBefore, limit).
		Where(sq.NotEq{"telegram_sent_at": nil}).
		Where(sq.Eq{"cached_until": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListArchivable: %w", err)
	}
	return r.scanRecordings(ctx, sql, args, "ListArchivable")
}

// ListExpiredUnconfirmed — muddati o'tgan, lekin Telegramga TUSHMAGAN yozuvlar.
//
// Bular o'chirilmaydi (fayl serverda qoladi va disk yeydi) — mahsulot qarori
// shunday: ma'lumot yo'qotishdan ko'ra to'lgan disk yaxshiroq. Mentor
// ogohlantiriladi, operator esa `retention_warned_at` bo'yicha ularni
// ko'radi.
func (r *recordingRepo) ListExpiredUnconfirmed(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error) {
	sql, args, err := r.retentionQuery(endedBefore, limit).
		Where(sq.Eq{"telegram_sent_at": nil}).
		Where(sq.Eq{"cached_until": nil}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListExpiredUnconfirmed: %w", err)
	}
	return r.scanRecordings(ctx, sql, args, "ListExpiredUnconfirmed")
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

// ClaimArchive — `ready → archived` (server nusxasi o'chirilmoqda, Telegramda qoladi).
//
// `ClaimExpire` bilan bir xil naqsh, ikkita farq bilan:
//   - yakuniy holat `archived` (yozuv YO'QOLMADI, joyi o'zgardi);
//   - `telegram_sent_at IS NOT NULL` sharti WHERE'da takrorlanadi (ro'yxat
//     olingandan keyingi oynada holat o'zgargan bo'lishi mumkin).
func (r *recordingRepo) ClaimArchive(ctx context.Context, id string, deletedAt time.Time) (bool, error) {
	sql, args, _ := r.builder.
		Update("recordings").
		Set("status", entity.RecordingStatusArchived).
		Set("deleted_at", deletedAt).
		Where(sq.And{
			sq.Eq{"id": id},
			sq.Eq{"status": entity.RecordingStatusReady},
			sq.NotEq{"telegram_sent_at": nil},
		}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("recordingRepo.ClaimArchive: %w", err)
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

// CountActive — hozir `recording` holatidagi yozuvlar soni (global egress cap).
func (r *recordingRepo) CountActive(ctx context.Context) (int, error) {
	sql, args, _ := r.builder.Select("COUNT(*)").From("recordings").
		Where(sq.Eq{"status": entity.RecordingStatusRecording}).ToSql()
	var n int
	if err := r.db.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		return 0, fmt.Errorf("recordingRepo.CountActive: %w", err)
	}
	return n, nil
}

// markReadyStates — MarkReady qaysi holatlardan o'tishga ruxsat beradi.
// Faqat `recording` (egress ketmoqda) va `processing` (Stop bosilgan, webhook
// kutilyapti). Boshqa har holatdan o'tish TAKROR/KECH hodisa demak.
var markReadyStates = []string{entity.RecordingStatusRecording, entity.RecordingStatusProcessing}

func (r *recordingRepo) MarkReady(ctx context.Context, egressID, objectKey string, durationSec int, sizeBytes int64, endedAt time.Time) (bool, error) {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusReady).
		Set("object_key", objectKey).
		Set("duration_sec", durationSec).
		Set("size_bytes", sizeBytes).
		Set("ended_at", endedAt).
		Where(sq.Eq{"egress_id": egressID}).
		Where(sq.Eq{"status": markReadyStates}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// ─── Telegram arxivi ─────────────────────────────────────────────────────────

// EnqueueTelegram — yozuvni navbatga qo'yadi (`telegram_next_attempt_at = NOW()`).
//
// FAQAT `ready` yozuv navbatga tushadi (yiqilgan egressning fayli yo'q) va
// FAQAT hali yuborilmagani (`telegram_sent_at IS NULL`) — webhook takroran
// kelsa allaqachon arxivlangan yozuv qayta yuborilmasin (guruhda dublikat
// video paydo bo'lardi).
func (r *recordingRepo) EnqueueTelegram(ctx context.Context, egressID string) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("telegram_next_attempt_at", sq.Expr("NOW()")).
		Where(sq.And{
			sq.Eq{"egress_id": egressID, "status": entity.RecordingStatusReady},
			sq.Eq{"telegram_sent_at": nil},
		}).ToSql()
	_, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("recordingRepo.EnqueueTelegram: %w", err)
	}
	return nil
}

// ClaimTelegramUpload — navbatdan bitta yozuvni atomik oladi.
//
// `FOR UPDATE SKIP LOCKED` — `ClaimTranscode` bilan bir xil sabab: ikki
// instans bir faylni ikki marta yuborsa guruhda ikkita bir xil video paydo
// bo'ladi va buni orqaga qaytarib bo'lmaydi.
//
// Urinish hisobi DARHOL oshadi va keyingi urinish 30 daqiqaga suriladi:
// yuklash soatlab ketishi mumkin, shu paytda jarayon o'lsa (deploy) yozuv
// o'z-o'zidan navbatga qaytadi. `maxAttempts` ga yetganlar tanlanmaydi —
// ular mentor aralashuvini kutadi.
func (r *recordingRepo) ClaimTelegramUpload(ctx context.Context, now time.Time, maxAttempts int) (*entity.Recording, error) {
	const q = `
		UPDATE recordings SET
			telegram_attempts        = telegram_attempts + 1,
			telegram_next_attempt_at = $1::timestamptz + interval '30 minutes'
		WHERE id = (
			SELECT id FROM recordings
			WHERE telegram_sent_at IS NULL
			  AND status = 'ready'
			  -- Transcode TUGASHINI kutamiz: aks holda Telegram'ga XOM (siqilmagan)
			  -- fayl ketardi (transcode server nusxasini kichraytiradi, lekin
			  -- Telegram undan oldin ulgursa katta nusxa ketib qolardi — o'lchangan:
			  -- sinov 229 MB xom ketdi, server esa 18 MB ga tushdi). done/failed/
			  -- skipped — terminal; NULL (egress/legacy) — kutmaymiz.
			  AND transcode_status IS DISTINCT FROM 'pending'
			  AND transcode_status IS DISTINCT FROM 'running'
			  AND telegram_next_attempt_at IS NOT NULL
			  AND telegram_next_attempt_at <= $1
			  AND telegram_attempts < $2
			ORDER BY telegram_next_attempt_at
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING ` + recordingCols
	rec, err := scanRecording(r.db.QueryRow(ctx, q, now, maxAttempts))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // navbat bo'sh — xato emas
	}
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ClaimTelegramUpload: %w", err)
	}
	return rec, nil
}

// MarkTelegramSent — muvaffaqiyat. `telegram_next_attempt_at` tozalanadi
// (navbatdan chiqadi) va xato matni ham (eski nosozlik UI'da osilib qolmasin).
func (r *recordingRepo) MarkTelegramSent(ctx context.Context, id string, chatID, messageID int64, fileID string) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("telegram_chat_id", chatID).
		Set("telegram_message_id", messageID).
		Set("telegram_file_id", nullIfEmpty(fileID)).
		Set("telegram_sent_at", sq.Expr("NOW()")).
		Set("telegram_next_attempt_at", nil).
		Set("telegram_error", nil).
		Where(sq.Eq{"id": id}).ToSql()
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("recordingRepo.MarkTelegramSent: %w", err)
	}
	return nil
}

func (r *recordingRepo) MarkTelegramFailed(ctx context.Context, id, errMsg string, nextAttempt *time.Time) error {
	q := r.builder.Update("recordings").
		Set("telegram_error", nullIfEmpty(errMsg)).
		Where(sq.Eq{"id": id})
	if nextAttempt != nil {
		q = q.Set("telegram_next_attempt_at", *nextAttempt)
	} else {
		// nil → urinishlar tugadi: navbatdan chiqariladi. Fayl esa
		// O'CHIRILMAYDI (retention `telegram_sent_at` ni talab qiladi).
		q = q.Set("telegram_next_attempt_at", nil)
	}
	sql, args, _ := q.ToSql()
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("recordingRepo.MarkTelegramFailed: %w", err)
	}
	return nil
}

// ─── Telegramdan qaytarib olish (restore) ────────────────────────────────────

func (r *recordingRepo) ClaimRestore(ctx context.Context, id string) (bool, error) {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusRestoring).
		Where(sq.And{
			sq.Eq{"id": id, "status": entity.RecordingStatusArchived},
			sq.NotEq{"telegram_file_id": nil},
		}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("recordingRepo.ClaimRestore: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

func (r *recordingRepo) FinishRestore(ctx context.Context, id, objectKey string, cachedUntil time.Time) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusReady).
		Set("object_key", objectKey).
		Set("cached_until", cachedUntil).
		Set("deleted_at", nil).
		Where(sq.Eq{"id": id, "status": entity.RecordingStatusRestoring}).ToSql()
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("recordingRepo.FinishRestore: %w", err)
	}
	return nil
}

func (r *recordingRepo) FailRestore(ctx context.Context, id, errMsg string) error {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusArchived).
		Set("telegram_error", nullIfEmpty(errMsg)).
		Where(sq.Eq{"id": id, "status": entity.RecordingStatusRestoring}).ToSql()
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("recordingRepo.FailRestore: %w", err)
	}
	return nil
}

func (r *recordingRepo) ListByStatus(ctx context.Context, status string, limit uint64) ([]*entity.Recording, error) {
	if limit == 0 {
		limit = 50
	}
	sql, args, err := r.builder.Select(recordingCols).From("recordings").
		Where(sq.Eq{"status": status}).
		OrderBy("created_at ASC").Limit(limit).ToSql()
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListByStatus: %w", err)
	}
	return r.scanRecordings(ctx, sql, args, "ListByStatus")
}

func (r *recordingRepo) ListCacheExpired(ctx context.Context, now time.Time, limit uint64) ([]*entity.Recording, error) {
	if limit == 0 {
		limit = 200
	}
	sql, args, err := r.builder.Select(recordingCols).From("recordings").
		Where(sq.Eq{"status": entity.RecordingStatusReady}).
		Where(sq.NotEq{"cached_until": nil}).
		Where(sq.Lt{"cached_until": now}).
		OrderBy("cached_until ASC").Limit(limit).ToSql()
	if err != nil {
		return nil, fmt.Errorf("recordingRepo.ListCacheExpired: %w", err)
	}
	return r.scanRecordings(ctx, sql, args, "ListCacheExpired")
}

// ClaimCacheEvict — kesh nusxasini `archived` ga qaytaradi.
//
// `cached_until` shartda: mentor tiklashdan keyin yana ochsa muddat
// uzaytirilgan bo'lishi mumkin va o'sha paytda faylni tortib olish
// «video o'rtasida to'xtadi» degan holatni berardi.
func (r *recordingRepo) ClaimCacheEvict(ctx context.Context, id string) (bool, error) {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusArchived).
		Set("cached_until", nil).
		Where(sq.And{
			sq.Eq{"id": id, "status": entity.RecordingStatusReady},
			// `sq.Lt` bilan emas, `sq.Expr` bilan: Lt qiymat sifatida
			// Sqlizer'ni QABUL QILMAYDI — u `NOW()` ni oddiy parametr deb
			// bog'lardi va shart hech qachon bajarilmasdi.
			sq.Expr("cached_until IS NOT NULL AND cached_until < NOW()"),
		}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, fmt.Errorf("recordingRepo.ClaimCacheEvict: %w", err)
	}
	return tag.RowsAffected() == 1, nil
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
func (r *recordingRepo) FinishTranscode(ctx context.Context, id string, newSize, originalSize int64, durationSec, trimmedSec int) error {
	q := r.builder.Update("recordings").
		Set("transcode_status", entity.TranscodeDone).
		Set("size_bytes", newSize).
		Set("original_size_bytes", originalSize)
	// Davomiylik faqat haqiqatan o'lchangan bo'lsa yoziladi — 0 bilan ustiga
	// yozish arxiv sahifasidagi vaqt shkalasini yo'q qilardi.
	if durationSec > 0 {
		q = q.Set("duration_sec", durationSec)
	}
	// Kesilgan miqdor QO'SHILADI, ustiga yozilmaydi: yozuv qayta transkod
	// qilinsa (masalan ishchi qayta yurgizilsa) siljish to'planib boradi va
	// har safar noldan hisoblansa chat sinxroni buzilardi.
	if trimmedSec > 0 {
		q = q.Set("content_offset_sec", sq.Expr("content_offset_sec + ?", trimmedSec))
	}
	sql, args, _ := q.Where(sq.Eq{"id": id}).ToSql()
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

func (r *recordingRepo) MarkFailed(ctx context.Context, egressID string, endedAt time.Time) (bool, error) {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusFailed).
		Set("ended_at", endedAt).
		Where(sq.Eq{"egress_id": egressID}).
		Where(sq.Eq{"status": markReadyStates}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

// FailStale — eskirgan `recording` yozuvlarni `failed` qiladi.
func (r *recordingRepo) FailStale(ctx context.Context, olderThan, endedAt time.Time) (int64, error) {
	sql, args, _ := r.builder.Update("recordings").
		Set("status", entity.RecordingStatusFailed).
		Set("ended_at", endedAt).
		Where(sq.Eq{"status": entity.RecordingStatusRecording}).
		Where(sq.Lt{"started_at": olderThan}).ToSql()
	tag, err := r.db.Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("recordingRepo.FailStale: %w", err)
	}
	return tag.RowsAffected(), nil
}

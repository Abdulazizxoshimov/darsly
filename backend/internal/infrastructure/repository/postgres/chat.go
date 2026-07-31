package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

type chatRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewChatRepo(p *pg.Postgres) repository.ChatRepository {
	return &chatRepo{db: p.DB, builder: p.Builder}
}

// chatColumns — o'qish uchun ustunlar. Bir joyda: `Select` va `Scan` tartibi
// mos kelmasa xato kompilyatsiyada emas, ish paytida (noto'g'ri maydonda)
// chiqadi va topish qiyin.
var chatColumns = []string{
	"id", "lesson_id", "sender_identity", "sender_name", "body", "to_identity", "created_at",
	"file_key", "file_name", "file_size", "file_mime",
}

// scanChat — [chatColumns] tartibida bitta qatorni o'qiydi.
//
// Fayl ustunlari NULL bo'lishi mumkin, shuning uchun ko'rsatkichlarga o'qiladi
// va `File` faqat kalit bor bo'lsa yasaladi (DB'dagi CHECK ham shuni kafolatlaydi).
func scanChat(row pgx.Row) (*entity.ChatMessage, error) {
	m := &entity.ChatMessage{}
	var key, name, mime *string
	var size *int64
	if err := row.Scan(
		&m.ID, &m.LessonID, &m.SenderIdentity, &m.SenderName, &m.Body, &m.ToIdentity, &m.CreatedAt,
		&key, &name, &size, &mime,
	); err != nil {
		return nil, err
	}
	if key != nil && *key != "" {
		f := &entity.ChatFile{Key: *key}
		if name != nil {
			f.Name = *name
		}
		if size != nil {
			f.Size = *size
		}
		if mime != nil {
			f.Mime = *mime
		}
		m.File = f
	}
	return m, nil
}

func (r *chatRepo) Create(ctx context.Context, m *entity.ChatMessage) error {
	var key, name, mime *string
	var size *int64
	if m.File != nil {
		key, name, mime, size = &m.File.Key, &m.File.Name, &m.File.Mime, &m.File.Size
	}
	sql, args, err := r.builder.
		Insert("chat_messages").
		Columns("id", "lesson_id", "sender_identity", "sender_name", "body", "to_identity", "created_at",
			"file_key", "file_name", "file_size", "file_mime").
		Values(m.ID, m.LessonID, m.SenderIdentity, m.SenderName, m.Body, m.ToIdentity, m.CreatedAt,
			key, name, size, mime).
		ToSql()
	if err != nil {
		return fmt.Errorf("chatRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("chatRepo.Create: %w", err)
	}
	return nil
}

func (r *chatRepo) ListByLesson(ctx context.Context, lessonID, viewerIdentity string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := r.builder.
		Select(chatColumns...).
		From("chat_messages").
		Where(sq.Eq{"lesson_id": lessonID}).
		// MODERATSIYA (№6): o'chirilgan xabar hech kimga qaytmaydi — mentorga ham.
		// Qator DB'da qoladi (moderatsiya izi), lekin API uni ko'rsatmaydi.
		Where(sq.Eq{"deleted_at": nil})

	// KO'RINUVCHANLIK. Shaxsiy xabarni faqat ikki tomon ko'radi va bu shart SQL'da
	// qo'llanadi — begona DM jarayon xotirasiga umuman kelmasin.
	if viewerIdentity == "" {
		q = q.Where(sq.Eq{"to_identity": nil}) // faqat ommaviy
	} else {
		q = q.Where(sq.Or{
			sq.Eq{"to_identity": nil},
			sq.Eq{"to_identity": viewerIdentity},
			sq.Eq{"sender_identity": viewerIdentity},
		})
	}

	if before != nil {
		q = q.Where(sq.Lt{"created_at": *before}) // kursor: faqat undan eski xabarlar
	}
	// Eng yangidan eskiga — uzun darsda oxirgi xabarlar ko'rinadi (offset yo'q, arzon).
	sql, args, err := q.OrderBy("created_at DESC").Limit(uint64(limit)).ToSql()
	if err != nil {
		return nil, fmt.Errorf("chatRepo.ListByLesson: %w", err)
	}
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("chatRepo.ListByLesson: %w", err)
	}
	defer rows.Close()

	var out []*entity.ChatMessage
	for rows.Next() {
		m, err := scanChat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *chatRepo) SoftDelete(ctx context.Context, lessonID, messageID, deletedBy string) (*entity.ChatMessage, error) {
	sql, args, err := r.builder.
		Update("chat_messages").
		Set("deleted_at", sq.Expr("NOW()")).
		Set("deleted_by", deletedBy).
		Where(sq.Eq{"id": messageID, "lesson_id": lessonID, "deleted_at": nil}).
		Suffix("RETURNING " + strings.Join(chatColumns, ", ")).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("chatRepo.SoftDelete: %w", err)
	}
	m, err := scanChat(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		// Yo'q, begona darsniki, yoki allaqachon o'chirilgan — uchalasi ham
		// chaqiruvchi uchun bir xil javob berishi kerak (mavjudlik oraklini
		// bermaslik uchun).
		return nil, apperr.NotFound("chat message")
	}
	if err != nil {
		return nil, fmt.Errorf("chatRepo.SoftDelete: %w", err)
	}
	return m, nil
}

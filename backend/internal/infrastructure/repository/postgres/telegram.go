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

type telegramRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewTelegramRepo(p *pg.Postgres) repository.TelegramRepository {
	return &telegramRepo{db: p.DB, builder: p.Builder}
}

// ─── Mentor ↔ Telegram akkaunt ───────────────────────────────────────────────

// linkUserCols — bog'lanish holati uchun yetarli MINIMAL to'plam.
//
// To'liq `users` qatorini o'qimaymiz: bu yerda parol hash'i va boshqa PII
// keraksiz, ular esa bir marta o'qilib log'ga yoki javobga sizib chiqishi
// mumkin bo'lgan yuza. Faqat kerakligi olinadi.
const linkUserCols = "id, email, full_name, role, is_active, telegram_user_id, telegram_username, telegram_linked_at"

func scanLinkUser(row pgx.Row) (*entity.User, error) {
	u := &entity.User{}
	err := row.Scan(&u.ID, &u.Email, &u.FullName, &u.Role, &u.IsActive,
		&u.TelegramUserID, &u.TelegramUsername, &u.TelegramLinkedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

// LinkUser — bir tranzaksiyada: eski egadan uzish + yangisiga bog'lash.
//
// Ikki UPDATE ATOMIK bo'lishi SHART. Aks holda birinchisi o'tib ikkinchisi
// yiqilsa, Telegram akkaunti hech kimga bog'lanmagan holatda qolardi va
// mentor "bog'ladim, lekin bot meni tanimayapti" degan holatga tushardi.
func (r *telegramRepo) LinkUser(ctx context.Context, userID string, telegramUserID int64, username string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("telegramRepo.LinkUser: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Bu Telegram akkaunti boshqa foydalanuvchida bo'lsa — uziladi (UNIQUE
	// indeks bilan to'qnashmaslik uchun; qarang interfeys izohi).
	relSQL, relArgs, _ := r.builder.Update("users").
		Set("telegram_user_id", nil).
		Set("telegram_username", nil).
		Set("telegram_linked_at", nil).
		Where(sq.And{sq.Eq{"telegram_user_id": telegramUserID}, sq.NotEq{"id": userID}}).ToSql()
	if _, err := tx.Exec(ctx, relSQL, relArgs...); err != nil {
		return fmt.Errorf("telegramRepo.LinkUser: release: %w", err)
	}

	setSQL, setArgs, _ := r.builder.Update("users").
		Set("telegram_user_id", telegramUserID).
		Set("telegram_username", nullIfEmpty(username)).
		Set("telegram_linked_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": userID}).ToSql()
	tag, err := tx.Exec(ctx, setSQL, setArgs...)
	if err != nil {
		return fmt.Errorf("telegramRepo.LinkUser: set: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return apperr.NotFound("user")
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("telegramRepo.LinkUser: commit: %w", err)
	}
	return nil
}

func (r *telegramRepo) UnlinkUser(ctx context.Context, userID string) error {
	sql, args, _ := r.builder.Update("users").
		Set("telegram_user_id", nil).
		Set("telegram_username", nil).
		Set("telegram_linked_at", nil).
		Where(sq.Eq{"id": userID}).ToSql()
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("telegramRepo.UnlinkUser: %w", err)
	}
	return nil
}

func (r *telegramRepo) GetUserByTelegramID(ctx context.Context, telegramUserID int64) (*entity.User, error) {
	sql, args, _ := r.builder.Select(linkUserCols).From("users").
		Where(sq.Eq{"telegram_user_id": telegramUserID, "deleted_at": nil}).ToSql()
	u, err := scanLinkUser(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("user")
	}
	return u, err
}

func (r *telegramRepo) GetLink(ctx context.Context, userID string) (*entity.User, error) {
	sql, args, _ := r.builder.Select(linkUserCols).From("users").
		Where(sq.Eq{"id": userID, "deleted_at": nil}).ToSql()
	u, err := scanLinkUser(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("user")
	}
	return u, err
}

// ─── Guruhlar ────────────────────────────────────────────────────────────────

const telegramChatCols = "chat_id, title, type, mentor_id, is_active, added_at, updated_at"

func scanTelegramChat(row pgx.Row) (*entity.TelegramChat, error) {
	c := &entity.TelegramChat{}
	err := row.Scan(&c.ChatID, &c.Title, &c.Type, &c.MentorID, &c.IsActive, &c.AddedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// UpsertChat — `my_chat_member` hodisasidan.
//
// `mentor_id` uchun `COALESCE(EXCLUDED.mentor_id, telegram_chats.mentor_id)`:
// yangi qiymat NULL bo'lsa (hodisa bog'lanmagan odamdan kelgan) MAVJUD egasi
// saqlanadi. Oddiy `SET mentor_id = EXCLUDED.mentor_id` bo'lsa guruh nomi
// o'zgargan paytdagi hodisa egalikni yo'q qilib yuborardi va mentor o'z
// guruhini ro'yxatda ko'rmay qolardi.
func (r *telegramRepo) UpsertChat(ctx context.Context, chat *entity.TelegramChat) error {
	const q = `
		INSERT INTO telegram_chats (chat_id, title, type, mentor_id, is_active, added_at, updated_at)
		VALUES ($1, $2, $3, $4, TRUE, NOW(), NOW())
		ON CONFLICT (chat_id) DO UPDATE SET
			title      = EXCLUDED.title,
			type       = EXCLUDED.type,
			mentor_id  = COALESCE(EXCLUDED.mentor_id, telegram_chats.mentor_id),
			is_active  = TRUE,
			updated_at = NOW()`
	if _, err := r.db.Exec(ctx, q, chat.ChatID, chat.Title, chat.Type, chat.MentorID); err != nil {
		return fmt.Errorf("telegramRepo.UpsertChat: %w", err)
	}
	return nil
}

func (r *telegramRepo) DeactivateChat(ctx context.Context, chatID int64) error {
	sql, args, _ := r.builder.Update("telegram_chats").
		Set("is_active", false).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"chat_id": chatID}).ToSql()
	if _, err := r.db.Exec(ctx, sql, args...); err != nil {
		return fmt.Errorf("telegramRepo.DeactivateChat: %w", err)
	}
	return nil
}

func (r *telegramRepo) ListChatsByMentor(ctx context.Context, mentorID string) ([]*entity.TelegramChat, error) {
	sql, args, _ := r.builder.Select(telegramChatCols).From("telegram_chats").
		Where(sq.Eq{"mentor_id": mentorID, "is_active": true}).
		OrderBy("title ASC").Limit(50).ToSql()
	rows, err := r.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("telegramRepo.ListChatsByMentor: %w", err)
	}
	defer rows.Close()

	var out []*entity.TelegramChat
	for rows.Next() {
		c, err := scanTelegramChat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *telegramRepo) GetChat(ctx context.Context, chatID int64) (*entity.TelegramChat, error) {
	sql, args, _ := r.builder.Select(telegramChatCols).From("telegram_chats").
		Where(sq.Eq{"chat_id": chatID}).ToSql()
	c, err := scanTelegramChat(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("telegram chat")
	}
	return c, err
}

// nullIfEmpty — bo'sh satrni NULL qiladi.
//
// Nega: `telegram_username` ixtiyoriy (Telegramda username bo'lmasligi
// mumkin). Bo'sh satr yozilsa UI'da `@` degan ma'nosiz yorliq chiqardi.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

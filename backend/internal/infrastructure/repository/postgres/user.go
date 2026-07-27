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

var allowedUserSortCols = map[string]bool{
	"created_at": true, "updated_at": true, "full_name": true,
	"email": true, "last_login_at": true,
}

type userRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewUserRepo(p *pg.Postgres) repository.UserRepository {
	return &userRepo{db: p.DB, builder: p.Builder}
}

const userCols = "id, email, password_hash, full_name, avatar_url, color, role, timezone, language, is_active, last_login_at, created_at, updated_at, deleted_at"

func scanUser(row pgx.Row) (*entity.User, error) {
	u := &entity.User{}
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName,
		&u.AvatarURL, &u.Color, &u.Role, &u.Timezone, &u.Language,
		&u.IsActive, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepo) Create(ctx context.Context, user *entity.User) error {
	sql, args, err := r.builder.
		Insert("users").
		Columns("id", "email", "password_hash", "full_name", "avatar_url", "color", "role", "timezone", "language", "is_active", "created_at", "updated_at").
		Values(user.ID, user.Email, user.PasswordHash, user.FullName, user.AvatarURL, user.Color, user.Role, user.Timezone, user.Language, user.IsActive, user.CreatedAt, user.UpdatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("email already exists")
		}
		return fmt.Errorf("userRepo.Create: %w", err)
	}
	return nil
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*entity.User, error) {
	sql, args, err := r.builder.
		Select(userCols).From("users").
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByID: %w", err)
	}
	u, err := scanUser(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("user")
	}
	return u, err
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	sql, args, err := r.builder.
		Select(userCols).From("users").
		Where(sq.And{sq.Eq{"email": email}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByEmail: %w", err)
	}
	u, err := scanUser(r.db.QueryRow(ctx, sql, args...))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperr.NotFound("user")
	}
	return u, err
}

func (r *userRepo) List(ctx context.Context, filter *entity.UserFilter) ([]*entity.User, int, error) {
	where := sq.And{sq.Eq{"deleted_at": nil}}
	if filter.Search != "" {
		where = append(where, sq.Or{
			sq.ILike{"full_name": "%" + filter.Search + "%"},
			sq.ILike{"email": "%" + filter.Search + "%"},
		})
	}
	if filter.Role != "" {
		where = append(where, sq.Eq{"role": filter.Role})
	}
	if filter.IsActive != nil {
		where = append(where, sq.Eq{"is_active": *filter.IsActive})
	}
	if filter.CreatedFrom != nil {
		where = append(where, sq.GtOrEq{"created_at": *filter.CreatedFrom})
	}
	if filter.CreatedTo != nil {
		where = append(where, sq.LtOrEq{"created_at": *filter.CreatedTo})
	}

	var total int
	cntSQL, cntArgs, _ := r.builder.Select("COUNT(*)").From("users").Where(where).ToSql()
	if err := r.db.QueryRow(ctx, cntSQL, cntArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("userRepo.List count: %w", err)
	}

	sortBy := "created_at"
	if allowedUserSortCols[filter.SortBy] {
		sortBy = filter.SortBy
	}
	order := "DESC"
	if filter.GetSortOrder() == entity.SortAsc {
		order = "ASC"
	}

	dataSQL, dataArgs, _ := r.builder.
		Select(userCols).From("users").Where(where).
		OrderBy(sortBy + " " + order).
		Limit(uint64(filter.GetLimit())).Offset(uint64(filter.Offset())).
		ToSql()

	rows, err := r.db.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("userRepo.List query: %w", err)
	}
	defer rows.Close()

	var users []*entity.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *userRepo) Update(ctx context.Context, user *entity.User) error {
	sql, args, err := r.builder.
		Update("users").
		Set("full_name", user.FullName).
		Set("avatar_url", user.AvatarURL).
		Set("color", user.Color).
		Set("role", user.Role).
		Set("timezone", user.Timezone).
		Set("language", user.Language).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"id": user.ID}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("userRepo.Update: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *userRepo) UpdatePassword(ctx context.Context, userID, passwordHash string) error {
	sql, args, err := r.builder.
		Update("users").
		Set("password_hash", passwordHash).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": userID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("userRepo.UpdatePassword: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *userRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	sql, args, err := r.builder.
		Update("users").
		Set("last_login_at", sq.Expr("NOW()")).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": userID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("userRepo.UpdateLastLogin: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *userRepo) SoftDelete(ctx context.Context, id string) error {
	sql, args, err := r.builder.
		Update("users").
		Set("deleted_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"id": id}, sq.Eq{"deleted_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("userRepo.SoftDelete: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *userRepo) DeleteHard(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	return err
}

func (r *userRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`, email,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("userRepo.ExistsByEmail: %w", err)
	}
	return exists, nil
}

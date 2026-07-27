package postgres

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	pg "github.com/zoom/darsly/internal/pkg/postgres"
)

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint violation.
// Defined once here; all files in package postgres share it.
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type authRepo struct {
	db      *pgxpool.Pool
	builder sq.StatementBuilderType
}

func NewAuthRepo(p *pg.Postgres) repository.AuthRepository {
	return &authRepo{db: p.DB, builder: p.Builder}
}

func (r *authRepo) CreateRefreshToken(ctx context.Context, rt *entity.RefreshToken) error {
	sql, args, err := r.builder.
		Insert("refresh_tokens").
		Columns("id", "user_id", "token_hash", "user_agent", "ip_address", "expires_at", "created_at").
		Values(rt.ID, rt.UserID, rt.TokenHash, rt.UserAgent, rt.IPAddress, rt.ExpiresAt, rt.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("authRepo.CreateRefreshToken: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("refresh token already exists")
		}
		return fmt.Errorf("authRepo.CreateRefreshToken: %w", err)
	}
	return nil
}

func (r *authRepo) GetRefreshTokenByHash(ctx context.Context, hash string) (*entity.RefreshToken, error) {
	sql, args, err := r.builder.
		Select("id", "user_id", "token_hash", "user_agent", "ip_address", "expires_at", "revoked_at", "created_at").
		From("refresh_tokens").
		Where(sq.And{sq.Eq{"token_hash": hash}, sq.Eq{"revoked_at": nil}, sq.Expr("expires_at > NOW()")}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("authRepo.GetRefreshTokenByHash: %w", err)
	}
	rt := &entity.RefreshToken{}
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&rt.ID, &rt.UserID, &rt.TokenHash, &rt.UserAgent, &rt.IPAddress,
		&rt.ExpiresAt, &rt.RevokedAt, &rt.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("refresh token")
		}
		return nil, fmt.Errorf("authRepo.GetRefreshTokenByHash: %w", err)
	}
	return rt, nil
}

func (r *authRepo) RevokeRefreshToken(ctx context.Context, id string) error {
	sql, args, err := r.builder.
		Update("refresh_tokens").
		Set("revoked_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("authRepo.RevokeRefreshToken: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *authRepo) RevokeAllUserTokens(ctx context.Context, userID string) error {
	sql, args, err := r.builder.
		Update("refresh_tokens").
		Set("revoked_at", sq.Expr("NOW()")).
		Where(sq.And{sq.Eq{"user_id": userID}, sq.Eq{"revoked_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("authRepo.RevokeAllUserTokens: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *authRepo) DeleteExpiredTokens(ctx context.Context) error {
	sql, args, err := r.builder.
		Delete("refresh_tokens").
		Where(sq.Or{sq.Expr("expires_at < NOW()"), sq.NotEq{"revoked_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("authRepo.DeleteExpiredTokens: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *authRepo) CreatePasswordReset(ctx context.Context, pr *entity.PasswordReset) error {
	sql, args, err := r.builder.
		Insert("password_resets").
		Columns("id", "user_id", "token_hash", "expires_at", "created_at").
		Values(pr.ID, pr.UserID, pr.TokenHash, pr.ExpiresAt, pr.CreatedAt).
		ToSql()
	if err != nil {
		return fmt.Errorf("authRepo.CreatePasswordReset: %w", err)
	}
	if _, err = r.db.Exec(ctx, sql, args...); err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("password reset token already exists")
		}
		return fmt.Errorf("authRepo.CreatePasswordReset: %w", err)
	}
	return nil
}

func (r *authRepo) GetPasswordResetByHash(ctx context.Context, hash string) (*entity.PasswordReset, error) {
	sql, args, err := r.builder.
		Select("id", "user_id", "token_hash", "expires_at", "used_at", "created_at").
		From("password_resets").
		Where(sq.And{sq.Eq{"token_hash": hash}, sq.Eq{"used_at": nil}, sq.Expr("expires_at > NOW()")}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("authRepo.GetPasswordResetByHash: %w", err)
	}
	pr := &entity.PasswordReset{}
	err = r.db.QueryRow(ctx, sql, args...).Scan(
		&pr.ID, &pr.UserID, &pr.TokenHash, &pr.ExpiresAt, &pr.UsedAt, &pr.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperr.NotFound("password reset token")
		}
		return nil, fmt.Errorf("authRepo.GetPasswordResetByHash: %w", err)
	}
	return pr, nil
}

func (r *authRepo) MarkPasswordResetUsed(ctx context.Context, id string) error {
	sql, args, err := r.builder.
		Update("password_resets").
		Set("used_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("authRepo.MarkPasswordResetUsed: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

func (r *authRepo) DeleteExpiredPasswordResets(ctx context.Context) error {
	sql, args, err := r.builder.
		Delete("password_resets").
		Where(sq.Or{sq.Expr("expires_at < NOW()"), sq.NotEq{"used_at": nil}}).
		ToSql()
	if err != nil {
		return fmt.Errorf("authRepo.DeleteExpiredPasswordResets: %w", err)
	}
	_, err = r.db.Exec(ctx, sql, args...)
	return err
}

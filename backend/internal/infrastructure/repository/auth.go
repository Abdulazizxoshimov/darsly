package repository

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type AuthRepository interface {
	CreateRefreshToken(ctx context.Context, rt *entity.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, hash string) (*entity.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id string) error
	RevokeAllUserTokens(ctx context.Context, userID string) error
	DeleteExpiredTokens(ctx context.Context) error

	CreatePasswordReset(ctx context.Context, pr *entity.PasswordReset) error
	GetPasswordResetByHash(ctx context.Context, hash string) (*entity.PasswordReset, error)
	// InvalidateUserPasswordResets foydalanuvchining ishlatilmagan reset tokenlarini
	// bekor qiladi (yangi token berishdan oldin — eskilari yaroqsiz bo'lsin).
	InvalidateUserPasswordResets(ctx context.Context, userID string) error
	MarkPasswordResetUsed(ctx context.Context, id string) error
	DeleteExpiredPasswordResets(ctx context.Context) error
}

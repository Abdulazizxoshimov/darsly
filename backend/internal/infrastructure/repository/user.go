package repository

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id string) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	List(ctx context.Context, filter *entity.UserFilter) ([]*entity.User, int, error)
	Update(ctx context.Context, user *entity.User) error
	UpdatePassword(ctx context.Context, userID, passwordHash string) error
	UpdateLastLogin(ctx context.Context, userID string) error
	SetDeletionRequested(ctx context.Context, userID string, requested bool) error
	SoftDelete(ctx context.Context, id string) error
	// DeleteHard qatorni butunlay o'chiradi (email'ni bo'shatadi) — registratsiya
	// yarim yo'lda uzilganda kompensatsiya uchun.
	DeleteHard(ctx context.Context, id string) error
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

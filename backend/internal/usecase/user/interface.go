package user

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UseCase interface {
	Create(ctx context.Context, req *entity.CreateUserReq) (*entity.User, error)
	GetByID(ctx context.Context, id string) (*entity.User, error)
	List(ctx context.Context, filter *entity.UserFilter) ([]*entity.User, int, error)
	Update(ctx context.Context, id string, req *entity.UpdateUserReq) (*entity.User, error)
	ChangePassword(ctx context.Context, id string, req *entity.ChangePasswordReq) error
	// ResetPassword — admin boshqa foydalanuvchi parolini joriy-parolsiz tiklaydi.
	// callerID/callerRole defence-in-depth uchun: target = o'z-o'zi YOKI chaqiruvchi admin.
	ResetPassword(ctx context.Context, callerID, callerRole, targetID, newPassword string) error
	Deactivate(ctx context.Context, id string) error
	Activate(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	// RequestDeletion — mentor o'z hisobini o'chirishni SO'RAYDI (o'zi o'chirmaydi).
	// Admin panelida belgi paydo bo'ladi; admin tasdiqlab `Delete` qiladi.
	RequestDeletion(ctx context.Context, userID string, requested bool) error
}

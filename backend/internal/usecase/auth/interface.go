package auth

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UseCase interface {
	Register(ctx context.Context, req *entity.RegisterReq, ip, userAgent string) (*entity.TokenPair, error)
	Login(ctx context.Context, req *entity.LoginReq, ip, userAgent string) (*entity.TokenPair, error)
	Refresh(ctx context.Context, req *entity.RefreshReq) (*entity.TokenPair, error)
	Logout(ctx context.Context, req *entity.LogoutReq) error
	ForgotPassword(ctx context.Context, req *entity.ForgotPasswordReq) error
	ResetPassword(ctx context.Context, req *entity.ResetPasswordReq) error
}

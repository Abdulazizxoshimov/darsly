package auth

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
)

// TxRunner — ko'p-yozuvli oqimni bitta tranzaksiyada bajaradi.
//
// `*storage.Storage` buni qondiradi (`RunInTx`). Usecase `storage` paketini
// ham, `pgx` ni ham BILMAYDI — u faqat "bu ikki yozuv birgalikda bajarilsin
// yoki umuman bajarilmasin" deydi.
//
// Nega kerak: ro'yxatdan o'tish `users` va `refresh_tokens` ga IKKI marta
// yozadi. Avval ikkinchi yozuv uzilsa birinchisi qo'lda o'chirilardi
// (kompensatsiya) — bu kompensatsiyaning o'zi uzilishi mumkin bo'lgan zaif
// joy edi. Tranzaksiyada rollback'ni DB kafolatlaydi.
type TxRunner interface {
	RunInTx(ctx context.Context, fn func(users repository.UserRepository, auth repository.AuthRepository) error) error
}

type UseCase interface {
	Register(ctx context.Context, req *entity.RegisterReq, ip, userAgent string) (*entity.TokenPair, error)
	Login(ctx context.Context, req *entity.LoginReq, ip, userAgent string) (*entity.TokenPair, error)
	Refresh(ctx context.Context, req *entity.RefreshReq) (*entity.TokenPair, error)
	Logout(ctx context.Context, req *entity.LogoutReq) error
	ForgotPassword(ctx context.Context, req *entity.ForgotPasswordReq) error
	ResetPassword(ctx context.Context, req *entity.ResetPasswordReq) error
}

package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"
	"github.com/zoom/darsly/internal/usecase/shared"
)

type useCase struct {
	repo   repository.UserRepository
	hasher hasher.Hasher
	tokens token.Maker
	log    logger.Logger
}

func New(repo repository.UserRepository, h hasher.Hasher, tokens token.Maker, log logger.Logger) UseCase {
	return &useCase{repo: repo, hasher: h, tokens: tokens, log: log}
}

// allowedRoles — ruxsat etilgan rollar allow-list'i. Noto'g'ri rol casbin policy'da
// topilmay foydalanuvchini butunlay bloklaydi (self-DoS) — shuning uchun usecase
// darajasida ham enum tekshiruvi (B3.3), handler validatsiyasidan mustaqil.
var allowedRoles = map[string]bool{
	"admin":   true,
	"mentor":  true,
	"student": true,
	"guest":   true,
}

func validRole(role string) bool { return allowedRoles[role] }

func (uc *useCase) Create(ctx context.Context, req *entity.CreateUserReq) (*entity.User, error) {
	hashed, err := uc.hasher.Hash(req.Password)
	if err != nil {
		uc.log.Error(ctx, "user.Create: hash failed", logger.SafeString("err", err.Error()))
		return nil, fmt.Errorf("user.Create hash: %w", err)
	}
	role := req.Role
	if role == "" {
		role = "student"
	}
	if !validRole(role) {
		return nil, apperr.BadRequest("invalid role")
	}
	u := &entity.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: hashed,
		FullName:     req.FullName,
		Color:        "#6366F1",
		Role:         role,
		// Mahsulot O'zbekiston uchun: yangi hisob darhol Toshkent vaqtida
		// bo'lsin. Avval "UTC" edi va ustoz profilda o'zi tuzatishi kerak edi —
		// tuzatmasa dars vaqtlari 5 soat surilib ko'rinardi.
		Timezone:     entity.DefaultTimezone,
		Language:     entity.DefaultLanguage,
		IsActive:     true,
	}
	if err := uc.repo.Create(ctx, u); err != nil {
		uc.log.Error(ctx, "user.Create: db error", logger.SafeEmail("email", req.Email), logger.SafeString("err", err.Error()))
		return nil, err
	}

	uc.log.Info(ctx, "user created", logger.String("id", u.ID))
	return u, nil
}

func (uc *useCase) GetByID(ctx context.Context, id string) (*entity.User, error) {
	// Yaroqsiz UUID Postgres'ga yetsa `22P02` xatosi 500 INTERNAL_ERROR bo'lib qaytadi
	// va Sentry'ga yozilardi (`GET /users/abc`). shared.ValidateID uni DB'ga bormasdan
	// 404 ga aylantiradi — mavjud bo'lmagan (lekin shakli to'g'ri) UUID javobi bilan bir xil.
	if err := shared.ValidateID(id, "user"); err != nil {
		return nil, err
	}
	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		uc.log.Debug(ctx, "user.GetByID: not found", logger.String("id", id))
		return nil, err
	}
	return u, nil
}

func (uc *useCase) List(ctx context.Context, filter *entity.UserFilter) ([]*entity.User, int, error) {
	return uc.repo.List(ctx, filter)
}

func (uc *useCase) Update(ctx context.Context, id string, req *entity.UpdateUserReq) (*entity.User, error) {
	// Yaroqsiz UUID Postgres'ga yetsa `22P02` xatosi 500 INTERNAL_ERROR bo'lib qaytadi
	// va Sentry'ga yozilardi (`GET /users/abc`). shared.ValidateID uni DB'ga bormasdan
	// 404 ga aylantiradi — mavjud bo'lmagan (lekin shakli to'g'ri) UUID javobi bilan bir xil.
	if err := shared.ValidateID(id, "user"); err != nil {
		return nil, err
	}
	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.FullName != nil {
		u.FullName = *req.FullName
	}
	if req.AvatarURL != nil {
		u.AvatarURL = req.AvatarURL
	}
	if req.Color != nil {
		u.Color = *req.Color
	}
	if req.Timezone != nil {
		u.Timezone = *req.Timezone
	}
	if req.Language != nil {
		u.Language = *req.Language
	}
	roleChanged := false
	if req.Role != nil {
		if !validRole(*req.Role) {
			return nil, apperr.BadRequest("invalid role")
		}
		roleChanged = u.Role != *req.Role
		u.Role = *req.Role
	}
	if err := uc.repo.Update(ctx, u); err != nil {
		uc.log.Error(ctx, "user.Update: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return nil, err
	}

	// ROL O'ZGARDI → barcha sessiyalar bekor qilinadi.
	//
	// Sabab: rol JWT `role` claim'ida yashaydi va `Rotate` uni ESKI token'dan
	// meros qiladi (token qatlami DB'ni bilmaydi — bu ataylab, u yerga repository
	// bog'lash qatlamlarni chalkashtirardi). Natijada adminlikdan olingan
	// foydalanuvchi refresh qilib admin claim'ini CHEKSIZ uzaytira olardi.
	// Sessiyani o'ldirish bu teshikni token qatlamiga tegmasdan yopadi: keyingi
	// kirishda claim DB'dagi haqiqiy roldan olinadi.
	//
	// Xato bo'lsa amal BEKOR QILINMAYDI (rol allaqachon DB'da), lekin bu
	// xavfsizlik hodisasi — Error darajasida yoziladi.
	if roleChanged && uc.tokens != nil {
		if err := uc.tokens.RevokeAllUserSessions(ctx, id); err != nil {
			uc.log.Error(ctx, "user.Update: rol o'zgardi, lekin sessiyalarni bekor qilib bo'lmadi",
				logger.String("id", id), logger.SafeString("err", err.Error()))
		} else {
			uc.log.Info(ctx, "user role changed — sessions revoked", logger.String("id", id))
		}
	}

	uc.log.Info(ctx, "user updated", logger.String("id", id))
	return u, nil
}

func (uc *useCase) ChangePassword(ctx context.Context, id string, req *entity.ChangePasswordReq) error {
	// Yaroqsiz UUID Postgres'ga yetsa `22P02` xatosi 500 INTERNAL_ERROR bo'lib qaytadi
	// va Sentry'ga yozilardi (`GET /users/abc`). shared.ValidateID uni DB'ga bormasdan
	// 404 ga aylantiradi — mavjud bo'lmagan (lekin shakli to'g'ri) UUID javobi bilan bir xil.
	if err := shared.ValidateID(id, "user"); err != nil {
		return err
	}
	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if !uc.hasher.Check(req.CurrentPassword, u.PasswordHash) {
		uc.log.Warn(ctx, "user.ChangePassword: wrong current password", logger.String("id", id))
		return apperr.BadRequest("current password is incorrect")
	}
	hashed, err := uc.hasher.Hash(req.NewPassword)
	if err != nil {
		uc.log.Error(ctx, "user.ChangePassword: hash failed", logger.SafeString("err", err.Error()))
		return fmt.Errorf("user.ChangePassword hash: %w", err)
	}
	if err := uc.repo.UpdatePassword(ctx, id, hashed); err != nil {
		uc.log.Error(ctx, "user.ChangePassword: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return err
	}
	uc.log.Info(ctx, "user password changed", logger.String("id", id))
	return nil
}

// ResetPassword — admin tomonidan boshqa foydalanuvchi parolini joriy-parolsiz
// tiklash. ChangePassword'dan farqi: joriy parol tekshirilmaydi (admin uni bilmaydi).
// Reset'dan so'ng maqsad foydalanuvchining barcha sessiyalari bekor qilinadi —
// eski access/refresh tokenlar bilan hech kim (jumladan ehtimoliy o'g'ri) kira olmaydi.
func (uc *useCase) ResetPassword(ctx context.Context, callerID, callerRole, targetID, newPassword string) error {
	if err := shared.ValidateID(targetID, "user"); err != nil {
		return err
	}
	// Defence-in-depth (RBAC bu route'ni admin'ga cheklaydi, lekin usecase ham enforce
	// qiladi): faqat admin YOKI o'z-o'ziga reset qila oladi.
	if callerRole != "admin" && callerID != targetID {
		uc.log.Warn(ctx, "user.ResetPassword: forbidden", logger.String("caller", callerID), logger.String("target", targetID))
		return apperr.Forbidden("access denied")
	}
	if _, err := uc.repo.GetByID(ctx, targetID); err != nil {
		return err
	}
	id := targetID
	hashed, err := uc.hasher.Hash(newPassword)
	if err != nil {
		uc.log.Error(ctx, "user.ResetPassword: hash failed", logger.SafeString("err", err.Error()))
		return fmt.Errorf("user.ResetPassword hash: %w", err)
	}
	if err := uc.repo.UpdatePassword(ctx, id, hashed); err != nil {
		uc.log.Error(ctx, "user.ResetPassword: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return err
	}
	if err := uc.tokens.RevokeAllUserSessions(ctx, id); err != nil {
		// Sessiya bekor qilish uzilsa ham parol allaqachon o'zgargan — kritik emas, faqat log.
		uc.log.Error(ctx, "user.ResetPassword: revoke sessions failed", logger.String("id", id), logger.SafeString("err", err.Error()))
	}
	uc.log.Info(ctx, "user password reset by admin", logger.String("id", id))
	return nil
}

func (uc *useCase) Deactivate(ctx context.Context, id string) error {
	// Yaroqsiz UUID Postgres'ga yetsa `22P02` xatosi 500 INTERNAL_ERROR bo'lib qaytadi
	// va Sentry'ga yozilardi (`GET /users/abc`). shared.ValidateID uni DB'ga bormasdan
	// 404 ga aylantiradi — mavjud bo'lmagan (lekin shakli to'g'ri) UUID javobi bilan bir xil.
	if err := shared.ValidateID(id, "user"); err != nil {
		return err
	}
	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	u.IsActive = false
	if err := uc.repo.Update(ctx, u); err != nil {
		uc.log.Error(ctx, "user.Deactivate: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return err
	}
	// Xavfsizlik: deaktivatsiya qilingan foydalanuvchining barcha faol sessiyalarini
	// darhol o'chiramiz — aks holda mavjud access/refresh token ishlashda davom etadi
	// (ValidateAccess/Rotate faqat Redis'ni tekshiradi).
	if uc.tokens != nil {
		if err := uc.tokens.RevokeAllUserSessions(ctx, id); err != nil {
			uc.log.Warn(ctx, "user.Deactivate: revoke sessions failed", logger.String("id", id), logger.SafeString("err", err.Error()))
		}
	}
	uc.log.Info(ctx, "user deactivated", logger.String("id", id))
	return nil
}

func (uc *useCase) Activate(ctx context.Context, id string) error {
	// Yaroqsiz UUID Postgres'ga yetsa `22P02` xatosi 500 INTERNAL_ERROR bo'lib qaytadi
	// va Sentry'ga yozilardi (`GET /users/abc`). shared.ValidateID uni DB'ga bormasdan
	// 404 ga aylantiradi — mavjud bo'lmagan (lekin shakli to'g'ri) UUID javobi bilan bir xil.
	if err := shared.ValidateID(id, "user"); err != nil {
		return err
	}
	u, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	u.IsActive = true
	if err := uc.repo.Update(ctx, u); err != nil {
		uc.log.Error(ctx, "user.Activate: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return err
	}
	uc.log.Info(ctx, "user activated", logger.String("id", id))
	return nil
}

func (uc *useCase) Delete(ctx context.Context, id string) error {
	// Yaroqsiz UUID Postgres'ga yetsa `22P02` xatosi 500 INTERNAL_ERROR bo'lib qaytadi
	// va Sentry'ga yozilardi (`GET /users/abc`). shared.ValidateID uni DB'ga bormasdan
	// 404 ga aylantiradi — mavjud bo'lmagan (lekin shakli to'g'ri) UUID javobi bilan bir xil.
	if err := shared.ValidateID(id, "user"); err != nil {
		return err
	}
	if err := uc.repo.SoftDelete(ctx, id); err != nil {
		uc.log.Error(ctx, "user.Delete: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return err
	}
	// O'chirilgan foydalanuvchining tokeni ishlashda davom etmasin: `ValidateAccess`
	// faqat Redis sessiyasini tekshiradi, DB'dagi `deleted_at` ni emas. Bu
	// `Deactivate` dagi bilan bir xil qoida.
	if uc.tokens != nil {
		if err := uc.tokens.RevokeAllUserSessions(ctx, id); err != nil {
			uc.log.Error(ctx, "user.Delete: sessiyalarni bekor qilib bo'lmadi",
				logger.String("id", id), logger.SafeString("err", err.Error()))
		}
	}
	uc.log.Info(ctx, "user deleted", logger.String("id", id))
	return nil
}

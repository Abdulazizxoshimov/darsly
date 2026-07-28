package auth

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	emailpkg "github.com/zoom/darsly/internal/infrastructure/email"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"
)

type useCase struct {
	userRepo    repository.UserRepository
	authRepo    repository.AuthRepository
	tokens      token.Maker
	hasher      hasher.Hasher
	resetTTL    time.Duration
	refreshTTL  time.Duration
	email       emailpkg.Sender
	frontendURL string
	log         logger.Logger
}

func New(
	userRepo repository.UserRepository,
	authRepo repository.AuthRepository,
	tokens token.Maker,
	h hasher.Hasher,
	resetTTL time.Duration,
	refreshTTL time.Duration,
	emailSender emailpkg.Sender,
	frontendURL string,
	log logger.Logger,
) UseCase {
	return &useCase{
		userRepo:    userRepo,
		authRepo:    authRepo,
		tokens:      tokens,
		hasher:      h,
		resetTTL:    resetTTL,
		refreshTTL:  refreshTTL,
		email:       emailSender,
		frontendURL: frontendURL,
		log:         log,
	}
}

func (uc *useCase) Register(ctx context.Context, req *entity.RegisterReq, ip, userAgent string) (*entity.TokenPair, error) {
	hashed, err := uc.hasher.Hash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("auth.Register hash: %w", err)
	}
	now := time.Now().UTC()
	u := &entity.User{
		ID:           uuid.NewString(),
		Email:        req.Email,
		PasswordHash: hashed,
		FullName:     req.FullName,
		Color:        "#6366F1",
		Role:         "student",
		// Mahsulot O'zbekiston uchun — `entity.DefaultTimezone` ga qarang.
		Timezone:     entity.DefaultTimezone,
		Language:     entity.DefaultLanguage,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := uc.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}

	// Keyingi qadamlardan biri uzilsa — yaratilgan user'ni tozalaymiz (email band qolmasin).
	cleanup := func(step string, cause error) (*entity.TokenPair, error) {
		if delErr := uc.userRepo.DeleteHard(ctx, u.ID); delErr != nil {
			uc.log.Error(ctx, "auth.Register: orphan user cleanup failed", logger.String("user_id", u.ID), logger.SafeString("err", delErr.Error()))
		}
		return nil, fmt.Errorf("auth.Register %s: %w", step, cause)
	}

	sessionID := uuid.NewString()
	access, refresh, err := uc.tokens.Generate(ctx, u.ID, sessionID, u.Role)
	if err != nil {
		return cleanup("generate tokens", err)
	}

	rt := &entity.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    u.ID,
		TokenHash: hashToken(refresh),
		ExpiresAt: now.Add(uc.refreshTTL),
		CreatedAt: now,
	}
	if ip != "" {
		rt.IPAddress = &ip
	}
	if userAgent != "" {
		rt.UserAgent = &userAgent
	}
	if err := uc.authRepo.CreateRefreshToken(ctx, rt); err != nil {
		return cleanup("store token", err)
	}
	if err := uc.tokens.StoreSession(ctx, sessionID, u.ID, uc.refreshTTL); err != nil {
		return cleanup("store session", err)
	}

	uc.log.Info(ctx, "auth.Register: success", logger.String("user_id", u.ID))
	return &entity.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (uc *useCase) Login(ctx context.Context, req *entity.LoginReq, ip, userAgent string) (*entity.TokenPair, error) {
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		uc.log.Warn(ctx, "auth.Login: user not found", logger.SafeEmail("email", req.Email))
		// Timing hujumidan himoya: user topilmasa ham bcrypt taqqoslashni bajaramiz,
		// shunda "user bor/yo'q" javob vaqti farqidan bilib bo'lmaydi (enumeration).
		uc.hasher.Check(req.Password, dummyBcryptHash)
		return nil, apperr.Unauthorized("invalid credentials")
	}
	if !user.IsActive {
		uc.log.Warn(ctx, "auth.Login: account deactivated", logger.String("user_id", user.ID))
		return nil, apperr.Forbidden("account is deactivated")
	}
	if !uc.hasher.Check(req.Password, user.PasswordHash) {
		uc.log.Warn(ctx, "auth.Login: wrong password", logger.String("user_id", user.ID))
		return nil, apperr.Unauthorized("invalid credentials")
	}

	sessionID := uuid.NewString()
	access, refresh, err := uc.tokens.Generate(ctx, user.ID, sessionID, user.Role)
	if err != nil {
		uc.log.Error(ctx, "auth.Login: generate tokens failed", logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
		return nil, fmt.Errorf("auth.Login generate tokens: %w", err)
	}

	now := time.Now().UTC()
	rt := &entity.RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: hashToken(refresh),
		ExpiresAt: now.Add(uc.refreshTTL),
		CreatedAt: now,
	}
	if ip != "" {
		rt.IPAddress = &ip
	}
	if userAgent != "" {
		rt.UserAgent = &userAgent
	}
	if err := uc.authRepo.CreateRefreshToken(ctx, rt); err != nil {
		uc.log.Error(ctx, "auth.Login: store refresh token failed", logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
		return nil, fmt.Errorf("auth.Login store token: %w", err)
	}
	if err := uc.tokens.StoreSession(ctx, sessionID, user.ID, uc.refreshTTL); err != nil {
		uc.log.Error(ctx, "auth.Login: store session failed", logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
		return nil, fmt.Errorf("auth.Login store session: %w", err)
	}

	uc.log.Info(ctx, "auth.Login: success", logger.String("user_id", user.ID))
	return &entity.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (uc *useCase) Refresh(ctx context.Context, req *entity.RefreshReq) (*entity.TokenPair, error) {
	access, refresh, err := uc.tokens.Rotate(ctx, req.RefreshToken)
	if err != nil {
		uc.log.Warn(ctx, "auth.Refresh: invalid token")
		return nil, apperr.Unauthorized("invalid or expired refresh token")
	}
	return &entity.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (uc *useCase) Logout(ctx context.Context, req *entity.LogoutReq) error {
	// Redis'dagi sessiya (access) va refresh JTI'sini bekor qilish — refresh
	// token'dan sid/jti olib. Bu access token'ni ham darhol yaroqsiz qiladi.
	if err := uc.tokens.RevokeRefresh(ctx, req.RefreshToken); err != nil {
		uc.log.Warn(ctx, "auth.Logout: token revoke failed", logger.SafeString("err", err.Error()))
	}
	// DB'dagi refresh token qatorini ham bekor qilish (audit/tarix uchun).
	if rt, err := uc.authRepo.GetRefreshTokenByHash(ctx, hashToken(req.RefreshToken)); err == nil {
		if err := uc.authRepo.RevokeRefreshToken(ctx, rt.ID); err != nil {
			uc.log.Warn(ctx, "auth.Logout: db revoke failed", logger.String("user_id", rt.UserID), logger.SafeString("err", err.Error()))
		}
		uc.log.Info(ctx, "auth.Logout: success", logger.String("user_id", rt.UserID))
	}
	return nil
}

func (uc *useCase) ForgotPassword(ctx context.Context, req *entity.ForgotPasswordReq) error {
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	rawToken := uuid.NewString()
	h := hashToken(rawToken)
	now := time.Now().UTC()
	pr := &entity.PasswordReset{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: h,
		ExpiresAt: now.Add(uc.resetTTL),
		CreatedAt: now,
	}
	if err := uc.authRepo.CreatePasswordReset(ctx, pr); err != nil {
		uc.log.Error(ctx, "auth.ForgotPassword: store reset token failed", logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
		return fmt.Errorf("auth.ForgotPassword: %w", err)
	}

	resetURL := uc.frontendURL + "/reset-password?token=" + rawToken
	if err := uc.email.Send(ctx, []string{user.Email}, "Parolni tiklash", "password_reset", map[string]string{
		"ResetURL": resetURL,
	}); err != nil {
		uc.log.Error(ctx, "auth.ForgotPassword: send email failed", logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
		// email xatosi foydalanuvchiga qaytarilmaydi (timing attack oldini olish uchun)
	}

	uc.log.Info(ctx, "auth.ForgotPassword: reset email sent", logger.String("user_id", user.ID))
	return nil
}

func (uc *useCase) ResetPassword(ctx context.Context, req *entity.ResetPasswordReq) error {
	h := hashToken(req.Token)
	pr, err := uc.authRepo.GetPasswordResetByHash(ctx, h)
	if err != nil {
		return apperr.BadRequest("invalid or expired reset token")
	}

	hashed, err := uc.hasher.Hash(req.NewPassword)
	if err != nil {
		uc.log.Error(ctx, "auth.ResetPassword: hash failed", logger.SafeString("err", err.Error()))
		return fmt.Errorf("auth.ResetPassword hash: %w", err)
	}
	if err := uc.userRepo.UpdatePassword(ctx, pr.UserID, hashed); err != nil {
		uc.log.Error(ctx, "auth.ResetPassword: update failed", logger.String("user_id", pr.UserID), logger.SafeString("err", err.Error()))
		return fmt.Errorf("auth.ResetPassword update: %w", err)
	}

	// Xavfsizlik: parol tiklangach barcha mavjud sessiyalarni bekor qilamiz
	// (o'g'irlangan sessiya na access ishlata olsin, na refresh rotatsiya qilsin).
	// 1) DB refresh_tokens qatorlari (audit/tarix); 2) Redis sessiya+jti kalitlari —
	// ValidateAccess/Rotate faqat Redis'ni tekshiradi, DB'ni emas, shuning uchun
	// Redis'ni tozalamasak o'g'irlangan sessiya tirik qoladi.
	if err := uc.authRepo.RevokeAllUserTokens(ctx, pr.UserID); err != nil {
		uc.log.Warn(ctx, "auth.ResetPassword: revoke db tokens failed", logger.String("user_id", pr.UserID), logger.SafeString("err", err.Error()))
	}
	if err := uc.tokens.RevokeAllUserSessions(ctx, pr.UserID); err != nil {
		uc.log.Warn(ctx, "auth.ResetPassword: revoke redis sessions failed", logger.String("user_id", pr.UserID), logger.SafeString("err", err.Error()))
	}

	uc.log.Info(ctx, "auth.ResetPassword: success", logger.String("user_id", pr.UserID))
	return uc.authRepo.MarkPasswordResetUsed(ctx, pr.ID)
}

// dummyBcryptHash — mavjud bo'lmagan foydalanuvchi uchun timing-himoya:
// hasher.Check shu yaroqli bcrypt hash bilan haqiqiy taqqoslash vaqtini sarflaydi.
const dummyBcryptHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEeO.iI1jOZ0mSDfyz.9vI0lQ2sMOr0aWTa"

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return fmt.Sprintf("%x", sum)
}

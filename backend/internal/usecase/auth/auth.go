package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	emailpkg "github.com/zoom/darsly/internal/infrastructure/email"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"
)

const (
	// B4 — hisob darajasidagi login-lockout (IP'dan MUSTAQIL).
	//
	// Nega IP yetmaydi: IP rate-limit (middleware) bitta IP'ni to'xtatadi, lekin
	// taqsimlangan hujum (botnet, ko'p IP) bitta hisobga cheksiz parol sinaydi.
	// Qulf (email + IP) JUFTLIGI bo'yicha: bitta IP'dan hujum baribir to'siladi,
	// lekin hujumchi qurbonning emailini BOSHQA foydalanuvchilar uchun qulflay
	// olmaydi (klassik account-lockout DoS yopilgan — qurbon o'z IP'sidan kira
	// oladi). Taqsimlangan (ko'p IP) hujumni IP rate-limit middleware'i va
	// har-IP 5 urinish chegarasi sekinlashtiradi.
	//
	// Hisoblagich FAQAT MAVJUD hisoblar uchun oshiriladi — soxta email spam'i
	// Redis'ni to'ldirmaydi.
	maxLoginAttempts  = 5
	loginFailWindow   = 15 * time.Minute
	loginLockCooldown = 15 * time.Minute

	// forgot-password per-email throttle: oynada ko'pi bilan shuncha email yuboriladi.
	maxForgotPerWindow = 3
	forgotWindow       = 15 * time.Minute
)

type useCase struct {
	userRepo    repository.UserRepository
	authRepo    repository.AuthRepository
	tokens      token.Maker
	hasher      hasher.Hasher
	cache       redis.Cache
	resetTTL    time.Duration
	refreshTTL  time.Duration
	email       emailpkg.Sender
	frontendURL string
	// tx — ko'p-yozuvli oqim uchun (ixtiyoriy; nil bo'lsa kompensatsiya yo'li).
	tx  TxRunner
	log logger.Logger
	// dummyHash — configlangan bcrypt cost bilan startup'da yasalgan soxta hash
	// (mavjud bo'lmagan user / qulf uchun timing-himoya; cost haqiqiy hashlarniki bilan teng).
	dummyHash string
}

func New(
	userRepo repository.UserRepository,
	authRepo repository.AuthRepository,
	tokens token.Maker,
	h hasher.Hasher,
	cache redis.Cache,
	resetTTL time.Duration,
	refreshTTL time.Duration,
	emailSender emailpkg.Sender,
	frontendURL string,
	tx TxRunner,
	log logger.Logger,
) UseCase {
	// Startup'da bir marta: hasher'ning O'Z cost'i bilan (const hash cost 12 edi,
	// default BCRYPT_COST=11 — vaqt farqi enumeration oracle bo'lardi).
	dummy, err := h.Hash("darsly-timing-dummy-password")
	if err != nil {
		dummy = dummyBcryptHash // zaxira (kutilmagan)
	}
	return &useCase{
		dummyHash:   dummy,
		userRepo:    userRepo,
		authRepo:    authRepo,
		tokens:      tokens,
		hasher:      h,
		cache:       cache,
		resetTTL:    resetTTL,
		refreshTTL:  refreshTTL,
		email:       emailSender,
		frontendURL: frontendURL,
		tx:          tx,
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
		Timezone:  entity.DefaultTimezone,
		Language:  entity.DefaultLanguage,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	// TARTIB MUHIM: tokenlar DB yozuvidan OLDIN generatsiya qilinadi.
	//
	// Ular `u.ID` va `sessionID` dan boshqa hech narsaga bog'liq emas (ikkalasi
	// ham shu yerda yasaladi), ya'ni user qatori hali bo'lmasa ham ishlaydi.
	// Buning sababi: `Generate` — REDIS chaqiruvi, va uni tranzaksiya ichida
	// bajarish mumkin emas edi (ochiq tranzaksiya tashqi servis javobini kutib
	// turishi poolni band qiladi). Endi ketma-ketlik: Redis → tranzaksiya → Redis.
	sessionID := uuid.NewString()
	access, refresh, err := uc.tokens.Generate(ctx, u.ID, sessionID, u.Role)
	if err != nil {
		return nil, fmt.Errorf("auth.Register generate tokens: %w", err)
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
	// ⭐ IKKI YOZUV — BITTA TRANZAKSIYA.
	//
	// Avval `users` alohida yozilar, keyin `refresh_tokens` yozilar va ikkinchisi
	// uzilsa birinchisi qo'lda `DeleteHard` bilan o'chirilardi. Bu naqshning
	// zaif joyi: kompensatsiyaning O'ZI uzilishi mumkin (DB shu payt yiqilgan
	// bo'lsa) — natijada yetim user qoladi va email abadiy band bo'lib turadi,
	// foydalanuvchi esa qayta ro'yxatdan o'ta olmaydi.
	//
	// Tranzaksiyada bu sinf muammosi yo'q: rollback'ni DB kafolatlaydi.
	if uc.tx != nil {
		err = uc.tx.RunInTx(ctx, func(users repository.UserRepository, auth repository.AuthRepository) error {
			if err := users.Create(ctx, u); err != nil {
				return err
			}
			return auth.CreateRefreshToken(ctx, rt)
		})
	} else {
		// Tranzaksiya beruvchi ulanmagan (masalan birlik testi) — eski
		// kompensatsiya yo'li. Xulq bir xil, kafolat kuchsizroq.
		err = uc.createUserAndTokenCompensating(ctx, u, rt)
	}
	if err != nil {
		return nil, fmt.Errorf("auth.Register persist: %w", err)
	}

	// Sessiya Redis'da — tranzaksiyadan KEYIN (tashqi servis tranzaksiya ichida
	// bo'lmasligi kerak). Bu uzilsa DB'da yozuv qoladi, lekin u zararsiz:
	// foydalanuvchi mavjud va qayta login qilib yangi sessiya ochadi. Yetim
	// user'dan farqli, bu holat foydalanuvchini BLOKLAMAYDI.
	if err := uc.tokens.StoreSession(ctx, sessionID, u.ID, uc.refreshTTL); err != nil {
		uc.log.Error(ctx, "auth.Register: sessiya saqlanmadi — qayta login kerak bo'ladi",
			logger.String("user_id", u.ID), logger.SafeString("err", err.Error()))
		return nil, fmt.Errorf("auth.Register store session: %w", err)
	}

	uc.log.Info(ctx, "auth.Register: success", logger.String("user_id", u.ID))
	return &entity.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// createUserAndTokenCompensating — tranzaksiyasiz zaxira yo'li (`TxRunner` nil).
// Ikkinchi yozuv uzilsa birinchisini tozalaydi.
func (uc *useCase) createUserAndTokenCompensating(
	ctx context.Context, u *entity.User, rt *entity.RefreshToken,
) error {
	if err := uc.userRepo.Create(ctx, u); err != nil {
		return err
	}
	if err := uc.authRepo.CreateRefreshToken(ctx, rt); err != nil {
		if delErr := uc.userRepo.DeleteHard(ctx, u.ID); delErr != nil {
			uc.log.Error(ctx, "auth.Register: orphan user cleanup failed",
				logger.String("user_id", u.ID), logger.SafeString("err", delErr.Error()))
		}
		return err
	}
	return nil
}

func (uc *useCase) Login(ctx context.Context, req *entity.LoginReq, ip, userAgent string) (*entity.TokenPair, error) {
	emailKey := normalizeEmail(req.Email)
	// Qulf kaliti (email+IP) — qurbon emailini begona IP'dan qulflab bo'lmaydi.
	lockID := emailKey + "|" + ip

	// B4 — hisob qulflanganmi? Parol tekshiruvidan OLDIN. `cache` nil bo'lsa
	// (test yoki Redis'siz konfiguratsiya) lockout o'chadi — login ishlayveradi.
	if uc.cache != nil {
		if locked, _ := uc.cache.Get(ctx, loginLockKey(lockID)); locked != "" {
			uc.log.Warn(ctx, "auth.Login: account locked (brute-force)", logger.SafeEmail("email", req.Email))
			// Timing/enumeration himoyasi: qulflangan javobda ham bcrypt vaqtini sarflaymiz.
			uc.hasher.Check(req.Password, uc.dummyHash)
			return nil, apperr.Unauthorized("too many failed attempts, try again later")
		}
	}

	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		uc.log.Warn(ctx, "auth.Login: user not found", logger.SafeEmail("email", req.Email))
		// Timing hujumidan himoya: user topilmasa ham bcrypt taqqoslashni bajaramiz,
		// shunda "user bor/yo'q" javob vaqti farqidan bilib bo'lmaydi (enumeration).
		// Hisoblagich ATAYLAB oshirilmaydi: mavjud bo'lmagan email soxta bo'lishi
		// mumkin (Redis spam) — qulf faqat haqiqiy hisoblar uchun.
		uc.hasher.Check(req.Password, uc.dummyHash)
		return nil, apperr.Unauthorized("invalid credentials")
	}
	// TARTIB: avval PAROL, keyin faollik. Aks holda noto'g'ri parol bilan ham
	// "account is deactivated" (403) javobi deaktiv akkaunt mavjudligini oshkor qilardi.
	if !uc.hasher.Check(req.Password, user.PasswordHash) {
		uc.log.Warn(ctx, "auth.Login: wrong password", logger.String("user_id", user.ID))
		uc.registerFailedLogin(ctx, lockID, req.Email)
		return nil, apperr.Unauthorized("invalid credentials")
	}
	if !user.IsActive {
		uc.log.Warn(ctx, "auth.Login: account deactivated", logger.String("user_id", user.ID))
		return nil, apperr.Forbidden("account is deactivated")
	}
	// Muvaffaqiyat — hisoblagich va qulfni tozalaymiz.
	if uc.cache != nil {
		_ = uc.cache.Del(ctx, loginFailKey(lockID), loginLockKey(lockID))
	}

	sessionID := uuid.NewString()

	// ⭐ BITTA AKKAUNT = BITTA FAOL SESSIYA (PRODUCT.md, akkaunt ulashishga qarshi).
	//
	// Yangi login eskilarini tugatadi: eski qurilma keyingi so'rovda 401 +
	// `SESSION_REVOKED` oladi va login ekraniga qaytadi.
	//
	// TARTIB MUHIM — tozalash tokenlar YARATILISHIDAN OLDIN. Teskarisi bo'lsa
	// (avval yarat, keyin tozala) parallel ikki login bir-birining yangi
	// sessiyasini o'chirib, ikkalasi ham chiqib ketardi. Bu tartibda esa
	// natija aniq: OXIRGI login g'olib.
	//
	// Refresh JTI'lariga ATAYLAB tegilmaydi — sababi
	// `token.JWTMaker.RevokeUserSessionsExcept` izohida (reuse-detektor yangi
	// sessiyani o'ldirib yubormasligi uchun).
	if err := uc.tokens.RevokeUserSessionsExcept(ctx, user.ID, sessionID); err != nil {
		// Tozalash uzilsa login'ni BLOKLAMAYMIZ: foydalanuvchini o'z akkauntidan
		// to'sish zarari, eski sessiyaning bir muddat tirik qolishidan katta.
		uc.log.Warn(ctx, "auth.Login: eski sessiyalarni tugatib bo'lmadi",
			logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
	}
	// DB'dagi eski refresh qatorlari ham bekor qilinadi (audit/tarix izchilligi).
	if err := uc.authRepo.RevokeAllUserTokens(ctx, user.ID); err != nil {
		uc.log.Warn(ctx, "auth.Login: eski DB refresh qatorlarini bekor qilib bo'lmadi",
			logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
	}

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

	// Oxirgi kirish vaqti — best-effort (xato login'ni to'smaydi).
	if err := uc.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		uc.log.Warn(ctx, "auth.Login: last_login_at yozilmadi",
			logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
	}

	uc.log.Info(ctx, "auth.Login: success", logger.String("user_id", user.ID))
	return &entity.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (uc *useCase) Refresh(ctx context.Context, req *entity.RefreshReq) (*entity.TokenPair, error) {
	access, refresh, err := uc.tokens.Rotate(ctx, req.RefreshToken)
	if err != nil {
		// Sessiya server tomonidan tugatilgan (boshqa qurilmada kirish, logout,
		// parol tiklash) — buni oddiy "token buzuq" dan ajratamiz, shunda klient
		// «Boshqa qurilmada kirildi» deb aniq xabar bera oladi.
		if errors.Is(err, token.ErrSessionRevoked) {
			uc.log.Info(ctx, "auth.Refresh: sessiya tugatilgan (boshqa qurilmada kirilgan bo'lishi mumkin)")
			return nil, apperr.SessionRevoked("session ended: signed in on another device")
		}
		uc.log.Warn(ctx, "auth.Refresh: invalid token")
		return nil, apperr.Unauthorized("invalid or expired refresh token")
	}
	return &entity.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

func (uc *useCase) Logout(ctx context.Context, userID, sessionID string, req *entity.LogoutReq) error {
	refresh := ""
	if req != nil {
		refresh = req.RefreshToken
	}
	if sessionID == "" && refresh == "" {
		return apperr.BadRequest("session or refresh_token required")
	}

	// Joriy (access token'dagi) sessiya HAR DOIM bekor qilinadi — refresh token
	// yuborilmasa ham (avval klient 204 olardi, sessiya esa tirik qolardi).
	if sessionID != "" {
		if err := uc.tokens.RevokeSession(ctx, sessionID); err != nil {
			uc.log.Warn(ctx, "auth.Logout: session revoke failed", logger.SafeString("err", err.Error()))
		}
	}

	if refresh == "" {
		uc.log.Info(ctx, "auth.Logout: success (session only)", logger.String("user_id", userID))
		return nil
	}

	// Refresh token CHAQIRUVCHINIKI bo'lishi shart: boshqa foydalanuvchining
	// (o'g'irlangan/taxminiy) tokeni bilan uning sessiyasini o'chirib bo'lmasin.
	if sid, err := uc.tokens.SessionFromRefresh(refresh); err == nil && sessionID != "" && sid != sessionID {
		uc.log.Warn(ctx, "auth.Logout: refresh token boshqa sessiyaga tegishli — o'tkazib yuborildi",
			logger.String("user_id", userID))
		return nil
	}
	rt, rtErr := uc.authRepo.GetRefreshTokenByHash(ctx, hashToken(refresh))
	if rtErr == nil && userID != "" && rt.UserID != userID {
		uc.log.Warn(ctx, "auth.Logout: refresh token boshqa foydalanuvchiniki — o'tkazib yuborildi",
			logger.String("user_id", userID))
		return nil
	}

	// Redis'dagi refresh JTI (va sessiya) — refresh token'dan sid/jti olib.
	if err := uc.tokens.RevokeRefresh(ctx, refresh); err != nil {
		uc.log.Warn(ctx, "auth.Logout: token revoke failed", logger.SafeString("err", err.Error()))
	}
	// DB'dagi refresh token qatorini ham bekor qilish (audit/tarix uchun).
	if rtErr == nil {
		if err := uc.authRepo.RevokeRefreshToken(ctx, rt.ID); err != nil {
			uc.log.Warn(ctx, "auth.Logout: db revoke failed", logger.String("user_id", rt.UserID), logger.SafeString("err", err.Error()))
		}
		uc.log.Info(ctx, "auth.Logout: success", logger.String("user_id", rt.UserID))
	}
	return nil
}

func (uc *useCase) ForgotPassword(ctx context.Context, req *entity.ForgotPasswordReq) error {
	// Per-email throttle: oshsa JIMGINA nil (email yubormasdan) — javob har doim
	// bir xil 204, shuning uchun enumeration/timing buzilmaydi. Email spam va
	// reset-token ko'payishini to'xtatadi. Cache nil bo'lsa o'chadi.
	if uc.cache != nil {
		if n, err := uc.cache.Incr(ctx, forgotKey(normalizeEmail(req.Email)), forgotWindow); err == nil && n > maxForgotPerWindow {
			uc.log.Warn(ctx, "auth.ForgotPassword: throttled", logger.SafeEmail("email", req.Email))
			return nil
		}
	}
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil
	}

	// Yangi token berishdan oldin eskilarini bekor qilamiz: bir vaqtda faqat
	// BITTA yaroqli reset havolasi bo'lsin (eski xatlar/skrinshotlar yaroqsiz).
	if err := uc.authRepo.InvalidateUserPasswordResets(ctx, user.ID); err != nil {
		uc.log.Warn(ctx, "auth.ForgotPassword: eski reset tokenlarni bekor qilib bo'lmadi",
			logger.String("user_id", user.ID), logger.SafeString("err", err.Error()))
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

// dummyBcryptHash — faqat hasher.Hash xato bersa zaxira.
const dummyBcryptHash = "$2a$12$C6UzMDM.H6dfI/f/IKcEeO.iI1jOZ0mSDfyz.9vI0lQ2sMOr0aWTa"

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return fmt.Sprintf("%x", sum)
}

// registerFailedLogin — noto'g'ri parol hisoblagichini oshiradi va chegaraga
// yetganda hisobni qulflaydi (B4). Faqat MAVJUD hisoblar uchun chaqiriladi.
func (uc *useCase) registerFailedLogin(ctx context.Context, lockID, rawEmail string) {
	if uc.cache == nil {
		return
	}
	n, err := uc.cache.Incr(ctx, loginFailKey(lockID), loginFailWindow)
	if err != nil {
		return
	}
	if n >= maxLoginAttempts {
		_ = uc.cache.Set(ctx, loginLockKey(lockID), "1", loginLockCooldown)
		uc.log.Warn(ctx, "auth.Login: account locked after too many failed attempts",
			logger.SafeEmail("email", rawEmail))
	}
}

// normalizeEmail — qulf kalitini "Case" bilan chetlab o'tishning oldini oladi.
func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }

// id — "email|ip" juftligi (Login'da yig'iladi).
func loginFailKey(id string) string { return "loginfail:" + id }
func loginLockKey(id string) string { return "loginlock:" + id }
func forgotKey(email string) string { return "forgotpw:" + email }

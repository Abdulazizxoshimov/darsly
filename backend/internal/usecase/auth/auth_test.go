package auth_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/email"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/testutil"
	"github.com/zoom/darsly/internal/usecase/auth"
)

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func newAuthUC(users *testutil.FakeUserRepo, auths *testutil.FakeAuthRepo, tokens *testutil.FakeTokenMaker) auth.UseCase {
	return auth.New(users, auths, tokens, hasher.New(4), time.Hour, 720*time.Hour, email.NewNopSender(), "http://frontend", testutil.NewLogger())
}

func TestRegister_Success(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)

	pair, err := uc.Register(context.Background(), &entity.RegisterReq{
		FullName: "Dilnoza", Email: "d@darsly.uz", Password: "parol12345",
	}, "1.2.3.4", "agent")
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)

	u, err := users.GetByEmail(context.Background(), "d@darsly.uz")
	require.NoError(t, err)
	require.Equal(t, "student", u.Role, "yangi user default 'student' bo'lishi kerak")
	require.Equal(t, entity.DefaultLanguage, u.Language)
	// Mahsulot O'zbekiston uchun: default `UTC` emas, Toshkent. Aks holda
	// yangi ustozning dars vaqtlari 5 soat surilib ko'rinardi.
	require.Equal(t, "Asia/Tashkent", u.Timezone)
	require.True(t, u.IsActive)
	require.NotEqual(t, "parol12345", u.PasswordHash, "parol hash qilingan bo'lishi kerak")
	require.Len(t, tokens.Sessions, 1, "sessiya saqlanishi kerak")
}

func TestRegister_DuplicateEmail(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)
	req := &entity.RegisterReq{FullName: "A", Email: "dup@darsly.uz", Password: "parol12345"}

	_, err := uc.Register(context.Background(), req, "", "")
	require.NoError(t, err)
	_, err = uc.Register(context.Background(), req, "", "")
	require.Error(t, err)
	require.True(t, apperr.IsConflict(err), "takroriy email conflict qaytarishi kerak")
}

// Register yarim yo'lda uzilsa (refresh token yozib bo'lmasa) yaratilgan user tozalanishi kerak.
func TestRegister_OrphanCleanup(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	auths.FailNextCreateToken = true
	uc := newAuthUC(users, auths, tokens)

	_, err := uc.Register(context.Background(), &entity.RegisterReq{
		FullName: "Orphan", Email: "orphan@darsly.uz", Password: "parol12345",
	}, "", "")
	require.Error(t, err)
	require.GreaterOrEqual(t, users.Calls["DeleteHard"], 1, "yetim user hard-delete qilinishi kerak")

	_, err = users.GetByEmail(context.Background(), "orphan@darsly.uz")
	require.Error(t, err, "email bo'shatilishi kerak (qayta ro'yxatdan o'tish mumkin)")
}

func TestLogin_WrongPassword(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)
	_, err := uc.Register(context.Background(), &entity.RegisterReq{FullName: "A", Email: "l@darsly.uz", Password: "parol12345"}, "", "")
	require.NoError(t, err)

	_, err = uc.Login(context.Background(), &entity.LoginReq{Email: "l@darsly.uz", Password: "notright"}, "", "")
	require.Error(t, err)
	require.True(t, errors.Is(err, apperr.Unauthorized("")), "noto'g'ri parol 401 qaytarishi kerak")
}

// Mavjud bo'lmagan user ham xatosiz (panic'siz) Unauthorized qaytarishi kerak (timing himoyasi).
func TestLogin_UnknownUser(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)
	_, err := uc.Login(context.Background(), &entity.LoginReq{Email: "nope@darsly.uz", Password: "whatever"}, "", "")
	require.Error(t, err)
	require.True(t, errors.Is(err, apperr.Unauthorized("")))
}

func TestLogin_Deactivated(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)
	_, err := uc.Register(context.Background(), &entity.RegisterReq{FullName: "A", Email: "de@darsly.uz", Password: "parol12345"}, "", "")
	require.NoError(t, err)
	u, _ := users.GetByEmail(context.Background(), "de@darsly.uz")
	u.IsActive = false
	require.NoError(t, users.Update(context.Background(), u))

	_, err = uc.Login(context.Background(), &entity.LoginReq{Email: "de@darsly.uz", Password: "parol12345"}, "", "")
	require.Error(t, err)
	require.True(t, apperr.IsForbidden(err), "deaktivatsiya qilingan user 403 olishi kerak")
}

func TestLogout_RevokesToken(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)
	pair, err := uc.Register(context.Background(), &entity.RegisterReq{FullName: "A", Email: "lo@darsly.uz", Password: "parol12345"}, "", "")
	require.NoError(t, err)

	require.NoError(t, uc.Logout(context.Background(), &entity.LogoutReq{RefreshToken: pair.RefreshToken}))
	require.Contains(t, tokens.Revoked, pair.RefreshToken, "logout refresh token'ni Redis'dan bekor qilishi kerak")
	require.GreaterOrEqual(t, auths.Calls["RevokeRefreshToken"], 1, "DB refresh token ham bekor qilinishi kerak")
}

func TestRefresh(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)
	pair, err := uc.Refresh(context.Background(), &entity.RefreshReq{RefreshToken: "some-refresh"})
	require.NoError(t, err)
	require.NotEmpty(t, pair.AccessToken)
	require.NotEmpty(t, pair.RefreshToken)
}

func TestResetPassword(t *testing.T) {
	users, auths, tokens := testutil.NewFakeUserRepo(), testutil.NewFakeAuthRepo(), testutil.NewFakeTokenMaker()
	uc := newAuthUC(users, auths, tokens)
	_, err := uc.Register(context.Background(), &entity.RegisterReq{FullName: "A", Email: "rp@darsly.uz", Password: "oldparol12"}, "", "")
	require.NoError(t, err)
	u, _ := users.GetByEmail(context.Background(), "rp@darsly.uz")
	oldHash := u.PasswordHash

	// Reset token'ni to'g'ridan-to'g'ri joylaymiz (odatda ForgotPassword email orqali yuboradi).
	rawToken := "reset-secret-token"
	require.NoError(t, auths.CreatePasswordReset(context.Background(), &entity.PasswordReset{
		ID: "pr1", UserID: u.ID, TokenHash: sha256hex(rawToken), ExpiresAt: time.Now().Add(time.Hour),
	}))

	require.NoError(t, uc.ResetPassword(context.Background(), &entity.ResetPasswordReq{
		Token: rawToken, NewPassword: "yangiparol123",
	}))

	u2, _ := users.GetByEmail(context.Background(), "rp@darsly.uz")
	require.NotEqual(t, oldHash, u2.PasswordHash, "parol o'zgargan bo'lishi kerak")
	require.GreaterOrEqual(t, auths.Calls["RevokeAllUserTokens"], 1, "parol tiklangach DB refresh token'lar bekor qilinishi kerak")
	require.Contains(t, tokens.RevokedUsers, u.ID, "parol tiklangach Redis sessiyalari ham bekor qilinishi kerak")

	// Noto'g'ri token → xato.
	require.Error(t, uc.ResetPassword(context.Background(), &entity.ResetPasswordReq{Token: "wrong", NewPassword: "x123456789"}))
}

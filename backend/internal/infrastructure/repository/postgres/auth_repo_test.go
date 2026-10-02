package postgres_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/entity"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/testutil"
)

// B-2: auth.go repozitoriysida umuman DB testi yo'q edi. Quyidagilar HAQIQIY
// SQL'da tekshiradi — soxta repo bu shartlarni bajarmaydi.

// GetPasswordResetByHash `used_at IS NULL AND expires_at > NOW()` shartini
// HURMAT qilishi kerak. Bu test ushlaydigan xato: agar shart tushib qolsa,
// ishlatilgan yoki muddati o'tgan reset token orqali parol qayta o'rnatilardi.
func TestAuthRepo_GetPasswordResetByHash_Filters(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	auth := pgRepo.NewAuthRepo(pg)
	u := makeUser(t, users, "pr_get@darsly.uz")

	// 1) Yaroqli (ishlatilmagan + muddati o'tmagan) → qaytadi.
	valid := &entity.PasswordReset{
		ID: uuid.NewString(), UserID: u.ID, TokenHash: "pr-valid-hash",
		ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, auth.CreatePasswordReset(ctx(), valid))
	got, err := auth.GetPasswordResetByHash(ctx(), "pr-valid-hash")
	require.NoError(t, err)
	require.Equal(t, valid.ID, got.ID)

	// 2) Ishlatilgan token → qaytmasligi kerak (used_at IS NULL sharti).
	used := &entity.PasswordReset{
		ID: uuid.NewString(), UserID: u.ID, TokenHash: "pr-used-hash",
		ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, auth.CreatePasswordReset(ctx(), used))
	require.NoError(t, auth.MarkPasswordResetUsed(ctx(), used.ID))
	_, err = auth.GetPasswordResetByHash(ctx(), "pr-used-hash")
	require.True(t, apperr.IsNotFound(err), "ishlatilgan reset token qaytarilmasligi kerak")

	// 3) Muddati o'tgan token → qaytmasligi kerak (expires_at > NOW() sharti).
	expired := &entity.PasswordReset{
		ID: uuid.NewString(), UserID: u.ID, TokenHash: "pr-expired-hash",
		ExpiresAt: time.Now().UTC().Add(-time.Hour), CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
	}
	require.NoError(t, auth.CreatePasswordReset(ctx(), expired))
	_, err = auth.GetPasswordResetByHash(ctx(), "pr-expired-hash")
	require.True(t, apperr.IsNotFound(err), "muddati o'tgan reset token qaytarilmasligi kerak")
}

// MarkPasswordResetUsed tokenni ishlatilgan deb belgilashi va shu tokenning
// keyingi validatsiyasini bloklashi kerak. Ushlaydigan xato: belgilash amal
// qilmasa, bitta reset token bir necha marta ishlatilardi.
func TestAuthRepo_MarkPasswordResetUsed(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	auth := pgRepo.NewAuthRepo(pg)
	u := makeUser(t, users, "pr_mark@darsly.uz")

	pr := &entity.PasswordReset{
		ID: uuid.NewString(), UserID: u.ID, TokenHash: "pr-mark-hash",
		ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, auth.CreatePasswordReset(ctx(), pr))

	// Belgilashdan oldin — yaroqli.
	_, err := auth.GetPasswordResetByHash(ctx(), "pr-mark-hash")
	require.NoError(t, err)

	require.NoError(t, auth.MarkPasswordResetUsed(ctx(), pr.ID))

	// Belgilashdan keyin — endi topilmaydi (used_at to'ldirilgan).
	_, err = auth.GetPasswordResetByHash(ctx(), "pr-mark-hash")
	require.True(t, apperr.IsNotFound(err), "belgilangandan keyin token yaroqsiz bo'lishi kerak")
}

// GetRefreshTokenByHash `revoked_at IS NULL AND expires_at > NOW()` shartini
// hurmat qilishi kerak. Ushlaydigan xato: bekor qilingan/muddati o'tgan refresh
// token orqali sessiya tiklanardi.
func TestAuthRepo_GetRefreshTokenByHash_Filters(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	auth := pgRepo.NewAuthRepo(pg)
	u := makeUser(t, users, "rt_get@darsly.uz")

	mk := func(hash string, exp time.Duration) *entity.RefreshToken {
		rt := &entity.RefreshToken{
			ID: uuid.NewString(), UserID: u.ID, TokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(exp), CreatedAt: time.Now().UTC(),
		}
		require.NoError(t, auth.CreateRefreshToken(ctx(), rt))
		return rt
	}

	// 1) Yaroqli → qaytadi.
	mk("rt-valid", time.Hour)
	got, err := auth.GetRefreshTokenByHash(ctx(), "rt-valid")
	require.NoError(t, err)
	require.Equal(t, u.ID, got.UserID)

	// 2) Bekor qilingan → qaytmasligi kerak.
	rev := mk("rt-revoked", time.Hour)
	require.NoError(t, auth.RevokeRefreshToken(ctx(), rev.ID))
	_, err = auth.GetRefreshTokenByHash(ctx(), "rt-revoked")
	require.True(t, apperr.IsNotFound(err), "bekor qilingan refresh token qaytarilmasligi kerak")

	// 3) Muddati o'tgan → qaytmasligi kerak.
	mk("rt-expired", -time.Hour)
	_, err = auth.GetRefreshTokenByHash(ctx(), "rt-expired")
	require.True(t, apperr.IsNotFound(err), "muddati o'tgan refresh token qaytarilmasligi kerak")
}

// RevokeAllUserTokens FAQAT o'sha foydalanuvchining barcha tirik tokenlarini
// bekor qilishi, boshqa foydalanuvchinikiga tegmasligi kerak. Ushlaydigan xato:
// parol reset / deaktivatsiyada sessiyalar tozalanmay qolishi yoki begona
// foydalanuvchi sessiyasi ham o'chib ketishi (WHERE user_id noto'g'ri bo'lsa).
func TestAuthRepo_RevokeAllUserTokens(t *testing.T) {
	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	auth := pgRepo.NewAuthRepo(pg)
	victim := makeUser(t, users, "rt_victim@darsly.uz")
	other := makeUser(t, users, "rt_other@darsly.uz")

	mk := func(userID, hash string) {
		require.NoError(t, auth.CreateRefreshToken(ctx(), &entity.RefreshToken{
			ID: uuid.NewString(), UserID: userID, TokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(),
		}))
	}
	mk(victim.ID, "rt-victim-1")
	mk(victim.ID, "rt-victim-2")
	mk(other.ID, "rt-other-1")

	require.NoError(t, auth.RevokeAllUserTokens(ctx(), victim.ID))

	// Qurbonning ikkala tokeni ham endi yaroqsiz.
	_, err := auth.GetRefreshTokenByHash(ctx(), "rt-victim-1")
	require.True(t, apperr.IsNotFound(err), "victim 1-token bekor bo'lishi kerak")
	_, err = auth.GetRefreshTokenByHash(ctx(), "rt-victim-2")
	require.True(t, apperr.IsNotFound(err), "victim 2-token bekor bo'lishi kerak")

	// Begona foydalanuvchi tokeni tirik qolishi kerak.
	_, err = auth.GetRefreshTokenByHash(ctx(), "rt-other-1")
	require.NoError(t, err, "boshqa foydalanuvchi tokeni buzilmasligi kerak")
}

// B-6 [HIGH] — TEST-ONLY, KOD O'ZGARMAYDI.
//
// auth.go:160 MarkPasswordResetUsed `WHERE id=$1` bilan ishlaydi, `AND used_at
// IS NULL` YO'Q. ResetPassword oqimi (usecase/auth/auth.go:348) tranzaksiyasiz
// bo'lgani uchun bu TOCTOU: ikki parallel so'rov ikkalasi ham GetPasswordResetByHash
// dan o'tib (used_at hali NULL), keyin ikkalasi ham UPDATE bajaradi — bir martalik
// token IKKI marta "ishlatiladi". To'g'ri xulq: ikkinchi belgilash 0 qatorga
// ta'sir qilishi (used_at O'ZGARMASLIGI) kerak.
//
// Kod hozir nuqsonli bo'lgani uchun bu test YIQILADI → t.Skip bilan himoyalangan.
// Tuzatishdan (`AND used_at IS NULL` qo'shilgach) keyin skip'ni olib tashlang.
func TestAuthRepo_MarkPasswordResetUsed_OneTime_B6(t *testing.T) {
	t.Skip("B-6: known defect — MarkPasswordResetUsed not atomic; un-skip after fix")

	pg := testutil.SetupTestDB(t)
	users := pgRepo.NewUserRepo(pg)
	auth := pgRepo.NewAuthRepo(pg)
	u := makeUser(t, users, "pr_b6@darsly.uz")

	pr := &entity.PasswordReset{
		ID: uuid.NewString(), UserID: u.ID, TokenHash: "pr-b6-hash",
		ExpiresAt: time.Now().UTC().Add(time.Hour), CreatedAt: time.Now().UTC(),
	}
	require.NoError(t, auth.CreatePasswordReset(ctx(), pr))

	// Birinchi belgilash — used_at to'ldiriladi.
	require.NoError(t, auth.MarkPasswordResetUsed(ctx(), pr.ID))

	var first time.Time
	require.NoError(t, pg.DB.QueryRow(ctx(),
		"SELECT used_at FROM password_resets WHERE id=$1", pr.ID).Scan(&first))

	// Vaqt ilgarilashi uchun kichik pauza (ikkinchi NOW() farqli bo'lsin).
	time.Sleep(10 * time.Millisecond)

	// Ikkinchi belgilash — bir martalik token qayta ishlatilmasligi kerak.
	require.NoError(t, auth.MarkPasswordResetUsed(ctx(), pr.ID))

	var second time.Time
	require.NoError(t, pg.DB.QueryRow(ctx(),
		"SELECT used_at FROM password_resets WHERE id=$1", pr.ID).Scan(&second))

	// TO'G'RI xulq: used_at o'zgarmagan (ikkinchi UPDATE 0 qatorga ta'sir qilgan).
	require.True(t, first.Equal(second),
		"B-6: bir martalik reset token ikki marta belgilanmasligi kerak (used_at o'zgarmasin)")
}

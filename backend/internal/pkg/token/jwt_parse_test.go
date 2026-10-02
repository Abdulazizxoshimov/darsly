package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// S-1 [MEDIUM] — jwt.go:643 parse(). Ikki klassik JWT hujumini rad etishni
// tekshiradi. `parse` xususiy — public ValidateAccess orqali sinaymiz.

// parseSecret — jwt_test.go dagi newMakerWithGrace ichidagi imzo kaliti bilan
// AYNAN bir xil (HS256 token yasab yubora olishimiz uchun).
var parseSecret = []byte("test-secret-at-least-32-chars-long-000")

// alg:none hujumi — imzosiz token. parse() `SigningMethodHMAC` bo'lmasa rad
// etishi kerak. Ushlaydigan xato: agar method tekshiruvi tushib qolsa, hujumchi
// imzoni butunlay aylanib o'tib istalgan sub/role bilan token yasardi.
func TestValidateAccess_RejectsAlgNone(t *testing.T) {
	m, _ := newMaker(t)
	ctx := context.Background()

	claims := jwt.MapClaims{
		"sub": "attacker", "sid": "sid-none", "role": "mentor", "type": "access",
		"iat": time.Now().Add(-time.Minute).Unix(),
		"exp": time.Now().Add(time.Hour).Unix(),
		"jti": "none-jti",
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	require.NoError(t, err)

	_, err = m.ValidateAccess(ctx, tok)
	require.Error(t, err, "alg:none token rad etilishi kerak (imzoni aylanib o'tishga urinish)")
}

// Muddati o'tgan token — to'g'ri imzolangan, lekin exp o'tgan. golang-jwt/v5 uni
// avtomatik rad etadi. Ushlaydigan xato: exp validatsiyasi o'chirilsa, eskirgan
// access token cheksiz ishlayverardi (sessiyani bekor qilib bo'lmasdi).
func TestValidateAccess_RejectsExpired(t *testing.T) {
	m, _ := newMaker(t)
	ctx := context.Background()

	now := time.Now()
	claims := jwt.MapClaims{
		"sub": "user", "sid": "sid-exp", "role": "student", "type": "access",
		"iat": now.Add(-2 * time.Hour).Unix(),
		"exp": now.Add(-time.Hour).Unix(), // muddati o'tgan
		"jti": "exp-jti",
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(parseSecret)
	require.NoError(t, err)

	_, err = m.ValidateAccess(ctx, tok)
	require.Error(t, err, "muddati o'tgan access token rad etilishi kerak")
}

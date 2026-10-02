package token_test

// S-3 — ValidateAccess Redis-outage grace oynasi (cheklangan fail-open).
//
// Redis yetib bo'lmasa ikki yomon variant bor: fail-open (bekor qilingan
// tokenlar tiriladi) yoki fail-closed (hamma 401, platforma to'xtaydi). Kod
// CHEKLANGAN fail-open tanlaydi: Redis uzilgan bo'lsa token FAQAT
// `redisOutageGrace` (5 daqiqa) ichida yaratilgan bo'lsa qabul qilinadi.
// Testlar shu chegarani mixlab qo'yadi — aks holda kimdir grace'ni cheksiz
// qilib, uzilishda barcha eski/bekor tokenlarni tiriltirib yuborardi.

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"
)

// jwt_test.go dagi newMakerWithGrace bilan AYNI signing key.
var outageSigningKey = []byte("test-secret-at-least-32-chars-long-000")

// deadRedisMaker — hech qachon ulanmaydigan Redis'ga qaratilgan maker
// (Redis outage'ni real server holda simulyatsiya qiladi). Har buyruq xato beradi.
func deadRedisMaker(t *testing.T) token.Maker {
	t.Helper()
	cli := goredis.NewClient(&goredis.Options{
		Addr:        "127.0.0.1:1", // ulanish rad etiladi
		DialTimeout: 200 * time.Millisecond,
		MaxRetries:  -1,
	})
	log := logger.New("error", "test", "test")
	return token.NewJWTMaker(outageSigningKey, 15*time.Minute, 720*time.Hour, 0, cli, "outagetest", log)
}

// mintAccess — berilgan `iat` bilan yaroqli imzolangan access token yasaydi
// (Generate Redis talab qiladi; bu yerda esa yoshini boshqarish kerak).
func mintAccess(t *testing.T, iat time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{
		"sub":  "u1",
		"sid":  "sid-1",
		"role": "student",
		"type": "access",
		"iat":  iat.Unix(),
		"exp":  time.Now().Add(15 * time.Minute).Unix(),
		"jti":  "jti-1",
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(outageSigningKey)
	require.NoError(t, err)
	return s
}

// ⭐ Redis uzilgan + token YANGI (grace ichida) → vaqtincha qabul qilinadi.
// Bug: grace bo'lmasa Redis uzilishida butun platforma 401 berardi (jonli
// darslar uzilardi).
func TestValidateAccess_RedisOutage_FreshTokenAccepted(t *testing.T) {
	m := deadRedisMaker(t)
	tok := mintAccess(t, time.Now()) // hozir yaratilgan

	claims, err := m.ValidateAccess(context.Background(), tok)
	require.NoError(t, err, "grace oynasidagi yangi token Redis uzilganda ham qabul qilinishi kerak")
	require.Equal(t, "u1", claims.Sub)
}

// ⭐ Redis uzilgan + token ESKI (grace tashqarisida) → RAD etiladi (fail-closed).
// Bug: grace cheksiz bo'lsa bekor qilingan/eskirgan token uzilish paytida
// qayta tirilib, chiqarilgan foydalanuvchi kira olardi.
func TestValidateAccess_RedisOutage_OldTokenRejected(t *testing.T) {
	m := deadRedisMaker(t)
	tok := mintAccess(t, time.Now().Add(-10*time.Minute)) // grace (5m) dan eski

	_, err := m.ValidateAccess(context.Background(), tok)
	require.Error(t, err, "grace tashqarisidagi eski token Redis uzilganda rad etilishi kerak")
}

// iat umuman yo'q token → RAD (yoshini aniqlab bo'lmaganda fail-closed).
func TestValidateAccess_RedisOutage_MissingIatRejected(t *testing.T) {
	m := deadRedisMaker(t)
	claims := jwt.MapClaims{
		"sub": "u1", "sid": "sid-1", "role": "student", "type": "access",
		"exp": time.Now().Add(15 * time.Minute).Unix(), "jti": "jti-2",
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(outageSigningKey)
	require.NoError(t, err)

	_, err = m.ValidateAccess(context.Background(), tok)
	require.Error(t, err, "iat siz token uzilishda rad etilishi kerak (yoshi noma'lum)")
}

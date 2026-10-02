package livekit

// S-2 — VerifyToken imzo tekshiruvi (guest identity manbai).
//
// Guest'lar (JWT'siz) chat/poll uchun room-token orqali autentifikatsiya
// qilinadi: VerifyToken imzoni tekshirib identity/name/room qaytaradi. Agar
// SOXTA imzoli token qabul qilinsa, istalgan kishi o'zini istalgan identity
// qilib ko'rsatib boshqa guest nomidan chat/ovoz yubora olardi.

import (
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/stretchr/testify/require"
)

// Yaroqli token → kutilgan identity/name/room.
func TestVerifyToken_ValidYieldsIdentity(t *testing.T) {
	c := testClient(t, time.Hour)
	tok, err := c.AccessToken("lesson_x", "guest1", "Mehmon", false)
	require.NoError(t, err)

	id, name, room, err := c.VerifyToken(tok)
	require.NoError(t, err)
	require.Equal(t, "guest1", id, "identity aynan tokendan olinishi kerak")
	require.Equal(t, "Mehmon", name)
	require.Equal(t, "lesson_x", room)
}

// ⭐ Boshqa sir bilan imzolangan token RAD etiladi (soxta imzo).
// Bug: imzo tekshirilmasa hujumchi o'z sirini bilan istalgan identity
// yozib, boshqa guest nomidan gapira olardi.
func TestVerifyToken_ForgedSignatureRejected(t *testing.T) {
	c := testClient(t, time.Hour)

	// Server BILMAYDIGAN sir bilan imzolangan, lekin bir xil API key bilan.
	forged, err := auth.NewAccessToken(testKey, "totally_different_secret_32_chars_xxxxxx").
		SetVideoGrant(&auth.VideoGrant{RoomJoin: true, Room: "lesson_x"}).
		SetIdentity("victim").
		SetName("Qurbon").
		SetValidFor(time.Hour).
		ToJWT()
	require.NoError(t, err)

	_, _, _, verr := c.VerifyToken(forged)
	require.Error(t, verr, "soxta imzoli token qabul qilinmasligi kerak")
}

// Buzuq/bo'lmagan token ham xato beradi (nil pointer / panic emas).
func TestVerifyToken_GarbageRejected(t *testing.T) {
	c := testClient(t, time.Hour)
	_, _, _, err := c.VerifyToken("not-a-jwt")
	require.Error(t, err)
}

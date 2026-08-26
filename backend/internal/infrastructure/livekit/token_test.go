package livekit

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/livekit/protocol/auth"
	"github.com/stretchr/testify/require"

	"github.com/zoom/darsly/internal/pkg/config"
)

const (
	testKey    = "devkey"
	testSecret = "secret_at_least_32_characters_long_000000"
)

func testClient(t *testing.T, ttl time.Duration) *Client {
	t.Helper()
	return New(config.LiveKitConfig{
		APIKey: testKey, APISecret: testSecret, Host: "http://livekit:7880", TokenTTL: ttl,
	})
}

// grants — tokenni imzo bo'yicha ochib, ichidagi da'volarni qaytaradi.
func grants(t *testing.T, token string) *auth.ClaimGrants {
	t.Helper()
	_, g := claimsAndGrants(t, token)
	return g
}

// tokenTTLOf — tokenning qolgan amal muddati (`exp` da'vosidan).
func tokenTTLOf(t *testing.T, token string) time.Duration {
	t.Helper()
	rc, _ := claimsAndGrants(t, token)
	return time.Until(rc.ExpiresAt.Time)
}

func claimsAndGrants(t *testing.T, token string) (*jwt.RegisteredClaims, *auth.ClaimGrants) {
	t.Helper()
	v, err := auth.ParseAPIToken(token)
	require.NoError(t, err)
	rc, g, err := v.Verify(testSecret)
	require.NoError(t, err)
	return rc, g
}

// C-1 — "kim host" savoliga klient soxtalashtira olmaydigan javob.
//
// Data-channel xabari kimdan kelganini klient payload'dan bilolmaydi (u soxta
// bo'lishi mumkin). LiveKit esa ishtirokchi metadata'sini AYNAN shu tokendan
// oladi va uni boshqa klientlarga o'zgarmas holda tarqatadi. `RoomAdmin` grant'i
// bu ish uchun yaramaydi — u boshqa klientlarga umuman ko'rinmaydi.
func TestAccessToken_SignsRoleMetadata(t *testing.T) {
	c := testClient(t, 6*time.Hour)

	t.Run("host tokeni role=host bilan imzolanadi", func(t *testing.T) {
		tok, err := c.AccessToken("lesson_x", "mentor1", "Dilnoza", true)
		require.NoError(t, err)

		var md RoleMetadata
		require.NoError(t, json.Unmarshal([]byte(grants(t, tok).Metadata), &md))
		require.Equal(t, RoleHost, md.Role)
	})

	t.Run("ishtirokchi tokeni role=participant bilan imzolanadi", func(t *testing.T) {
		tok, err := c.AccessToken("lesson_x", "guest1", "Mehmon", false)
		require.NoError(t, err)

		var md RoleMetadata
		require.NoError(t, json.Unmarshal([]byte(grants(t, tok).Metadata), &md))
		require.Equal(t, RoleParticipant, md.Role,
			"mehmon tokeni host deb belgilanmasin — bu butun ishonch modelini buzardi")
	})
}

// C-2 — chiqarilgan ishtirokchining eski tokeni bilan qaytish oynasi.
//
// `RemoveParticipant` tokenni bekor qila olmaydi (LiveKit'da ban ro'yxati yo'q),
// shuning uchun oynani TTL cheklaydi: ishtirokchi ko'pi bilan shuncha vaqtdan
// keyin token yangilash uchun backendga — ya'ni ban tekshiruviga — kelishi shart.
func TestAccessToken_ParticipantTTLIsShort(t *testing.T) {
	t.Run("ishtirokchi tokeni 30 daqiqadan oshmaydi", func(t *testing.T) {
		c := testClient(t, 6*time.Hour) // konfiguratsiya uzun bo'lsa ham
		tok, err := c.AccessToken("lesson_x", "guest1", "Mehmon", false)
		require.NoError(t, err)

		ttl := tokenTTLOf(t, tok)
		require.LessOrEqual(t, ttl, participantTokenTTLMax+time.Minute)
		require.Greater(t, ttl, 25*time.Minute, "juda qisqa bo'lsa qayta ulanish uzilardi")
	})

	t.Run("host tokeni dars davomiyligicha qoladi", func(t *testing.T) {
		c := testClient(t, 6*time.Hour)
		tok, err := c.AccessToken("lesson_x", "mentor1", "Dilnoza", true)
		require.NoError(t, err)

		ttl := tokenTTLOf(t, tok)
		require.Greater(t, ttl, participantTokenTTLMax,
			"ustoz dars o'rtasida tokeni tugab qolgani uchun chiqib ketmasin")
	})

	t.Run("konfiguratsiya qisqaroq bo'lsa uzaytirilmaydi", func(t *testing.T) {
		c := testClient(t, 5*time.Minute)
		tok, err := c.AccessToken("lesson_x", "guest1", "Mehmon", false)
		require.NoError(t, err)

		ttl := tokenTTLOf(t, tok)
		require.LessOrEqual(t, ttl, 6*time.Minute)
	})
}

// ZOOM MODELI (PRODUCT.md D-intervyu, №11): o'quvchi ham mikrofon va kamerani
// ERKIN yoqa oladi — lekin EKRAN ULASHISH faqat ustozda. Data-channel hammaga
// ochiq (chat/reaksiya busiz ishlamaydi) — aynan shu "hammaga ochiq" C-1 ni
// zarur qiladi. Grant o'zgarsa bu shartnoma ham qayta ko'rib chiqilishi kerak.
func TestAccessToken_Grants(t *testing.T) {
	c := testClient(t, time.Hour)

	host, err := c.AccessToken("lesson_x", "mentor1", "Dilnoza", true)
	require.NoError(t, err)
	hg := grants(t, host).Video
	require.True(t, *hg.CanPublish)
	require.True(t, hg.RoomAdmin)
	require.True(t, hg.RoomRecord)
	require.Empty(t, hg.CanPublishSources, "host uchun manba cheklovi YO'Q (ekran ulashish ham kiradi)")

	part, err := c.AccessToken("lesson_x", "guest1", "Mehmon", false)
	require.NoError(t, err)
	pg := grants(t, part).Video
	require.True(t, *pg.CanPublish, "o'quvchi MIKROFONNI O'ZI yoqa oladi (ovozli savol darhol, ustoz ruxsatisiz)")
	require.False(t, pg.RoomAdmin, "o'quvchida moderatsiya huquqi bo'lmasin")
	require.False(t, pg.RoomRecord)
	require.True(t, *pg.CanPublishData, "chat/reaksiya uchun kerak")

	// Default manba: FAQAT mikrofon. Kamera (video) ustoz 'allow-speak' bergach
	// qo'shiladi (studentVideoSources). Ekran ulashish hech qachon yo'q.
	// (Bo'sh ro'yxat LiveKit'da "hammasi ochiq" degani — shuning uchun aniq tekshiramiz.)
	require.NotEmpty(t, pg.CanPublishSources, "bo'sh ro'yxat = hamma manba ochiq — bu xato bo'lardi")
	require.ElementsMatch(t, []string{"microphone"}, pg.CanPublishSources,
		"default: FAQAT mikrofon; kamera ustoz ruxsati bilan, screen_share hech qachon yo'q")
}

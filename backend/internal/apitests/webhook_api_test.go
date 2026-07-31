package apitests_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"testing"
	"time"

	"github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	livekitpkg "github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/pkg/config"
)

// M20 — `POST /webhooks/livekit` imzo tekshiruvi.
//
// # Nega bu test alohida muhim
//
// Bu endpoint OCHIQ (auth middleware'idan tashqarida) va uni internetdan har
// kim chaqira oladi. Yagona himoya — LiveKit imzosi. Imzo tekshiruvi buzilsa
// (masalan kutubxona yangilanishida, yoki kalitlar noto'g'ri uzatilsa) begona
// odam soxta `egress_ended` yuborib yozuvlarni "failed" deb belgilashi yoki
// `participant_joined` bilan ban mantiqini qo'zg'atishi mumkin bo'lardi.
//
// Auditgacha bu yo'l faqat QO'LDA tekshirilgan edi.

const (
	whKey    = "webhook-test-key"
	whSecret = "webhook-test-secret-at-least-32-chars-000"
)

// signedWebhook — LiveKit yuboradigan so'rovning AYNAN nusxasi:
// protojson tanasi + tananing sha256'si imzolangan JWT (`url_notifier.send`).
func signedWebhook(t *testing.T, url string, ev *lkproto.WebhookEvent, key, secret string) *http.Request {
	t.Helper()
	body, err := protojson.Marshal(ev)
	require.NoError(t, err)

	sum := sha256.Sum256(body)
	tok, err := auth.NewAccessToken(key, secret).
		SetValidFor(5 * time.Minute).
		SetSha256(base64.StdEncoding.EncodeToString(sum[:])).
		ToJWT()
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", tok)
	req.Header.Set("Content-Type", "application/webhook+json")
	return req
}

func TestLiveKitWebhookSignature(t *testing.T) {
	lk := livekitpkg.New(config.LiveKitConfig{
		Host: "http://livekit.invalid:7880", APIKey: whKey, APISecret: whSecret,
	})
	require.True(t, lk.Enabled(), "test klienti yoqilgan bo'lishi kerak")

	srv, _ := newTestServerWithLiveKit(t, lk)
	clearRateLimits(t)
	url := srv.URL + "/api/v1/webhooks/livekit"

	ev := &lkproto.WebhookEvent{
		Event: webhook.EventParticipantJoined,
		Room:  &lkproto.Room{Name: "lesson_11111111-1111-4111-8111-111111111111"},
		Participant: &lkproto.ParticipantInfo{
			Identity: "guest_test",
		},
	}

	t.Run("to'g'ri imzo qabul qilinadi", func(t *testing.T) {
		res, err := http.DefaultClient.Do(signedWebhook(t, url, ev, whKey, whSecret))
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("boshqa SIR bilan imzolangan so'rov rad etiladi", func(t *testing.T) {
		res, err := http.DefaultClient.Do(
			signedWebhook(t, url, ev, whKey, "boshqa-sir-at-least-32-characters-0000"))
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode,
			"noto'g'ri sir bilan imzo qabul qilinmasligi kerak")
	})

	t.Run("noma'lum KALIT rad etiladi", func(t *testing.T) {
		res, err := http.DefaultClient.Do(signedWebhook(t, url, ev, "begona-kalit", whSecret))
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode)
	})

	t.Run("imzosiz so'rov rad etiladi", func(t *testing.T) {
		body, err := protojson.Marshal(ev)
		require.NoError(t, err)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/webhook+json")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode,
			"ochiq endpoint imzosiz chaqiruvni qabul qilmasligi SHART")
	})

	// ⭐ Eng nozik holat: imzo TO'G'RI, lekin tana almashtirilgan.
	//
	// Imzo tananing sha256'sini qamrab oladi. Faqat JWT tekshirilib, sha256
	// solishtirilmasa, hujumchi bir marta ushlangan yaroqli imzo bilan
	// ISTALGAN hodisani yuborardi (masalan har bir yozuvni "failed" qilib).
	t.Run("imzo to'g'ri, lekin tana o'zgartirilgan — rad etiladi", func(t *testing.T) {
		req := signedWebhook(t, url, ev, whKey, whSecret)

		tampered := &lkproto.WebhookEvent{
			Event: webhook.EventEgressEnded,
			Room:  &lkproto.Room{Name: "lesson_22222222-2222-4222-8222-222222222222"},
		}
		newBody, err := protojson.Marshal(tampered)
		require.NoError(t, err)
		req.Body = http.NoBody
		req2, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(newBody))
		require.NoError(t, err)
		req2.Header = req.Header.Clone() // ESKI (yaroqli) imzo

		res, err := http.DefaultClient.Do(req2)
		require.NoError(t, err)
		defer res.Body.Close()
		require.Equal(t, http.StatusUnauthorized, res.StatusCode,
			"tana o'zgarganda imzo yaroqsiz bo'lishi kerak (sha256 bog'lanishi)")
	})
}

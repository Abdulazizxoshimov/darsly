package v1

import (
	"testing"

	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"
	"github.com/stretchr/testify/require"
)

// Yozib olish QAYSI webhook hodisasida boshlanishi — mahsulot qarori.
//
// Bu qaror bir marta noto'g'ri bo'lgan: yozuv `room.HostToken` da (token
// berilganda) boshlanardi va egress 5 daqiqa media kutib `egress_aborted
// "Start signal not received"` bilan bekor bo'lardi. Test o'sha saboqni
// qotiradi.
func TestRecordingTriggerRoom(t *testing.T) {
	room := &livekit.Room{Name: "lesson_11111111-1111-4111-8111-111111111111"}

	t.Run("track_published yozuvni boshlaydi", func(t *testing.T) {
		name, ok := recordingTriggerRoom(&livekit.WebhookEvent{
			Event: webhook.EventTrackPublished, Room: room,
		})
		require.True(t, ok)
		require.Equal(t, room.Name, name)
	})

	t.Run("participant_joined YETARLI EMAS", func(t *testing.T) {
		// Ishtirokchi kirdi, lekin hali trek e'lon qilmadi (ruxsat oynasida
		// turibdi). Egress hozir boshlansa yana 5 daqiqa bo'sh kutardi.
		_, ok := recordingTriggerRoom(&livekit.WebhookEvent{
			Event: webhook.EventParticipantJoined, Room: room,
		})
		require.False(t, ok)
	})

	t.Run("boshqa hodisalar tegmaydi", func(t *testing.T) {
		for _, e := range []string{
			webhook.EventRoomStarted, webhook.EventRoomFinished,
			webhook.EventTrackUnpublished, webhook.EventEgressEnded,
			webhook.EventParticipantLeft, "",
		} {
			_, ok := recordingTriggerRoom(&livekit.WebhookEvent{Event: e, Room: room})
			require.False(t, ok, "hodisa yozuvni boshlamasligi kerak: %q", e)
		}
	})

	t.Run("xonasiz hodisa yiqitmaydi", func(t *testing.T) {
		_, ok := recordingTriggerRoom(&livekit.WebhookEvent{Event: webhook.EventTrackPublished})
		require.False(t, ok)
		_, ok = recordingTriggerRoom(&livekit.WebhookEvent{
			Event: webhook.EventTrackPublished, Room: &livekit.Room{Name: ""},
		})
		require.False(t, ok)
		_, ok = recordingTriggerRoom(nil)
		require.False(t, ok)
	})
}

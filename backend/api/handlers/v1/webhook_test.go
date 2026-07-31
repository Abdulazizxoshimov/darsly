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

// C-2: ban qaysi webhook hodisasida qo'llanadi.
//
// `recordingTriggerRoom` bilan bir xil sababdan alohida sinaladi — bu ham
// mahsulot/xavfsizlik qarori, va uni imzolangan HTTP so'rovi yasamasdan
// tekshirish mumkin.
//
// Aynan `participant_joined`: `track_published` dan farqli, u ishtirokchi
// media e'lon qilmasa ham keladi — buzg'unchi esa kamera yoqmasdan ham
// chat/so'rovnoma orqali darsni buza oladi.
func TestJoinedParticipant(t *testing.T) {
	room := &livekit.Room{Name: "lesson_11111111-1111-4111-8111-111111111111"}
	p := &livekit.ParticipantInfo{Identity: "buzgunchi", Name: "Buzg'unchi"}

	t.Run("participant_joined ban tekshiruvini ishga tushiradi", func(t *testing.T) {
		rn, identity, name, ok := joinedParticipant(&livekit.WebhookEvent{
			Event: webhook.EventParticipantJoined, Room: room, Participant: p,
		})
		require.True(t, ok)
		require.Equal(t, room.Name, rn)
		require.Equal(t, "buzgunchi", identity)
		require.Equal(t, "Buzg'unchi", name,
			"ism ham qaytishi kerak — mentor blocklist ism bo'yicha tekshiradi (№4)")
	})

	t.Run("identity'siz hodisa e'tiborsiz", func(t *testing.T) {
		_, _, _, ok := joinedParticipant(&livekit.WebhookEvent{
			Event: webhook.EventParticipantJoined, Room: room,
			Participant: &livekit.ParticipantInfo{},
		})
		require.False(t, ok)
	})

	t.Run("xonasiz hodisa e'tiborsiz", func(t *testing.T) {
		_, _, _, ok := joinedParticipant(&livekit.WebhookEvent{
			Event: webhook.EventParticipantJoined, Participant: p,
		})
		require.False(t, ok)
	})

	t.Run("boshqa hodisalar ban tekshirmaydi", func(t *testing.T) {
		for _, e := range []string{
			webhook.EventTrackPublished, webhook.EventRoomStarted,
			webhook.EventRoomFinished, webhook.EventParticipantLeft,
		} {
			_, _, _, ok := joinedParticipant(&livekit.WebhookEvent{Event: e, Room: room, Participant: p})
			require.False(t, ok, e)
		}
	})

	t.Run("nil hodisa panika qilmaydi", func(t *testing.T) {
		_, _, _, ok := joinedParticipant(nil)
		require.False(t, ok)
	})
}

// №11 (Zoom modeli): ovoz siyosati QAYSI hodisada qo'llanadi — mahsulot qarori.
//
// Aynan `track_published` + trek turi AUDIO: `participant_joined` da trek hali
// yo'q (mute qiladigan narsaning o'zi bo'lmaydi), kamera publish'i esa ovoz
// siyosatiga daxlsiz.
func TestPublishedAudioTrack(t *testing.T) {
	room := &livekit.Room{Name: "lesson_11111111-1111-4111-8111-111111111111"}
	p := &livekit.ParticipantInfo{Identity: "oquvchi"}
	audio := &livekit.TrackInfo{Sid: "TR_a", Type: livekit.TrackType_AUDIO}
	video := &livekit.TrackInfo{Sid: "TR_v", Type: livekit.TrackType_VIDEO}

	t.Run("audio track_published siyosatni ishga tushiradi", func(t *testing.T) {
		rn, identity, ok := publishedAudioTrack(&livekit.WebhookEvent{
			Event: webhook.EventTrackPublished, Room: room, Participant: p, Track: audio,
		})
		require.True(t, ok)
		require.Equal(t, room.Name, rn)
		require.Equal(t, "oquvchi", identity)
	})

	t.Run("video (kamera) publish'i tegmaydi", func(t *testing.T) {
		_, _, ok := publishedAudioTrack(&livekit.WebhookEvent{
			Event: webhook.EventTrackPublished, Room: room, Participant: p, Track: video,
		})
		require.False(t, ok, "kamera yoqish mikrofonni mute qilmasligi kerak")
	})

	t.Run("track'siz yoki to'liqsiz hodisa e'tiborsiz", func(t *testing.T) {
		_, _, ok := publishedAudioTrack(&livekit.WebhookEvent{
			Event: webhook.EventTrackPublished, Room: room, Participant: p,
		})
		require.False(t, ok)
		_, _, ok = publishedAudioTrack(&livekit.WebhookEvent{
			Event: webhook.EventTrackPublished, Room: room, Track: audio,
		})
		require.False(t, ok)
		_, _, ok = publishedAudioTrack(&livekit.WebhookEvent{
			Event: webhook.EventTrackPublished, Participant: p, Track: audio,
		})
		require.False(t, ok)
		_, _, ok = publishedAudioTrack(nil)
		require.False(t, ok)
	})

	t.Run("boshqa hodisalar siyosat qo'llamaydi", func(t *testing.T) {
		for _, e := range []string{
			webhook.EventParticipantJoined, webhook.EventTrackUnpublished,
			webhook.EventRoomStarted, webhook.EventEgressEnded,
		} {
			_, _, ok := publishedAudioTrack(&livekit.WebhookEvent{Event: e, Room: room, Participant: p, Track: audio})
			require.False(t, ok, e)
		}
	})
}

// №2 (avto-yakun): QAYSI hodisa xona bo'shaganini bildiradi — mahsulot qarori.
//
// Bu signal grace hisobining boshlanish nuqtasi: noto'g'ri hodisadan
// hisoblansa dars ustoz hali xonada turganda yakunlanib ketardi.
func TestRoomOccupancy(t *testing.T) {
	full := &livekit.Room{Name: "lesson_11111111-1111-4111-8111-111111111111", NumParticipants: 2}
	empty := &livekit.Room{Name: "lesson_11111111-1111-4111-8111-111111111111", NumParticipants: 0}

	t.Run("oxirgi ishtirokchi chiqdi → bo'sh", func(t *testing.T) {
		rn, isEmpty, ok := roomOccupancy(&livekit.WebhookEvent{Event: webhook.EventParticipantLeft, Room: empty})
		require.True(t, ok)
		require.True(t, isEmpty)
		require.Equal(t, empty.Name, rn)
	})

	t.Run("kimdir qoldi → bo'sh emas", func(t *testing.T) {
		_, isEmpty, ok := roomOccupancy(&livekit.WebhookEvent{Event: webhook.EventParticipantLeft, Room: full})
		require.True(t, ok)
		require.False(t, isEmpty, "xonada odam qolgan bo'lsa hisob boshlanmasligi kerak")
	})

	t.Run("ishtirokchi kirdi → bo'sh emas", func(t *testing.T) {
		_, isEmpty, ok := roomOccupancy(&livekit.WebhookEvent{Event: webhook.EventParticipantJoined, Room: empty})
		require.True(t, ok)
		require.False(t, isEmpty, "kirish hisobni bekor qilishi kerak")
	})

	t.Run("room_finished → bo'sh", func(t *testing.T) {
		_, isEmpty, ok := roomOccupancy(&livekit.WebhookEvent{Event: webhook.EventRoomFinished, Room: full})
		require.True(t, ok)
		require.True(t, isEmpty, "SFU xonani o'chirgan bo'lsa u albatta bo'sh")
	})

	t.Run("boshqa hodisalar va to'liqsiz hodisa e'tiborsiz", func(t *testing.T) {
		for _, e := range []string{
			webhook.EventTrackPublished, webhook.EventRoomStarted, webhook.EventEgressEnded,
		} {
			_, _, ok := roomOccupancy(&livekit.WebhookEvent{Event: e, Room: empty})
			require.False(t, ok, e)
		}
		_, _, ok := roomOccupancy(&livekit.WebhookEvent{Event: webhook.EventParticipantLeft})
		require.False(t, ok)
		_, _, ok = roomOccupancy(nil)
		require.False(t, ok)
	})
}

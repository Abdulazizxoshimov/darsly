package livekit

import (
	"testing"

	"github.com/livekit/protocol/livekit"
)

// BE-13: "so'zga ruxsat" (allow-speak) studentga EKRAN ULASHISHNI bermasligi kerak.
// LiveKit'da bo'sh CanPublishSources = BARCHA manbalar, shuning uchun ro'yxat aniq
// berilishi shart. Bu test regressiya qulfi: kimdir ro'yxatni olib tashlasa yoki
// SCREEN_SHARE qo'shsa test yiqiladi.
func TestStudentPermission_NoScreenShare(t *testing.T) {
	for _, canPublish := range []bool{true, false} {
		p := studentPermission(canPublish)

		if len(p.CanPublishSources) == 0 {
			t.Fatal("CanPublishSources bo'sh — LiveKit buni 'barcha manbalar' deb tushunadi (ekran ulashish ochiq qoladi)")
		}
		if p.CanPublish != canPublish {
			t.Fatalf("CanPublish = %v, kutilgan %v", p.CanPublish, canPublish)
		}
		if !p.CanSubscribe {
			t.Fatal("student darsni ko'rishi kerak (CanSubscribe)")
		}

		got := map[livekit.TrackSource]bool{}
		for _, s := range p.CanPublishSources {
			got[s] = true
		}
		if !got[livekit.TrackSource_CAMERA] {
			t.Error("kamera ruxsati yo'q — 'so'zga ruxsat'da video bo'lishi kerak")
		}
		if !got[livekit.TrackSource_MICROPHONE] {
			t.Error("mikrofon ruxsati yo'q — 'so'zga ruxsat'ning asosiy maqsadi")
		}
		if got[livekit.TrackSource_SCREEN_SHARE] {
			t.Error("SCREEN_SHARE studentga berilgan — webinar modeliga zid (darsni buzish vektori)")
		}
		if got[livekit.TrackSource_SCREEN_SHARE_AUDIO] {
			t.Error("SCREEN_SHARE_AUDIO studentga berilgan")
		}
	}
}

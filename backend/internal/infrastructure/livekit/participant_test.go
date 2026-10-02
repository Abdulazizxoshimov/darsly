package livekit

import (
	"testing"

	"github.com/livekit/protocol/livekit"
)

// studentPermission Variant B: MIKROFON DOIM ochiq (CanPublish=true), KAMERA esa
// allowCamera bilan; SCREEN_SHARE hech qachon berilmaydi. LiveKit'da bo'sh
// CanPublishSources = BARCHA manbalar, shuning uchun ro'yxat hech qachon bo'sh
// qolmasligi shart. Bu test regressiya qulfi.
func TestStudentPermission_NoScreenShare(t *testing.T) {
	for _, allowCamera := range []bool{true, false} {
		p := studentPermission(allowCamera)

		if len(p.CanPublishSources) == 0 {
			t.Fatal("CanPublishSources bo'sh — LiveKit buni 'barcha manbalar' deb tushunadi (ekran ulashish ochiq qoladi)")
		}
		if !p.CanPublish {
			t.Fatal("CanPublish false — o'quvchi kamida mikrofonni yoqa olishi kerak")
		}
		if !p.CanSubscribe {
			t.Fatal("student darsni ko'rishi kerak (CanSubscribe)")
		}

		got := map[livekit.TrackSource]bool{}
		for _, s := range p.CanPublishSources {
			got[s] = true
		}
		if !got[livekit.TrackSource_MICROPHONE] {
			t.Error("mikrofon ruxsati yo'q — o'quvchi ovozi DOIM ochiq bo'lishi kerak")
		}
		if got[livekit.TrackSource_CAMERA] != allowCamera {
			t.Errorf("kamera ruxsati = %v, kutilgan %v (allowCamera bilan mos)", got[livekit.TrackSource_CAMERA], allowCamera)
		}
		if got[livekit.TrackSource_SCREEN_SHARE] {
			t.Error("SCREEN_SHARE studentga berilgan — webinar modeliga zid (darsni buzish vektori)")
		}
		if got[livekit.TrackSource_SCREEN_SHARE_AUDIO] {
			t.Error("SCREEN_SHARE_AUDIO studentga berilgan")
		}
	}
}

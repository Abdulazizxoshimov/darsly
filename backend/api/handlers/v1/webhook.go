package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
)

// LiveKitWebhook godoc
// @Summary      LiveKit webhook (Egress status) — imzo bilan himoyalangan, ochiq endpoint
// @Tags         webhook
// @Accept       json
// @Param        Authorization  header  string  true  "LiveKit imzo tokeni"
// @Success      200
// @Router       /api/v1/webhooks/livekit [post]
func LiveKitWebhook(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.LiveKit == nil || !h.LiveKit.Enabled() {
			c.Status(http.StatusOK) // LiveKit sozlanmagan — jimgina qabul qilamiz
			return
		}
		// Imzoni tekshiradi (body sha256 + JWT).
		event, err := h.LiveKit.ParseWebhook(c.Request)
		if err != nil {
			hs.AbortError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid webhook signature")
			return
		}

		// ⭐ YOZIB OLISHNI BOSHLASH — xonada haqiqatan media paydo bo'lganda.
		//
		// Nega `track_published`, `participant_joined` emas: egress "start
		// signal"ni ishtirokchi kirganda emas, **trek e'lon qilinganda** oladi.
		// Ishtirokchi kirib, ruxsat oynasida turib qolsa yoki mikrofonni
		// yoqmasa — egress baribir 5 daqiqa kutib bekor bo'lardi.
		//
		// Xato JIMGINA yutiladi (200 qaytadi) va bu ataylab: non-200 bo'lsa
		// LiveKit shu hodisani qayta-qayta yuboradi va har urinishda yangi
		// egress boshlashga harakat qilinardi — 4 yadroli serverda bu
		// yozuvlar ko'chkisiga aylanardi. Xatoning o'zi usecase ichida
		// jurnalga yozilgan (`recording.startEgress`), ya'ni yo'qolmaydi.
		if roomName, ok := recordingTriggerRoom(event); ok {
			_ = h.Recording.EnsureForRoom(c.Request.Context(), roomName)
		}

		// ⭐ BAN'NI QO'LLASH — chiqarilgan ishtirokchi qaytib kirsa uzamiz.
		//
		// `RemoveParticipant` faqat joriy ulanishni uzadi; token qo'lda qoladi va
		// buzg'unchi backendga umuman murojaat qilmasdan qayta ulana oladi. Bu
		// hodisa esa aynan SFU tomonidan, HAR ULANISHDA keladi — ya'ni tokeni
		// qayerdan bo'lishidan qat'i nazar. Ism ham uzatiladi: mentor darajasidagi
		// doimiy qora ro'yxat (№4) ism bo'yicha tekshiriladi. Batafsil: `room.EnforceJoin`.
		if rn, identity, name, ok := joinedParticipant(event); ok {
			h.Room.EnforceJoin(c.Request.Context(), rn, identity, name)
		}

		// ⭐ XONA BO'SHLIGI (avto-yakun, №2) — grace hisobining boshlanish nuqtasi.
		//
		// Webhook aniq LAHZANI beradi: oxirgi ishtirokchi chiqqan payt. Ishchi
		// (`worker.AutoEndWorker`) bu belgini ko'rib, `LESSON_EMPTY_GRACE` o'tgach
		// darsni yakunlaydi. Webhook yo'qolsa ham ishchi xonani o'zi tekshirib
		// belgini qo'yadi — ya'ni bu tezlik uchun, ishonchlilik uchun emas.
		if rn, empty, ok := roomOccupancy(event); ok {
			if empty {
				h.Room.NoteRoomEmpty(c.Request.Context(), rn)
			} else {
				h.Room.NoteRoomOccupied(c.Request.Context(), rn)
			}
		}

		// ⭐ OVOZ SIYOSATI (Zoom modeli, №11) — audio trek e'lon qilinganda dars
		// sozlamalari qo'llanadi: mute_on_entry (birinchi publish mute bilan
		// boshlanadi) va allow_self_unmute=false (har publish qayta mute).
		// Xato jimgina yutiladi (EnforceJoin bilan bir xil sabab).
		if rn, identity, ok := publishedAudioTrack(event); ok {
			h.Room.EnforceAudioPolicy(c.Request.Context(), rn, identity)
		}

		if ei := event.EgressInfo; ei != nil {
			var objectKey string
			var durationSec int
			var sizeBytes int64
			if len(ei.FileResults) > 0 {
				f := ei.FileResults[0]
				objectKey = f.Filename
				durationSec = int(f.Duration / 1_000_000_000) // ns → s
				sizeBytes = f.Size
			}

			var herr error
			switch ei.Status {
			case livekit.EgressStatus_EGRESS_COMPLETE:
				herr = h.Recording.HandleEgress(c.Request.Context(), ei.EgressId, true, objectKey, durationSec, sizeBytes)
			case livekit.EgressStatus_EGRESS_LIMIT_REACHED:
				// LIMIT_REACHED = vaqt/hajm chegarasiga yetildi, LEKIN fayl YOZILDI
				// (audit topilma #5). Yaroqli fayl bo'lsa uni `completed` deb qabul
				// qilamiz — aks holda haqiqiy MP4 "failed" bo'lib MinIO'da yetim
				// qolib ketardi (retention faqat `ready` yozuvni tozalaydi).
				if sizeBytes > 0 && objectKey != "" {
					herr = h.Recording.HandleEgress(c.Request.Context(), ei.EgressId, true, objectKey, durationSec, sizeBytes)
				} else {
					herr = h.Recording.HandleEgress(c.Request.Context(), ei.EgressId, false, "", 0, 0)
				}
			case livekit.EgressStatus_EGRESS_FAILED,
				livekit.EgressStatus_EGRESS_ABORTED:
				herr = h.Recording.HandleEgress(c.Request.Context(), ei.EgressId, false, "", 0, 0)
			default:
				// ACTIVE / STARTING / ENDING — oraliq holatlar, e'tiborsiz.
			}
			// DB persistensiya uzilsa non-200 — LiveKit hodisani qayta yuboradi
			// (aks holda tayyor yozuv abadiy "processing"da qolardi).
			if herr != nil {
				hs.AbortError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "could not persist egress event")
				return
			}
		}

		c.Status(http.StatusOK)
	}
}

// recordingTriggerRoom — webhook hodisasi yozib olishni boshlashi kerakmi va
// qaysi xona uchun.
//
// Sof funksiya (ayrilgan sabab): bu KARORNING o'zi — qaysi hodisada yozuv
// boshlanadi — mahsulot qoidasi, va u imzolangan HTTP so'rovi yasamasdan
// sinalishi kerak. Handler'ning qolgani shunchaki simlash.
//
// `track_published` tanlanganining sababi `recording.UseCase.EnsureRecording`
// izohida: egress "start signal"ni ishtirokchi kirganda emas, trek e'lon
// qilinganda oladi.
func recordingTriggerRoom(event *livekit.WebhookEvent) (string, bool) {
	if event == nil || event.Event != webhook.EventTrackPublished {
		return "", false
	}
	if event.Room == nil || event.Room.Name == "" {
		return "", false
	}
	return event.Room.Name, true
}

// joinedParticipant — hodisa "ishtirokchi xonaga kirdi"mi va kim.
//
// `recordingTriggerRoom` bilan bir xil sababdan sof funksiya: qaysi hodisada
// ban qo'llanishi xavfsizlik qoidasi va u imzolangan HTTP so'rovisiz sinalishi
// kerak.
//
// Aynan `participant_joined`: bu SFU'ning ulanish qabul qilgani, ya'ni token
// qayerdan kelganidan qat'i nazar keladigan yagona ishonchli signal.
//
// name — ishtirokchining ko'rsatilgan ismi (tokenga imzolangan): mentor
// darajasidagi doimiy qora ro'yxat (№4) ism bo'yicha tekshiriladi. Bo'sh
// bo'lishi mumkin — u holda faqat identity-ban ishlaydi.
func joinedParticipant(event *livekit.WebhookEvent) (roomName, identity, name string, ok bool) {
	if event == nil || event.Event != webhook.EventParticipantJoined {
		return "", "", "", false
	}
	if event.Room == nil || event.Room.Name == "" {
		return "", "", "", false
	}
	if event.Participant == nil || event.Participant.Identity == "" {
		return "", "", "", false
	}
	return event.Room.Name, event.Participant.Identity, event.Participant.Name, true
}

// roomOccupancy — hodisa xona BAND/BO'SH ekanini bildiradimi.
//
// `recordingTriggerRoom` bilan bir xil sababdan sof funksiya: "qaysi hodisa
// xona bo'shaganini bildiradi" — mahsulot qoidasi (avto-yakun) va u imzolangan
// HTTP so'rovisiz sinalishi kerak.
//
// Hodisalar:
//   - `participant_left` → xonada qolgan ishtirokchilar soni 0 bo'lsa BO'SH;
//   - `participant_joined` → BAND (grace hisobi bekor qilinadi);
//   - `room_finished` → SFU xonani o'chirdi, ya'ni albatta BO'SH.
//
// ⚠️ `NumParticipants` LiveKit'ning suratidan olinadi va nazariy jihatdan
// eskirgan bo'lishi mumkin. Bu XAVFSIZ: ishchi darsni yakunlashdan oldin
// xonani LiveKit'dan qayta so'rab tekshiradi (`autoEndReason`), ya'ni yolg'on
// "bo'sh" signali faqat hisobni erta boshlaydi, noto'g'ri yakun yasamaydi.
func roomOccupancy(event *livekit.WebhookEvent) (roomName string, empty, ok bool) {
	if event == nil || event.Room == nil || event.Room.Name == "" {
		return "", false, false
	}
	switch event.Event {
	case webhook.EventParticipantLeft:
		return event.Room.Name, event.Room.NumParticipants == 0, true
	case webhook.EventParticipantJoined:
		return event.Room.Name, false, true
	case webhook.EventRoomFinished:
		return event.Room.Name, true, true
	default:
		return "", false, false
	}
}

// publishedAudioTrack — hodisa "AUDIO trek e'lon qilindi"mi, qaysi xonada va kim.
//
// Ovoz siyosati (mute_on_entry / allow_self_unmute) qaysi hodisada qo'llanishi —
// mahsulot qarori, shuning uchun `recordingTriggerRoom` kabi sof funksiya.
//
// Nega `participant_joined` emas: u paytda trek hali YO'Q — mute qiladigan
// narsaning o'zi bo'lmaydi. Egress-trigger'dan farqli bu yerda trek TURI ham
// tekshiriladi: kamera (video) publish'i ovoz siyosatiga daxlsiz.
func publishedAudioTrack(event *livekit.WebhookEvent) (roomName, identity string, ok bool) {
	if event == nil || event.Event != webhook.EventTrackPublished {
		return "", "", false
	}
	if event.Room == nil || event.Room.Name == "" {
		return "", "", false
	}
	if event.Participant == nil || event.Participant.Identity == "" {
		return "", "", false
	}
	if event.Track == nil || event.Track.Type != livekit.TrackType_AUDIO {
		return "", "", false
	}
	return event.Room.Name, event.Participant.Identity, true
}

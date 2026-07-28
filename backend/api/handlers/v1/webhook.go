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
			case livekit.EgressStatus_EGRESS_FAILED,
				livekit.EgressStatus_EGRESS_ABORTED,
				livekit.EgressStatus_EGRESS_LIMIT_REACHED:
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

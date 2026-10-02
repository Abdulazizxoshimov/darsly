package room

import (
	"context"
	"strconv"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// ── AVTO-YAKUN (PRODUCT.md «Dars hayoti») ───────────────────────────────────
//
// Ikki mustaqil qoida bor va ikkalasi ham SERVER tomonda bajarilishi shart —
// mentor ilovani yopib qo'yishi, telefoni o'chishi yoki internet uzilishi mumkin,
// ya'ni "Yakunlash" tugmasi bosilishiga umid qilib bo'lmaydi:
//
//  1. TEXNIK LIMIT: dars `started_at` dan `LESSON_MAX_DURATION` (default 4 soat)
//     o'tgach avtomatik yakunlanadi. Bu unutilgan darslar serverning egress/CPU
//     resurslarini kunlab yeb turishining oldini oladi.
//
//  2. BO'SH XONA: xona bo'shagach `LESSON_EMPTY_GRACE` (default 20 daqiqa)
//     kutiladi va hali ham bo'sh bo'lsa dars yakunlanadi. Grace kerak, chunki
//     mobil internet uzilishida mentor bir-ikki daqiqaga tushib qolishi odatiy
//     hol — darhol yakunlash uni darsdan mahrum qilardi (reopen YO'Q).
//
// ⚠️ LiveKit'ning `empty_timeout` bilan ARALASHTIRMASLIK kerak: u faqat SFU
// xonasini o'chiradi, bizning DB'dagi dars holatiga daxli yo'q. Bizga aynan
// DARS holati kerak (havola ishlamasin, yozuv yopilsin, ro'yxatda "tugagan"
// ko'rinsin).

// roomEmptyKey — xona qachondan beri bo'sh (unix soniya). Grace hisobi shu
// belgidan boshlanadi.
//
// Belgi ikki manbadan qo'yiladi:
//   - webhook (`participant_left` → xonada hech kim qolmadi, `room_finished`) —
//     ANIQ vaqt, ya'ni grace haqiqatan xona bo'shagan lahzadan boshlanadi;
//   - sweep'ning o'zi (webhook yetib kelmagan/yo'qolgan holat uchun) — bu
//     zaxira yo'l, grace keyingi tick'dan boshlanadi.
func roomEmptyKey(lessonID string) string { return "room:empty:" + lessonID }

// roomEmptyTTL — bo'shlik belgisining muddati. Eng uzun dars (4 soat) + grace'dan
// sezilarli uzun bo'lishi kerak, aks holda belgi hisob tugamasdan yo'qolib,
// grace qaytadan boshlanardi.
const roomEmptyTTL = 12 * time.Hour

// NoteRoomEmpty — LiveKit webhook'i: xonada ishtirokchi qolmadi.
// Belgi allaqachon bo'lsa TEGILMAYDI (SetNX) — birinchi bo'shash vaqti muhim,
// keyingi hodisalar hisobni qaytadan boshlab yubormasligi kerak.
func (uc *useCase) NoteRoomEmpty(ctx context.Context, rn string) {
	lessonID, ok := shared.LessonIDFromRoom(rn)
	if !ok || uc.cache == nil {
		return
	}
	uc.markEmptySince(ctx, lessonID, time.Now().UTC())
}

// NoteRoomOccupied — LiveKit webhook'i: xonaga kimdir kirdi (yoki hamon bor).
// Bo'shlik belgisi tozalanadi — grace hisobi bekor qilinadi.
func (uc *useCase) NoteRoomOccupied(ctx context.Context, rn string) {
	lessonID, ok := shared.LessonIDFromRoom(rn)
	if !ok || uc.cache == nil {
		return
	}
	_ = uc.cache.Del(ctx, roomEmptyKey(lessonID))
}

// markEmptySince bo'shlik belgisini (birinchi marta) qo'yadi va belgidagi
// vaqtni qaytaradi. Belgi bor bo'lsa o'shanisi qaytadi.
func (uc *useCase) markEmptySince(ctx context.Context, lessonID string, now time.Time) (time.Time, bool) {
	if uc.cache == nil {
		return time.Time{}, false
	}
	key := roomEmptyKey(lessonID)
	ok, err := uc.cache.SetNX(ctx, key, strconv.FormatInt(now.Unix(), 10), roomEmptyTTL)
	if err != nil {
		// Redis yetib bo'lmadi — hisobni boshlay olmaymiz. Darsni yakunlash
		// XAVFLIROQ bo'lardi (jonli darsni uzib yuborish), shuning uchun "bo'shlik
		// aniqlanmadi" deb qaytamiz.
		uc.log.Warn(ctx, "room.autoend: bo'shlik belgisini yozib bo'lmadi",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return time.Time{}, false
	}
	if ok {
		return now, true
	}
	raw, err := uc.cache.Get(ctx, key)
	if err != nil || raw == "" {
		return now, true // poyga: belgi shu orada yo'qoldi — hozirdan hisoblaymiz
	}
	sec, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return now, true
	}
	return time.Unix(sec, 0).UTC(), true
}

// SweepAutoEnd — jonli darslarni ko'rib chiqadi va yakunlash shartiga
// tushganlarini yakunlaydi. Yakunlangan darslar sonini qaytaradi.
//
// Fon ishchisi (`worker.AutoEndWorker`) uni davriy chaqiradi. Idempotent va
// ko'p-instansga xavfsiz: yakunlash `lessonRepo.ClaimEnd` (atomik CAS) orqali,
// ya'ni qayta ishga tushirilganda yoki ikki instansda dublikat yakun bo'lmaydi.
func (uc *useCase) SweepAutoEnd(ctx context.Context, maxDuration, emptyGrace time.Duration) int {
	lessons, err := uc.lessonRepo.ListLive(ctx)
	if err != nil {
		uc.log.Warn(ctx, "room.SweepAutoEnd: jonli darslarni o'qib bo'lmadi", logger.SafeString("err", err.Error()))
		return 0
	}
	now := time.Now().UTC()
	// Eskirgan `recording` yozuvlar (telefon complete qilmagan / webhook yo'qolgan):
	// dars max davomiyligidan + 10 daqiqa o'tgach failed.
	if maxDuration > 0 && uc.recorder != nil {
		uc.recorder.ReapStaleRecordings(ctx, now.Add(-(maxDuration + 10*time.Minute)))
	}
	ended := 0
	for _, l := range lessons {
		reason, ok := uc.autoEndReason(ctx, l, now, maxDuration, emptyGrace)
		if !ok {
			continue
		}
		if uc.autoEnd(ctx, l, now, reason) {
			ended++
		}
	}
	return ended
}

// autoEndReason — dars yakunlanishi kerakmi va nega ("max_duration" | "empty_room").
func (uc *useCase) autoEndReason(
	ctx context.Context, l *entity.Lesson, now time.Time, maxDuration, emptyGrace time.Duration,
) (string, bool) {
	// 1) Texnik limit. `started_at` bo'lmasa (eski qator yoki qo'lda `live`
	//    qilingan dars) `updated_at` ga tayanamiz — limitsiz qolib ketmasin.
	started := l.StartedAt
	if started == nil {
		started = &l.UpdatedAt
	}
	if maxDuration > 0 && now.Sub(*started) >= maxDuration {
		return "max_duration", true
	}

	// 2) Bo'sh xona. LiveKit o'chirilgan bo'lsa (dev muhiti) bu qoida
	//    QO'LLANMAYDI: xonada kim borligini bilishning yo'li yo'q va har jonli
	//    darsni bo'sh deb yakunlash falokat bo'lardi.
	if emptyGrace <= 0 || !uc.livekit.Enabled() {
		return "", false
	}
	// ⚠️ Bu tekshiruv XONA O'CHIRILGAN holatda ham to'g'ri ishlashi SHART: LiveKit
	// o'zining `empty_timeout` (5 daq) bo'yicha bo'sh xonani o'chiradi va shundan
	// keyin bizning grace hali tugamagan bo'ladi. Empirik tekshirildi (2026-07-31,
	// dev SFU): mavjud bo'lmagan xona uchun `ListParticipants` XATO EMAS, bo'sh
	// ro'yxat qaytaradi — ya'ni "xona yo'q" = "bo'sh" deb to'g'ri baholanadi.
	// Agar bu xulq o'zgarsa (xato qaytsa) bo'sh xona qoidasi jimgina ishlamay
	// qo'yadi va dars faqat 4 soatlik limit bilan yakunlanadi.
	listCtx, cancel := lkCtx(ctx)
	parts, err := uc.livekit.ListParticipantViews(listCtx, roomName(l.ID))
	cancel()
	if err != nil {
		// SFU javob bermadi — bo'shlikni TASDIQLAY olmaymiz. Jonli darsni
		// noto'g'ri uzib yuborishdan ko'ra keyingi tick'ni kutish arzon.
		uc.log.Warn(ctx, "room.SweepAutoEnd: xona holatini o'qib bo'lmadi",
			logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		return "", false
	}
	if len(parts) > 0 {
		// Xonada odam bor — webhook noto'g'ri qo'ygan bo'lishi mumkin bo'lgan
		// belgini tozalaymiz (yolg'on-pozitiv butunlay yopiladi).
		if uc.cache != nil {
			_ = uc.cache.Del(ctx, roomEmptyKey(l.ID))
		}
		// ⭐ YOZUV RECONCILIATION (system-design audit R3).
		//
		// Odatda yozuv `track_published` webhook'idan boshlanadi. Agar o'sha
		// webhook yo'qolsa (deploy paytida, LiveKit 429/retry tugashi) dars
		// JIMGINA yozilmay qolardi va buni hech kim sezmasdi. Sweep har tick'da
		// jonli darsni ko'radi: yozuv yoqilgan, xonada HAQIQIY media bor
		// (unmuted trek), lekin yozuv ketmayapti bo'lsa — o'zi boshlaydi.
		// `EnsureRecording` idempotent va cap/lock/local-mode'ni o'zi tekshiradi.
		uc.reconcileRecording(ctx, l, parts)
		return "", false
	}
	since, ok := uc.markEmptySince(ctx, l.ID, now)
	if !ok {
		return "", false
	}
	if now.Sub(since) >= emptyGrace {
		return "empty_room", true
	}
	return "", false
}

// reconcileRecording — yo'qolgan `track_published` webhook'i o'rnini bosadi:
// jonli dars, yozuv yoqilgan, xonada HAQIQIY media (unmuted trek) bor, lekin
// yozuv ketmayapti bo'lsa — yozuvni boshlaydi (audit R3).
//
// Faqat UNMUTED trek borligini talab qiladi: `EnsureRecording` egressni media
// paydo bo'lganda ishga tushiradi ("5 daqiqalik tuzoq" — recording usecase
// izohi), muted-only xonada egress bo'sh Chrome aylantirib bekor bo'lardi.
// `EnsureRecording` idempotent: cap/lock/local-mode/yozuv yoqilganini o'zi
// tekshiradi, shuning uchun bu yerda faqat arzon oldingi tekshiruvlar.
func (uc *useCase) reconcileRecording(ctx context.Context, l *entity.Lesson, parts []entity.RoomParticipant) {
	if uc.recorder == nil || !l.IsRecordingEnabled {
		return
	}
	if uc.recorder.IsRecording(ctx, l.ID) {
		return
	}
	hasMedia := false
	for _, p := range parts {
		if !p.AudioMuted || !p.VideoMuted {
			hasMedia = true
			break
		}
	}
	if !hasMedia {
		return
	}
	if err := uc.recorder.EnsureRecording(ctx, l.ID); err != nil {
		uc.log.Warn(ctx, "room.reconcileRecording: yozuvni boshlab bo'lmadi",
			logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		return
	}
	uc.log.Info(ctx, "room.reconcileRecording: yo'qolgan webhook o'rniga yozuv boshlandi",
		logger.String("lesson_id", l.ID))
}

// autoEnd — darsni atomik yakunlaydi va tozalashni bajaradi.
// G'olib bo'lmasak (boshqa instans yoki mentor allaqachon yakunlagan) false.
func (uc *useCase) autoEnd(ctx context.Context, l *entity.Lesson, now time.Time, reason string) bool {
	claimed, err := uc.lessonRepo.ClaimEnd(ctx, l.ID, now)
	if err != nil {
		uc.log.Warn(ctx, "room.SweepAutoEnd: yakunlashni band qilib bo'lmadi",
			logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		return false
	}
	if !claimed {
		return false // dars allaqachon yakunlangan
	}
	uc.teardown(ctx, l.ID)
	uc.log.Info(ctx, "room: dars avtomatik yakunlandi",
		logger.String("lesson_id", l.ID), logger.String("reason", reason))
	return true
}

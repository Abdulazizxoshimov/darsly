package shared

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// Darsdan chiqarilgan ("kick" qilingan) ishtirokchi reyestri.
//
// # Nega alohida, umumiy paket
//
// LiveKit'ning `RemoveParticipant` chaqiruvi faqat JORIY ulanishni uzadi —
// ishtirokchining tokeni qo'lida qoladi va TTL tugagunicha yaroqli. Ya'ni
// "chiqarib yuborish" o'z-o'zidan qaytib kirishni to'smaydi; ban ro'yxati
// LiveKit'da yo'q va `room.auto_create` yoqilgan.
//
// Avval bu ro'yxat `usecase/room` ichida yopiq edi va faqat TOKEN BERISHDA
// tekshirilardi. Natijada chiqarilgan buzg'unchi backendga umuman murojaat
// qilmasdan (`room.connect(wsUrl, eskiToken)`) qaytib kelardi va chat, qo'l
// ko'tarish, so'rovnoma endpointlari uni bemalol qabul qilaverardi — ustozning
// "chiqarish" tugmasi amalda kuchsiz edi.
//
// Shuning uchun ban tekshiruvi xonaga tegishli BARCHA usecase'larga kerak
// (room, roomstate, chat, poll) va bu yerda — ular umumiy ko'radigan joyda —
// turadi.
//
// Himoya ikki qatlamli:
//  1. Qisqa ishtirokchi tokeni (`livekit.participantTokenTTLMax`) — eski token
//     bilan qaytib ulanish oynasini cheklaydi.
//  2. `participant_joined` webhook'i va har bir ochiq endpoint — shu reyestrni
//     tekshiradi.

// BanTTL — chiqarilgan ishtirokchi shuncha vaqt qayta kira olmaydi.
//
// Dars uzunligidan kattaroq: dars tugagach ban ham keraksiz (yangi darsda yangi
// xona, yangi token). Redis kalitni o'zi tozalaydi.
const BanTTL = 6 * time.Hour

// BanKey — Redis kaliti. Format `usecase/room` dagi eski qiymat bilan bir xil
// (ishlab turgan darsdagi banlar deploy'dan keyin ham amalda qoladi).
func BanKey(lessonID, identity string) string { return "room:ban:" + lessonID + ":" + identity }

// Ban — ishtirokchini darsdan chiqarilganlar ro'yxatiga qo'shadi.
func Ban(ctx context.Context, cache redis.Cache, lessonID, identity string) error {
	if cache == nil || identity == "" {
		return nil
	}
	return cache.Set(ctx, BanKey(lessonID, identity), "1", BanTTL)
}

// IsBanned — ishtirokchi shu darsdan chiqarilganmi.
//
// Redis yetib bo'lmasa `false` qaytadi (fail-open): ban — moderatsiya qulayligi,
// autentifikatsiya emas. Redis uzilganda butun darsga kirishni to'sib qo'yish
// zarari chiqarilgan bitta kishining qaytib kirishidan ancha katta.
func IsBanned(ctx context.Context, cache redis.Cache, lessonID, identity string) bool {
	if cache == nil || identity == "" {
		return false
	}
	v, err := cache.Get(ctx, BanKey(lessonID, identity))
	return err == nil && v != ""
}

// ErrBanned — chiqarilgan ishtirokchi amal bajarishga urinsa qaytadigan xato.
// 403: token yaroqli (401 emas), lekin bu odamga ruxsat yo'q.
func ErrBanned() error {
	return apperr.Forbidden("you have been removed from this lesson")
}

// liveKey — "bu dars hozir jonli" belgisi (faqat IJOBIY holat keshlanadi).
func liveKey(lessonID string) string { return "room:live:" + lessonID }

// liveTTL — jonli belgi keshi. Qisqa, chunki u xavfsizlik qarori emas, tezlik
// optimizatsiyasi: dars yakunlanganda `InvalidateLive` kalitni darhol o'chiradi,
// TTL esa faqat o'sha o'chirish uzilgan holat uchun tarmoq (fail-safe).
const liveTTL = 60 * time.Second

// InvalidateLive — dars yakunlanganda jonli keshni tozalaydi.
func InvalidateLive(ctx context.Context, cache redis.Cache, lessonID string) {
	if cache != nil {
		_ = cache.Del(ctx, liveKey(lessonID))
	}
}

// isLive — dars jonlimi, Redis keshi bilan.
//
// # Nega kesh
// Bu tekshiruv reaksiya va qo'l ko'tarish yo'llariga qo'shildi — ular ilgari DB'ga
// UMUMAN bormasdi (faqat Redis + LiveKit broadcast). Webinar modelida (1000+
// o'quvchi) har reaksiyaga bitta `SELECT` qo'shish sekundiga yuzlab ortiqcha
// so'rov degani. Shuning uchun natija keshlanadi — `joinlink.mentorName` bilan
// bir xil naqsh.
//
// Faqat IJOBIY holat keshlanadi: "jonli emas" javobi keshlansa, dars boshlangan
// paytda o'quvchilar TTL tugagunicha rad etilardi. Aksincha, "jonli" holatning
// eskirishi `InvalidateLive` bilan yopiladi.
func isLive(ctx context.Context, repo repository.LessonRepository, cache redis.Cache, lessonID string) (bool, error) {
	if cache != nil {
		if v, err := cache.Get(ctx, liveKey(lessonID)); err == nil && v != "" {
			return true, nil
		}
	}
	l, err := repo.GetByID(ctx, lessonID)
	if err != nil {
		return false, err
	}
	if l.Status != entity.LessonStatusLive {
		return false, nil
	}
	if cache != nil {
		_ = cache.Set(ctx, liveKey(lessonID), "1", liveTTL)
	}
	return true, nil
}

// GuardRoomAction — xona ichidagi OCHIQ amallar (chat, qo'l, reaksiya, ovoz
// berish) uchun yagona kirish tekshiruvi: dars jonli bo'lishi va ishtirokchi
// chiqarilmagan bo'lishi shart.
//
// Ikkalasi birga tekshiriladi, chunki ikkalasi ham bitta savolga javob beradi:
// "bu odam hozir shu xonada amal bajara oladimi?". Yakunlangan darsga chat
// yozish yoki ovoz berish ham xuddi ban kabi ma'nosiz — xona yopilgan, lekin
// endpoint ochiq qolgan edi.
//
// Tartib ataylab: avval ban (Redis, arzon va xavfsizlik uchun asosiysi), keyin
// dars holati (keshlangan).
func GuardRoomAction(
	ctx context.Context,
	repo repository.LessonRepository,
	cache redis.Cache,
	lessonID, identity string,
) error {
	if err := ValidateID(lessonID, "lesson"); err != nil {
		return err
	}
	if identity == "" {
		return apperr.BadRequest("identity is required")
	}
	if IsBanned(ctx, cache, lessonID, identity) {
		return ErrBanned()
	}
	live, err := isLive(ctx, repo, cache, lessonID)
	if err != nil {
		return err
	}
	if !live {
		return apperr.BadRequest("lesson is not live")
	}
	return nil
}

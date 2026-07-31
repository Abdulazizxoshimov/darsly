// Package roomstate — dars xonasining o'tkinchi holati: qo'l ko'tarish va reaksiyalar.
package roomstate

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// Qo'llar Redis hash'ida yashaydi: `room:hands:<lessonID>` → identity → handEntry(JSON).
// TTL — eng uzun darsdan ancha katta, lekin abadiy emas: dars nosoz tugasa
// (server qulasa) kalit o'z-o'zidan yo'qoladi va Redis axlat yig'maydi.
const handsTTL = 12 * time.Hour

// Reaksiya uchun tezlik cheklovi: bitta ishtirokchi 10 soniyada 5 marta.
//
// Avval 2 soniyada 1 ta edi va bu HAQIQIY foydalanishni buzardi: qarsak
// (👏👏👏) tabiiy ravishda ketma-ket bosiladi va ikkinchi bosishdayoq 429
// olardi — o'quvchi uchun "tugma ishlamadi" degani. O'rtacha tezlik bir xil
// (10 s da 5 ta), lekin qisqa portlashga ruxsat beriladi.
//
// Klient tomonda ham cheklov bor (`roomLogic.rateLimiter`), lekin u faqat halol
// klientni to'xtatadi — server tomondagisi esa haqiqiy himoya.
const (
	reactionWindow = 10 * time.Second
	reactionMax    = 5
)

// allowedReactions — RUXSAT ETILGAN emoji to'plami.
//
// # Nega server ham tekshiradi
//
// Reaksiya klient tanlovidan keladi, lekin endpoint ochiq: `curl` bilan
// `{"emoji":"<istalgan 16 bayt>"}` yuborib bo'lardi va u butun xona ekranida
// suzib o'tardi — ya'ni moderatsiyasiz matn kanali. Uzunlik cheklovi (16 bayt)
// buni to'smaydi: haqorat qisqa bo'ladi.
//
// To'plam klientlardagi ro'yxat bilan bir xil
// (`frontend/src/livekit/Controls.jsx: REACTIONS`). Klientga ro'yxat
// SERVERDAN berilmaydi (mahsulot qarori) — u konstanta, shuning uchun bu
// yerdagi to'plam klientnikining ustidan qo'yiladigan qopqoq bo'lib qoladi:
// yangi emoji qo'shilsa IKKALA joyga ham qo'shiladi.
var allowedReactions = map[string]bool{
	"👍":  true, // like
	"👏":  true, // qarsak
	"❤️": true, // yurak (U+2764 U+FE0F — variatsiya selektori bilan)
	"😂":  true,
	"😮":  true,
	"🎉":  true,
	"✋":  true, // "savolim bor" belgisi (qo'l ko'tarishdan farqli — o'tkinchi)
}

// handEntry — Redis'da saqlanadigan yozuv. `At` — Unix millisekund: klient
// data-channel'dan keladigan `at` maydoni bilan bir xil birlikda bo'lsin.
type handEntry struct {
	Name string `json:"n"`
	At   int64  `json:"t"`
}

type useCase struct {
	lessonRepo repository.LessonRepository
	livekit    LiveKit
	cache      redis.Cache
	// recorder — ixtiyoriy (nil bo'lishi mumkin): yozuv indikatori uchun.
	recorder Recorder
	log      logger.Logger
}

func New(
	lessonRepo repository.LessonRepository,
	lk LiveKit,
	cache redis.Cache,
	recorder Recorder,
	log logger.Logger,
) UseCase {
	return &useCase{lessonRepo: lessonRepo, livekit: lk, cache: cache, recorder: recorder, log: log}
}

func handsKey(lessonID string) string     { return "room:hands:" + lessonID }
func reactKey(lessonID, id string) string { return "rl:react:" + lessonID + ":" + id }

// broadcast — xabarni LiveKit data-channel orqali xonaga yuboradi.
//
// WebSocket EMAS: guest'da JWT yo'q va u Hub'ga ulanmaydi, LiveKit esa allaqachon
// ulangan va kechikishi pastroq. Xabar shakli klient kutayotgani bilan aynan mos
// (`frontend/src/livekit/messaging.js`).
func (uc *useCase) broadcast(ctx context.Context, lessonID string, msg any) {
	if uc.livekit == nil || !uc.livekit.Enabled() {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error(ctx, "roomstate: marshal failed", logger.SafeString("err", err.Error()))
		return
	}
	// Tarqatish muvaffaqiyatsiz bo'lsa ham amal BEKOR QILINMAYDI: holat serverda
	// saqlangan va kech kirgan klient uni `State` orqali baribir oladi.
	if err := uc.livekit.SendData(ctx, shared.RoomName(lessonID), data); err != nil {
		uc.log.Warn(ctx, "roomstate: broadcast failed",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
	}
}

// handMsg — klient (`messaging.js`) kutayotgan xabar shakli.
func handMsg(identity, name string, raised bool, at int64) map[string]any {
	return map[string]any{"kind": "hand", "identity": identity, "name": name, "raised": raised, "at": at}
}

func (uc *useCase) SetHand(ctx context.Context, lessonID, identity, name string, raised bool) error {
	// Dars jonli va ishtirokchi chiqarilmagan bo'lishi shart (`shared.GuardRoomAction`).
	// Busiz chiqarilgan buzg'unchi eski tokeni bilan qo'l ko'tarib ustozning
	// navbat ro'yxatini to'ldirib tashlay olardi.
	if err := shared.GuardRoomAction(ctx, uc.lessonRepo, uc.cache, lessonID, identity); err != nil {
		return err
	}
	if name == "" {
		name = identity
	}

	at := time.Now().UTC().UnixMilli()
	if raised {
		// TAKRORIY ko'tarish navbatdagi o'rinni buzmasligi kerak: agar qo'l
		// allaqachon ko'tarilgan bo'lsa, eski vaqt saqlanadi. Aks holda o'quvchi
		// tugmani ikki marta bosib navbatda oldinga (yoki orqaga) siljib qolardi.
		if cur, err := uc.entry(ctx, lessonID, identity); err == nil && cur != nil {
			at = cur.At
		}
		b, _ := json.Marshal(handEntry{Name: name, At: at})
		if err := uc.cache.HSet(ctx, handsKey(lessonID), map[string]any{identity: string(b)}, handsTTL); err != nil {
			uc.log.Error(ctx, "roomstate.SetHand: redis error", logger.SafeString("err", err.Error()))
			return apperr.Internal(err)
		}
	} else if err := uc.cache.HDel(ctx, handsKey(lessonID), identity); err != nil {
		uc.log.Error(ctx, "roomstate.SetHand: redis del error", logger.SafeString("err", err.Error()))
		return apperr.Internal(err)
	}

	uc.broadcast(ctx, lessonID, handMsg(identity, name, raised, at))
	return nil
}

func (uc *useCase) LowerHand(ctx context.Context, mentorID, lessonID, identity string) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	if identity == "" {
		return apperr.BadRequest("identity is required")
	}
	if err := uc.cache.HDel(ctx, handsKey(lessonID), identity); err != nil {
		return apperr.Internal(err)
	}
	uc.broadcast(ctx, lessonID, handMsg(identity, "", false, time.Now().UTC().UnixMilli()))
	uc.log.Info(ctx, "roomstate: hand lowered by host",
		logger.String("lesson_id", lessonID), logger.String("identity", identity))
	return nil
}

func (uc *useCase) LowerAll(ctx context.Context, mentorID, lessonID string) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	if err := uc.cache.Del(ctx, handsKey(lessonID)); err != nil {
		return apperr.Internal(err)
	}
	uc.broadcast(ctx, lessonID, map[string]any{"kind": "hand", "act": "lower_all"})
	uc.log.Info(ctx, "roomstate: all hands lowered", logger.String("lesson_id", lessonID))
	return nil
}

func (uc *useCase) State(ctx context.Context, lessonID string) (*entity.RoomState, error) {
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return nil, err
	}
	// Yozuv indikatori — ishtirokchi o'zi yozilayotganini KO'RISHI kerak.
	rec := uc.recorder != nil && uc.recorder.IsRecording(ctx, lessonID)
	muteOnEntry, allowSelfUnmute := uc.audioPolicy(ctx, lessonID)

	raw, err := uc.cache.HGetAll(ctx, handsKey(lessonID))
	if err != nil {
		// Kalit yo'q — bu XATO EMAS, shunchaki hech kim qo'l ko'tarmagan.
		// (redisCache.HGetAll bo'sh hash uchun ham xato qaytaradi.)
		return &entity.RoomState{
			Hands: []entity.RaisedHand{}, Recording: rec,
			MuteOnEntry: muteOnEntry, AllowSelfUnmute: allowSelfUnmute,
		}, nil
	}

	hands := make([]entity.RaisedHand, 0, len(raw))
	for identity, v := range raw {
		var e handEntry
		if json.Unmarshal([]byte(v), &e) != nil {
			continue // buzuq yozuv butun ro'yxatni yiqitmasin
		}
		hands = append(hands, entity.RaisedHand{
			Identity: identity,
			Name:     e.Name,
			RaisedAt: time.UnixMilli(e.At).UTC(),
		})
	}
	// Navbat tartibi SERVERDA hisoblanadi (hash tartibsiz qaytaradi).
	// Teng vaqtda identity bo'yicha — tartib determinial bo'lsin.
	sort.Slice(hands, func(i, j int) bool {
		if hands[i].RaisedAt.Equal(hands[j].RaisedAt) {
			return hands[i].Identity < hands[j].Identity
		}
		return hands[i].RaisedAt.Before(hands[j].RaisedAt)
	})
	return &entity.RoomState{
		Hands: hands, Recording: rec,
		MuteOnEntry: muteOnEntry, AllowSelfUnmute: allowSelfUnmute,
	}, nil
}

// audioPolicy — darsning ovoz siyosati (`entity.RoomState` izohiga qara).
//
// Dars o'qib bo'lmasa RUXSAT BERUVCHI qiymatlar qaytadi (`allow_self_unmute=true`).
// Sabab: bu maydonlar faqat UI ko'rsatkichi, chegara emas. DB nosozligida
// o'quvchining mikrofon tugmasini o'chirib qo'yish — mavjud bo'lmagan taqiqni
// ko'rsatish, ya'ni real cheklovsiz real zarar. Teskari xato (tugma yoniq, server
// mute qiladi) esa mavjud xulqdan yomonroq emas.
func (uc *useCase) audioPolicy(ctx context.Context, lessonID string) (muteOnEntry, allowSelfUnmute bool) {
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil || l == nil {
		uc.log.Warn(ctx, "roomstate.State: dars o'qilmadi, ovoz siyosati default'da",
			logger.String("lesson_id", lessonID))
		return false, true
	}
	return l.MuteOnEntry, l.AllowSelfUnmute
}

func (uc *useCase) Reaction(ctx context.Context, lessonID, identity, name, emoji string) error {
	// Chiqarilgan ishtirokchi emoji bilan ham xonani buza olmasin (`SetHand` bilan izchil).
	if err := shared.GuardRoomAction(ctx, uc.lessonRepo, uc.cache, lessonID, identity); err != nil {
		return err
	}
	// Ruxsat etilgan to'plam (`allowedReactions` izohiga qara) — bu tekshiruvsiz
	// reaksiya kanali moderatsiyasiz matn kanaliga aylanardi.
	if !allowedReactions[emoji] {
		return apperr.BadRequest("unsupported reaction")
	}
	// Server tomondagi tezlik cheklovi. Redis yiqilsa (xato) — reaksiyani
	// BLOKLAMAYMIZ: emoji darsning kritik funksiyasi emas, lekin uni to'xtatib
	// qo'yish foydalanuvchi uchun tushunarsiz "hech nima ishlamayapti" bo'lardi.
	n, err := uc.cache.Incr(ctx, reactKey(lessonID, identity), reactionWindow)
	if err == nil && n > reactionMax {
		return apperr.TooManyRequests("too many reactions — slow down")
	}
	if name == "" {
		name = identity
	}
	// `identity` — klient o'z reaksiyasini ikki marta ko'rmasligi uchun: u optimistik
	// tarzda darhol ko'rsatadi, keyin serverdan qaytgan echo'ni o'tkazib yuboradi.
	uc.broadcast(ctx, lessonID, map[string]any{"kind": "reaction", "emoji": emoji, "name": name, "identity": identity})
	return nil
}

func (uc *useCase) Clear(ctx context.Context, lessonID string) error {
	return uc.cache.Del(ctx, handsKey(lessonID))
}

// entry — bitta ishtirokchining joriy yozuvi (yo'q bo'lsa nil, xatosiz).
func (uc *useCase) entry(ctx context.Context, lessonID, identity string) (*handEntry, error) {
	raw, err := uc.cache.HGetAll(ctx, handsKey(lessonID))
	if err != nil {
		return nil, err
	}
	v, ok := raw[identity]
	if !ok {
		return nil, nil
	}
	var e handEntry
	if err := json.Unmarshal([]byte(v), &e); err != nil {
		return nil, err
	}
	return &e, nil
}

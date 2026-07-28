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

// Reaksiya uchun tezlik cheklovi: bitta ishtirokchi 2 soniyada 1 marta.
// Klient tomonda ham cheklov bor (`roomLogic.rateLimiter`), lekin u faqat halol
// klientni to'xtatadi — server tomondagisi esa haqiqiy himoya.
const (
	reactionWindow = 2 * time.Second
	reactionMax    = 1
)

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
	log        logger.Logger
}

func New(lessonRepo repository.LessonRepository, lk LiveKit, cache redis.Cache, log logger.Logger) UseCase {
	return &useCase{lessonRepo: lessonRepo, livekit: lk, cache: cache, log: log}
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
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return err
	}
	if identity == "" {
		return apperr.BadRequest("identity is required")
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
	raw, err := uc.cache.HGetAll(ctx, handsKey(lessonID))
	if err != nil {
		// Kalit yo'q — bu XATO EMAS, shunchaki hech kim qo'l ko'tarmagan.
		// (redisCache.HGetAll bo'sh hash uchun ham xato qaytaradi.)
		return &entity.RoomState{Hands: []entity.RaisedHand{}}, nil
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
	return &entity.RoomState{Hands: hands}, nil
}

func (uc *useCase) Reaction(ctx context.Context, lessonID, identity, name, emoji string) error {
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return err
	}
	if identity == "" {
		return apperr.BadRequest("identity is required")
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

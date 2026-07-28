package chat

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// Tezlik cheklovi: bitta ishtirokchi 5 soniyada 5 ta xabar.
//
// Ustozga TEGMAYDI (u JWT bilan keladi va o'z darsining egasi) — ustozning
// tez-tez yozishi normal, buzg'unchilik esa xonadagi noma'lum ishtirokchidan
// keladi. Klient tomonda ham cheklov bor, lekin u faqat halol klientni to'xtatadi.
const (
	chatWindow = 5 * time.Second
	chatMax    = 5
)

type useCase struct {
	repo       repository.ChatRepository
	lessonRepo repository.LessonRepository
	userRepo   repository.UserRepository
	livekit    LiveKit
	cache      redis.Cache
	log        logger.Logger
}

func New(
	repo repository.ChatRepository,
	lessonRepo repository.LessonRepository,
	userRepo repository.UserRepository,
	lk LiveKit,
	cache redis.Cache,
	log logger.Logger,
) UseCase {
	return &useCase{repo: repo, lessonRepo: lessonRepo, userRepo: userRepo, livekit: lk, cache: cache, log: log}
}

func chatRateKey(lessonID, identity string) string { return "rl:chat:" + lessonID + ":" + identity }

// deliver — xabarni tarqatadi. Shaxsiy bo'lsa FAQAT ikki tomonga.
//
// Yetkazish xatosi amalni bekor qilmaydi: xabar DB'da saqlangan va tarixdan
// baribir keladi. Aksincha bo'lsa — saqlangan, lekin "yuborilmadi" deb xato
// qaytarilgan xabar foydalanuvchini ikki marta yozishga majbur qilardi.
func (uc *useCase) deliver(ctx context.Context, msg *entity.ChatMessage) {
	if uc.livekit == nil || !uc.livekit.Enabled() {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error(ctx, "chat: marshal failed", logger.SafeString("err", err.Error()))
		return
	}
	room := shared.RoomName(msg.LessonID)

	var derr error
	if msg.ToIdentity != nil {
		// Ikkala tomon ham oladi: qabul qiluvchi — xabarni, yuboruvchi — o'z
		// yozganini boshqa qurilmada/qayta ulanganda ko'rish uchun.
		derr = uc.livekit.SendDataTo(ctx, room, data, []string{*msg.ToIdentity, msg.SenderIdentity})
	} else {
		derr = uc.livekit.SendData(ctx, room, data)
	}
	if derr != nil {
		uc.log.Warn(ctx, "chat: broadcast failed",
			logger.String("lesson_id", msg.LessonID), logger.SafeString("err", derr.Error()))
	}
}

// build — xabar obyektini yasaydi va saqlaydi.
func (uc *useCase) build(ctx context.Context, lessonID, identity, name, body, to string) (*entity.ChatMessage, error) {
	msg := &entity.ChatMessage{
		ID:             uuid.NewString(),
		LessonID:       lessonID,
		SenderIdentity: identity,
		SenderName:     name,
		Body:           body,
		CreatedAt:      time.Now().UTC(),
	}
	if to != "" && to != identity { // o'ziga yozish ma'nosiz — ommaviy deb qaraymiz
		msg.ToIdentity = &to
	}
	if err := uc.repo.Create(ctx, msg); err != nil {
		uc.log.Error(ctx, "chat: db error",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil, err
	}
	uc.deliver(ctx, msg)
	return msg, nil
}

// ─── Host yo'li (JWT + egalik) ────────────────────────────────────────────────

func (uc *useCase) Send(ctx context.Context, mentorID, lessonID, body, to string) (*entity.ChatMessage, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	senderName := "Host"
	if m, err := uc.userRepo.GetByID(ctx, mentorID); err == nil {
		senderName = m.FullName
	}
	msg, err := uc.build(ctx, lessonID, mentorID, senderName, body, to)
	if err != nil {
		return nil, err
	}
	uc.log.Info(ctx, "chat message sent (host)", logger.String("lesson_id", lessonID))
	return msg, nil
}

func (uc *useCase) History(ctx context.Context, mentorID, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	// Host identity'si = mentorID (`room.HostToken` shunday beradi), ya'ni ustoz
	// o'zining shaxsiy yozishmalarini ko'radi, boshqalarnikini emas. Ustoz
	// darsning egasi bo'lsa-da, o'quvchilarning bir-biriga yozgani unga tegishli
	// emas — bu maxfiylik qarori, texnik cheklov emas.
	return uc.repo.ListByLesson(ctx, lessonID, mentorID, before, limit)
}

// ─── Xona yo'li (LiveKit room-token) ──────────────────────────────────────────

func (uc *useCase) SendFromRoom(ctx context.Context, lessonID, identity, name, body, to string) (*entity.ChatMessage, error) {
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return nil, err
	}
	if identity == "" {
		return nil, apperr.BadRequest("identity is required")
	}
	if uc.cache != nil {
		n, err := uc.cache.Incr(ctx, chatRateKey(lessonID, identity), chatWindow)
		if err == nil && n > chatMax {
			return nil, apperr.TooManyRequests("too many messages — slow down")
		}
	}
	if name == "" {
		name = identity
	}
	msg, err := uc.build(ctx, lessonID, identity, name, body, to)
	if err != nil {
		return nil, err
	}
	uc.log.Info(ctx, "chat message sent (participant)", logger.String("lesson_id", lessonID))
	return msg, nil
}

func (uc *useCase) HistoryForRoom(ctx context.Context, lessonID, identity string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return nil, err
	}
	if identity == "" {
		return nil, apperr.BadRequest("identity is required")
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	return uc.repo.ListByLesson(ctx, lessonID, identity, before, limit)
}

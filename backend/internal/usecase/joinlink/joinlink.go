package joinlink

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/waitingroom"
)

// Passcode brute-force himoyasi.
const (
	maxPasscodeAttempts  = 5
	passcodeFailWindow   = 10 * time.Minute
	passcodeLockCooldown = 5 * time.Minute
)

type useCase struct {
	lessonRepo  repository.LessonRepository
	userRepo    repository.UserRepository
	hasher      hasher.Hasher
	cache       redis.Cache
	room        room.UseCase
	waitingRoom waitingroom.UseCase
	log         logger.Logger
}

func New(lessonRepo repository.LessonRepository, userRepo repository.UserRepository, h hasher.Hasher, cache redis.Cache, roomUC room.UseCase, waitingUC waitingroom.UseCase, log logger.Logger) UseCase {
	return &useCase{lessonRepo: lessonRepo, userRepo: userRepo, hasher: h, cache: cache, room: roomUC, waitingRoom: waitingUC, log: log}
}

func (uc *useCase) Preview(ctx context.Context, slug string) (*entity.LessonPublic, error) {
	l, err := uc.resolveActive(ctx, slug)
	if err != nil {
		return nil, err
	}
	return uc.toPublic(ctx, l), nil
}

func (uc *useCase) Join(ctx context.Context, slug string, req *entity.JoinLessonReq) (*entity.JoinLessonResp, error) {
	l, err := uc.resolveActive(ctx, slug)
	if err != nil {
		return nil, err
	}

	// Qulflangan dars — yangi ishtirokchilar kira olmaydi.
	if l.IsLocked {
		return nil, apperr.Forbidden("lesson is locked by the host")
	}

	// Parol tekshiruvi (brute-force lockout bilan).
	if l.PasscodeHash != nil {
		if locked, _ := uc.cache.Get(ctx, lockKey(l.JoinSlug)); locked != "" {
			uc.log.Warn(ctx, "joinlink.Join: slug locked (brute-force)", logger.String("lesson_id", l.ID))
			return nil, apperr.Forbidden("too many failed attempts, try again later")
		}
		if req.Passcode == nil || *req.Passcode == "" {
			return nil, apperr.Unauthorized("passcode required")
		}
		if !uc.hasher.Check(*req.Passcode, *l.PasscodeHash) {
			n, _ := uc.cache.Incr(ctx, failKey(l.JoinSlug), passcodeFailWindow)
			if n >= maxPasscodeAttempts {
				_ = uc.cache.Set(ctx, lockKey(l.JoinSlug), "1", passcodeLockCooldown)
				uc.log.Warn(ctx, "joinlink.Join: passcode locked after too many attempts", logger.String("lesson_id", l.ID))
			}
			return nil, apperr.Unauthorized("invalid passcode")
		}
		// Muvaffaqiyat — hisoblagichni tozalaymiz.
		_ = uc.cache.Del(ctx, failKey(l.JoinSlug))
	}

	resp := &entity.JoinLessonResp{
		Lesson:   uc.toPublic(ctx, l),
		NextStep: "join",
	}

	// Kutish xonasi yoqilgan bo'lsa — so'rov yaratiladi, mentor tasdiqlashini kutadi.
	if l.IsWaitingRoomEnabled {
		guestName := ""
		if req.GuestName != nil {
			guestName = *req.GuestName
		}
		wr, err := uc.waitingRoom.CreateRequest(ctx, l, guestName)
		if err != nil {
			return nil, err
		}
		resp.NextStep = "waiting_room"
		resp.RequestID = wr.ID
		uc.log.Info(ctx, "joinlink.Join: waiting room", logger.String("lesson_id", l.ID), logger.String("request_id", wr.ID))
		return resp, nil
	}

	// To'g'ridan-to'g'ri kirish — participant tokeni beriladi.
	identity := "guest_" + uuid.NewString()
	displayName := "Mehmon"
	if req.GuestName != nil && *req.GuestName != "" {
		displayName = *req.GuestName
	}
	rt, err := uc.room.ParticipantToken(ctx, l, identity, displayName)
	if err != nil {
		return nil, err
	}
	resp.Room = rt

	uc.log.Info(ctx, "joinlink.Join: token issued", logger.String("lesson_id", l.ID), logger.String("identity", identity))
	return resp, nil
}

// resolveActive slug bo'yicha darsni topadi va kirish mumkinligini tekshiradi.
func (uc *useCase) resolveActive(ctx context.Context, slug string) (*entity.Lesson, error) {
	l, err := uc.lessonRepo.GetBySlug(ctx, slug)
	if err != nil {
		if apperr.IsNotFound(err) {
			return nil, apperr.NotFound("lesson")
		}
		return nil, err
	}
	switch l.Status {
	case entity.LessonStatusCancelled:
		return nil, apperr.BadRequest("lesson is cancelled")
	case entity.LessonStatusEnded:
		return nil, apperr.BadRequest("lesson has already ended")
	}
	return l, nil
}

func failKey(slug string) string { return "joinfail:" + slug }
func lockKey(slug string) string { return "joinlock:" + slug }

func (uc *useCase) toPublic(ctx context.Context, l *entity.Lesson) *entity.LessonPublic {
	return &entity.LessonPublic{
		ID:                   l.ID,
		Title:                l.Title,
		MentorName:           uc.mentorName(ctx, l.MentorID),
		ScheduledAt:          l.ScheduledAt,
		Status:               l.Status,
		HasPasscode:          l.PasscodeHash != nil,
		IsWaitingRoomEnabled: l.IsWaitingRoomEnabled,
	}
}

// mentorName mentor ismini qaytaradi — Redis cache bilan (har join'da DB o'qishni kamaytiradi).
func (uc *useCase) mentorName(ctx context.Context, mentorID string) string {
	key := "mentorname:" + mentorID
	if name, err := uc.cache.Get(ctx, key); err == nil && name != "" {
		return name
	}
	name := ""
	if m, err := uc.userRepo.GetByID(ctx, mentorID); err == nil {
		name = m.FullName
	}
	if name != "" {
		_ = uc.cache.Set(ctx, key, name, time.Hour)
	}
	return name
}

package room

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// Hamma chiqib ketgach xona shuncha soniya ochiq turadi (qayta ulanish uchun).
const roomEmptyTimeoutSec = 300

// Xona yaratilgani Redis'da shuncha vaqt eslab qolinadi (join hot-path'da qayta
// CreateRoom chaqirmaslik uchun — kechikishni kamaytiradi).
const roomReadyTTL = 6 * time.Hour

// LiveKit API chaqiruvlari uchun qat'iy deadline — backend↔LiveKit tarmog'i
// yomonlashsa request WriteTimeout'gacha osilib qolmasin (SDK klienti timeout'siz).
const lkOpTimeout = 10 * time.Second

// lkCtx LiveKit chaqiruvini lkOpTimeout bilan cheklaydi (chaqiruvchi ctx bekor bo'lsa
// ham darhol uzadi). Har chaqiruvdan keyin cancel() chaqirilishi shart.
func lkCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, lkOpTimeout)
}

type useCase struct {
	lessonRepo repository.LessonRepository
	userRepo   repository.UserRepository
	livekit    LiveKit
	cache      redis.Cache
	log        logger.Logger
	// ensureG — bir xona uchun bir vaqtda faqat bitta EnsureRoom (thundering herd:
	// 500 talaba bir vaqtda kirsa 500 ta SFU CreateRoom o'rniga bitta chaqiruv).
	ensureG singleflight.Group
}

func New(lessonRepo repository.LessonRepository, userRepo repository.UserRepository, lk LiveKit, cache redis.Cache, log logger.Logger) UseCase {
	return &useCase{lessonRepo: lessonRepo, userRepo: userRepo, livekit: lk, cache: cache, log: log}
}

func roomReadyKey(lessonID string) string { return "room:ready:" + lessonID }

// markRoomReady xona yaratilganini Redis'ga belgilaydi.
func (uc *useCase) markRoomReady(ctx context.Context, lessonID string) {
	if uc.cache != nil {
		_ = uc.cache.Set(ctx, roomReadyKey(lessonID), "1", roomReadyTTL)
	}
}

// ensureRoomOnce faqat xona hali yaratilmagan bo'lsa CreateRoom chaqiradi
// (join hot-path'da ortiqcha SFU chaqiruvini oldini oladi).
func (uc *useCase) ensureRoomOnce(ctx context.Context, lessonID, rn string) {
	if uc.cache != nil {
		if v, _ := uc.cache.Get(ctx, roomReadyKey(lessonID)); v != "" {
			return // allaqachon yaratilgan
		}
	}
	// singleflight: bir xona uchun bir vaqtda faqat bitta EnsureRoom bajariladi
	// (qolgan parallel join'lar shu natijani kutadi — SFU thundering herd yo'q).
	_, _, _ = uc.ensureG.Do(roomReadyKey(lessonID), func() (any, error) {
		lctx, cancel := lkCtx(ctx)
		defer cancel()
		if err := uc.livekit.EnsureRoom(lctx, rn, roomEmptyTimeoutSec); err != nil {
			uc.log.Warn(ctx, "room.ensureRoomOnce: ensure failed (auto-create fallback)", logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
			return nil, nil
		}
		uc.markRoomReady(ctx, lessonID)
		return nil, nil
	})
}

func (uc *useCase) HostToken(ctx context.Context, mentorID, lessonID string) (*entity.RoomToken, error) {
	// Avval mavjudlik/egalik/holat tekshiriladi — video servis o'chirilgan bo'lsa ham
	// yo'q dars 404, begona dars 403 bo'lib qolsin (500 bilan niqoblanmasin).
	// ValidateID: yaroqsiz UUID Postgres'ga yetmasin (22P02 → 500). OwnedLesson'ga
	// o'tmaymiz — bu yerdagi xato matni ("not the host") API kontraktida saqlanadi.
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return nil, err
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if l.MentorID != mentorID {
		return nil, apperr.Forbidden("you are not the host of this lesson")
	}
	if l.Status == entity.LessonStatusEnded || l.Status == entity.LessonStatusCancelled {
		return nil, apperr.BadRequest("lesson is not active")
	}
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}

	start := time.Now()
	defer func() { metrics.LiveKitOpDuration.WithLabelValues("host_token").Observe(time.Since(start).Seconds()) }()

	roomName := roomName(l.ID)
	ensureCtx, ensureCancel := lkCtx(ctx)
	err = uc.livekit.EnsureRoom(ensureCtx, roomName, roomEmptyTimeoutSec)
	ensureCancel()
	if err != nil {
		metrics.LiveKitErrors.WithLabelValues("create_room").Inc()
		uc.log.Error(ctx, "room.HostToken: ensure room failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		return nil, apperr.Internal(fmt.Errorf("could not create video room: %w", err))
	}
	uc.markRoomReady(ctx, l.ID) // keyingi join'larda CreateRoom takrorlanmasin

	// Darsni jonli holatga o'tkazish (birinchi marta host qo'shilganda).
	if l.Status != entity.LessonStatusLive {
		now := time.Now().UTC()
		l.Status = entity.LessonStatusLive
		if l.StartedAt == nil {
			l.StartedAt = &now
		}
		if err := uc.lessonRepo.Update(ctx, l); err != nil {
			uc.log.Warn(ctx, "room.HostToken: could not mark lesson live", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		}
	}

	hostName := "Host"
	if m, err := uc.userRepo.GetByID(ctx, mentorID); err == nil {
		hostName = m.FullName
	}

	token, err := uc.livekit.AccessToken(roomName, mentorID, hostName, true)
	if err != nil {
		metrics.LiveKitErrors.WithLabelValues("host_token").Inc()
		return nil, apperr.Internal(fmt.Errorf("room.HostToken generate token: %w", err))
	}

	metrics.RoomTokensIssued.WithLabelValues("host").Inc()
	uc.log.Info(ctx, "room.HostToken: issued", logger.String("lesson_id", l.ID), logger.String("mentor_id", mentorID))
	return &entity.RoomToken{
		Token:    token,
		WSURL:    uc.livekit.WSURL(),
		RoomName: roomName,
		Identity: mentorID,
		Role:     entity.RoomRoleHost,
	}, nil
}

func (uc *useCase) ParticipantToken(ctx context.Context, l *entity.Lesson, identity, displayName string) (*entity.RoomToken, error) {
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}
	roomName := roomName(l.ID)
	// Xonani faqat kerak bo'lsa yaratamiz (hot-path'da ortiqcha SFU chaqiruvi yo'q).
	uc.ensureRoomOnce(ctx, l.ID, roomName)
	if displayName == "" {
		displayName = "Mehmon"
	}

	token, err := uc.livekit.AccessToken(roomName, identity, displayName, false)
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("room.ParticipantToken generate token: %w", err))
	}

	metrics.RoomTokensIssued.WithLabelValues("participant").Inc()
	uc.log.Info(ctx, "room.ParticipantToken: issued", logger.String("lesson_id", l.ID), logger.String("identity", identity))
	return &entity.RoomToken{
		Token:    token,
		WSURL:    uc.livekit.WSURL(),
		RoomName: roomName,
		Identity: identity,
		Role:     entity.RoomRoleParticipant,
	}, nil
}

func (uc *useCase) EndLesson(ctx context.Context, mentorID, lessonID string) error {
	if err := shared.ValidateID(lessonID, "lesson"); err != nil {
		return err
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return err
	}
	if l.MentorID != mentorID {
		return apperr.Forbidden("you are not the host of this lesson")
	}

	now := time.Now().UTC()
	l.Status = entity.LessonStatusEnded
	l.EndedAt = &now
	if err := uc.lessonRepo.Update(ctx, l); err != nil {
		uc.log.Error(ctx, "room.EndLesson: update failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		return err
	}

	if uc.livekit.Enabled() {
		delCtx, delCancel := lkCtx(ctx)
		err := uc.livekit.DeleteRoom(delCtx, roomName(l.ID))
		delCancel()
		if err != nil {
			uc.log.Warn(ctx, "room.EndLesson: delete room failed", logger.String("lesson_id", l.ID), logger.SafeString("err", err.Error()))
		}
	}
	// Stale "room-ready" flagni tozalaymiz (aks holda qayta ulanishda EnsureRoom o'tkazib yuboriladi).
	if uc.cache != nil {
		_ = uc.cache.Del(ctx, roomReadyKey(l.ID))
	}

	uc.log.Info(ctx, "room.EndLesson: ended", logger.String("lesson_id", l.ID))
	return nil
}

// roomName dars ID sidan barqaror LiveKit xona nomini hosil qiladi (shared.RoomName).
func roomName(lessonID string) string {
	return shared.RoomName(lessonID)
}

func (uc *useCase) ListParticipants(ctx context.Context, mentorID, lessonID string) ([]entity.RoomParticipant, error) {
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	items, err := uc.livekit.ListParticipantViews(lctx, roomName(lessonID))
	if err != nil {
		return nil, apperr.Internal(fmt.Errorf("room.ListParticipants: %w", err))
	}
	return items, nil
}

// isHostIdentity — LiveKit'dagi host identity'si mentorID bilan AYNAN bir xil
// (HostToken: `AccessToken(roomName, mentorID, hostName, true)`), shuning uchun
// moderatsiya amallarining nishoni mentorID bo'lsa — ustoz o'ziga qarshi amal
// bajarayotgan bo'ladi. MuteAll bu qoidani allaqachon qo'llaydi (host'ni o'tkazib
// yuboradi); qolgan uch amalda guard yo'q edi. Eng xatarlisi allow-speak: u
// CanPublishSources'ni [CAMERA, MICROPHONE] ga tushiradi va ustozning EKRAN
// ULASHISHINI o'chirib qo'yadi (qayta ulanmaguncha tiklanmaydi).
func isHostIdentity(mentorID, identity string) bool {
	return identity != "" && identity == mentorID
}

func (uc *useCase) MuteParticipant(ctx context.Context, mentorID, lessonID, identity string, audioOnly bool) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	// Host'ni moderatsiya API'si orqali mute qilmaymiz — MuteAll bilan izchil.
	// O'z mikrofonini o'chirish klient SDK'sining ishi (lokal mute), server-mute emas.
	if isHostIdentity(mentorID, identity) {
		return apperr.BadRequest("cannot mute the host: use the client SDK to mute your own microphone")
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.MuteParticipant(lctx, roomName(lessonID), identity, audioOnly); err != nil {
		return apperr.Internal(fmt.Errorf("room.MuteParticipant: %w", err))
	}
	uc.log.Info(ctx, "room: participant muted", logger.String("lesson_id", lessonID), logger.String("identity", identity))
	return nil
}

func (uc *useCase) MuteAll(ctx context.Context, mentorID, lessonID string) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	rn := roomName(lessonID)
	listCtx, listCancel := lkCtx(ctx)
	parts, err := uc.livekit.ListParticipantViews(listCtx, rn)
	listCancel()
	if err != nil {
		return apperr.Internal(fmt.Errorf("room.MuteAll list: %w", err))
	}
	for _, p := range parts {
		if p.Identity == mentorID {
			continue // host'ni mute qilmaymiz
		}
		muteCtx, muteCancel := lkCtx(ctx)
		_ = uc.livekit.MuteParticipant(muteCtx, rn, p.Identity, true)
		muteCancel()
	}
	uc.log.Info(ctx, "room: mute all", logger.String("lesson_id", lessonID))
	return nil
}

func (uc *useCase) RemoveParticipant(ctx context.Context, mentorID, lessonID, identity string) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	// Host o'zini xonadan chiqarib yuborsa dars "live" holatida host'siz qoladi
	// (ishtirokchilar boshsiz xonada) — darsni tugatish uchun alohida endpoint bor.
	if isHostIdentity(mentorID, identity) {
		return apperr.BadRequest("cannot remove the host: use POST /lessons/:id/end to finish the lesson")
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.RemoveParticipant(lctx, roomName(lessonID), identity); err != nil {
		return apperr.Internal(fmt.Errorf("room.RemoveParticipant: %w", err))
	}
	uc.log.Info(ctx, "room: participant removed", logger.String("lesson_id", lessonID), logger.String("identity", identity))
	return nil
}

func (uc *useCase) SetSpeakPermission(ctx context.Context, mentorID, lessonID, identity string, canPublish bool) error {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return err
	}
	// Host'ning publish huquqlari FAQAT host token'idan keladi (screen-share bilan).
	// Bu yerdan o'tsa student to'plamiga (kamera+mikrofon) tushib, ekran ulashish
	// o'chib qolardi — mobil ilovaning asosiy funksiyasi.
	if isHostIdentity(mentorID, identity) {
		return apperr.BadRequest("cannot change the host's publish permissions: host rights come from the host token")
	}
	lctx, cancel := lkCtx(ctx)
	defer cancel()
	if err := uc.livekit.SetParticipantPublish(lctx, roomName(lessonID), identity, canPublish); err != nil {
		return apperr.Internal(fmt.Errorf("room.SetSpeakPermission: %w", err))
	}
	uc.log.Info(ctx, "room: speak permission set", logger.String("lesson_id", lessonID), logger.String("identity", identity), logger.Int("can_publish", boolToInt(canPublish)))
	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

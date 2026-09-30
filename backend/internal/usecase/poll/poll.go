package poll

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

type useCase struct {
	repo       repository.PollRepository
	lessonRepo repository.LessonRepository
	livekit    LiveKit
	// cache — darsdan chiqarilganlar reyestri uchun (`shared.GuardRoomAction`).
	cache redis.Cache
	log   logger.Logger
}

func New(
	repo repository.PollRepository,
	lessonRepo repository.LessonRepository,
	lk LiveKit,
	cache redis.Cache,
	log logger.Logger,
) UseCase {
	return &useCase{repo: repo, lessonRepo: lessonRepo, livekit: lk, cache: cache, log: log}
}

func (uc *useCase) Create(ctx context.Context, mentorID, lessonID, question string, options []string, resultsVisibility string) (*entity.Poll, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	// Default YOPIQ tomonda: maydonni yubormagan (eski) klient natijani
	// tasodifan ochib qo'ymasin.
	switch resultsVisibility {
	case "":
		resultsVisibility = entity.PollResultsMentorOnly
	case entity.PollResultsMentorOnly, entity.PollResultsPublic:
	default:
		return nil, apperr.BadRequest("results_visibility must be mentor_only or public")
	}

	p := &entity.Poll{
		ID:                uuid.NewString(),
		LessonID:          lessonID,
		Question:          question,
		Options:           options,
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
		ResultsVisibility: resultsVisibility,
	}
	if err := uc.repo.Create(ctx, p); err != nil {
		uc.log.Error(ctx, "poll.Create: db error", logger.SafeString("err", err.Error()))
		return nil, err
	}
	uc.log.Info(ctx, "poll created",
		logger.String("poll_id", p.ID), logger.String("lesson_id", lessonID),
		logger.String("results_visibility", resultsVisibility))
	return p, nil
}

func (uc *useCase) Close(ctx context.Context, mentorID, pollID string) (*entity.PollResults, error) {
	p, err := uc.ownedPoll(ctx, mentorID, "", pollID)
	if err != nil {
		return nil, err
	}
	if err := uc.repo.Close(ctx, pollID); err != nil {
		return nil, err
	}
	// YOPISH ≠ E'LON QILISH. Yopilgan so'rovnoma natijasi ham o'quvchiga
	// avtomatik ochilmaydi — buni faqat `Publish` qiladi. Aks holda asoschining
	// «natija faqat E'lon qilish bosilganda chiqadi» qoidasi buzilardi.
	p.IsActive = false
	return uc.results(ctx, p)
}

func (uc *useCase) ListByLesson(ctx context.Context, mentorID, lessonID string) ([]*entity.Poll, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	return uc.repo.ListByLesson(ctx, lessonID)
}

func (uc *useCase) Vote(ctx context.Context, pollID, voterIdentity, tokenRoom string, optionIndex int) error {
	// Yaroqsiz UUID Postgres'ga yetmasin (22P02 → 500): bu endpoint OCHIQ, ya'ni
	// autentifikatsiyasiz 500 generatori bo'lardi va Sentry'ni ko'mib tashlardi.
	if err := shared.ValidateID(pollID, "poll"); err != nil {
		return err
	}
	p, err := uc.repo.GetByID(ctx, pollID)
	if err != nil {
		return err
	}
	// Room-token poll'ning darsiga tegishli bo'lishi shart — aks holda boshqa darsning
	// (yoki o'z darsi host) tokeni bilan begona so'rovnoma natijasini soxtalashtirish mumkin.
	if tokenRoom != shared.RoomName(p.LessonID) {
		return apperr.Forbidden("room token is not valid for this poll's lesson")
	}
	// Chiqarilgan ishtirokchi ovoz bera olmasin: uning tokeni hali yaroqli va
	// tekshiruvsiz u natijani buzishda davom etardi.
	if err := shared.GuardRoomAction(ctx, uc.lessonRepo, uc.cache, p.LessonID, voterIdentity); err != nil {
		return err
	}
	if !p.IsActive {
		return apperr.BadRequest("poll is closed")
	}
	if optionIndex < 0 || optionIndex >= len(p.Options) {
		return apperr.BadRequest("invalid option index")
	}
	if err := uc.repo.Vote(ctx, pollID, voterIdentity, optionIndex); err != nil {
		if apperr.IsBadRequest(err) { // repo shartli upsert: poll shu orada yopilgan
			return err
		}
		uc.log.Error(ctx, "poll.Vote: db error", logger.String("poll_id", pollID), logger.SafeString("err", err.Error()))
		return err
	}
	return nil
}

// Results — natijalar. `tokenRoom` — chaqiruvchining LiveKit room-token'idagi xona
// (bo'sh bo'lsa tekshirilmaydi: bu ICHKI/HOST yo'li).
//
// Nega token kerak: avval bu endpoint UMUMAN ochiq edi — poll ID'ni bilgan har kim
// (masalan sinfdosh, yoki ID'ni chatdan ko'rgan begona) natijani o'qiy olardi.
// Ovoz berish esa allaqachon token talab qilardi, ya'ni himoya nomutanosib edi.
//
// Endi bunga E'LON tekshiruvi ham qo'shildi (№7): xonada bo'lish natijani
// ko'rishga yetmaydi.
func (uc *useCase) Results(ctx context.Context, pollID, viewerIdentity, tokenRoom string) (*entity.PollResults, error) {
	if err := shared.ValidateID(pollID, "poll"); err != nil {
		return nil, err
	}
	p, err := uc.repo.GetByID(ctx, pollID)
	if err != nil {
		return nil, err
	}
	if tokenRoom != "" && tokenRoom != shared.RoomName(p.LessonID) {
		return nil, apperr.Forbidden("room token is not valid for this poll's lesson")
	}
	// Chiqarilgan (kick) ishtirokchi token'i hali yaroqli bo'lsa ham natijani o'qiy olmasin.
	if viewerIdentity != "" && shared.IsBanned(ctx, uc.cache, p.LessonID, viewerIdentity) {
		return nil, shared.ErrBanned()
	}
	if !p.ResultsVisibleTo(uc.isHost(ctx, p.LessonID, viewerIdentity)) {
		// 403 (bo'sh natija emas): "0 ovoz" bilan "ko'rsatilmaydi" ni farqlab
		// bo'lmasa klient noto'g'ri diagramma chizardi va o'quvchi so'rovnoma
		// buzuq deb o'ylardi.
		return nil, apperr.Forbidden("poll results are not published yet")
	}
	return uc.results(ctx, p)
}

func (uc *useCase) Publish(ctx context.Context, mentorID, lessonID, pollID string) (*entity.PollResults, error) {
	p, err := uc.ownedPoll(ctx, mentorID, lessonID, pollID)
	if err != nil {
		return nil, err
	}
	// `mentor_only` — REJIM, e'lon qilish uni bekor qilmaydi.
	//
	// Nega jimgina "public"ga o'tkazib yubormaymiz: rejim yaratishda ongli
	// tanlangan va u so'rovnomaga ovoz bergan o'quvchiga berilgan va'da
	// ("bu natija ko'rsatilmaydi"). Uni keyin bir tugma bilan buzish mumkin
	// bo'lsa, rejimning ma'nosi qolmaydi. Klient bu holatda tugmani umuman
	// ko'rsatmasligi kerak; 400 — himoya qatlami.
	if p.ResultsVisibility != entity.PollResultsPublic {
		return nil, apperr.BadRequest("this poll was created as mentor_only — its results cannot be published")
	}

	published, err := uc.repo.Publish(ctx, pollID)
	if err != nil {
		return nil, err
	}
	res, err := uc.results(ctx, published)
	if err != nil {
		return nil, err
	}
	uc.broadcast(ctx, published.LessonID, entity.NewPollPublishedEvent(res))
	uc.log.Info(ctx, "poll results published",
		logger.String("poll_id", pollID), logger.String("lesson_id", published.LessonID))
	return res, nil
}

// ─── ichki yordamchilar ───────────────────────────────────────────────────────

// ownedPoll — poll'ni yuklaydi va uni chaqiruvchi mentor darsiga tegishliligini
// tekshiradi. lessonID bo'sh bo'lmasa, poll AYNAN o'sha darsniki bo'lishi shart
// (yo'ldagi dars ID si bilan mos kelishi) — begona darsning yo'li orqali
// murojaat qilib bo'lmasin.
func (uc *useCase) ownedPoll(ctx context.Context, mentorID, lessonID, pollID string) (*entity.Poll, error) {
	if err := shared.ValidateID(pollID, "poll"); err != nil {
		return nil, err
	}
	p, err := uc.repo.GetByID(ctx, pollID)
	if err != nil {
		return nil, err
	}
	if lessonID != "" && lessonID != p.LessonID {
		return nil, apperr.NotFound("poll")
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, p.LessonID); err != nil {
		return nil, err
	}
	return p, nil
}

// results — ovozlarni sanaydi va natija obyektini yasaydi (ko'rinuvchanlik
// TEKSHIRILMAYDI — chaqiruvchining ishi).
func (uc *useCase) results(ctx context.Context, p *entity.Poll) (*entity.PollResults, error) {
	counts, err := uc.repo.Counts(ctx, p.ID, len(p.Options))
	if err != nil {
		return nil, err
	}
	total := 0
	for _, c := range counts {
		total += c
	}
	return &entity.PollResults{Poll: p, Counts: counts, Total: total}, nil
}

// isHost — chaqiruvchi shu darsning ustozimi.
//
// Host'ning LiveKit identity'si = mentorID (`room.HostToken` shunday beradi),
// shuning uchun room-token'ning o'zi mentorlikni isbotlaydi va qo'shimcha JWT
// kerak emas. Guest identity'si esa "guest_<uuid>" — u hech qachon mentor
// UUID'siga teng bo'lolmaydi.
func (uc *useCase) isHost(ctx context.Context, lessonID, identity string) bool {
	if identity == "" {
		// Ichki (host) chaqiruv: `Close`/`Publish` allaqachon egalikni tekshirgan.
		return true
	}
	l, err := uc.lessonRepo.GetByID(ctx, lessonID)
	if err != nil {
		return false
	}
	return l.MentorID == identity
}

// broadcast — xonaga data-channel xabari (yiqilsa amal bekor qilinmaydi:
// e'lon DB'da yozilgan va o'quvchi `GET /polls/:id/results` bilan baribir oladi).
func (uc *useCase) broadcast(ctx context.Context, lessonID string, msg any) {
	if uc.livekit == nil || !uc.livekit.Enabled() {
		return
	}
	data, err := json.Marshal(msg)
	if err != nil {
		uc.log.Error(ctx, "poll: marshal failed", logger.SafeString("err", err.Error()))
		return
	}
	if err := uc.livekit.SendData(ctx, shared.RoomName(lessonID), data); err != nil {
		uc.log.Warn(ctx, "poll: broadcast failed",
			logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
	}
}

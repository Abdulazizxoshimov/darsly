package poll

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

type useCase struct {
	repo       repository.PollRepository
	lessonRepo repository.LessonRepository
	log        logger.Logger
}

func New(repo repository.PollRepository, lessonRepo repository.LessonRepository, log logger.Logger) UseCase {
	return &useCase{repo: repo, lessonRepo: lessonRepo, log: log}
}

func (uc *useCase) Create(ctx context.Context, mentorID, lessonID, question string, options []string) (*entity.Poll, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	p := &entity.Poll{
		ID:        uuid.NewString(),
		LessonID:  lessonID,
		Question:  question,
		Options:   options,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
	}
	if err := uc.repo.Create(ctx, p); err != nil {
		uc.log.Error(ctx, "poll.Create: db error", logger.SafeString("err", err.Error()))
		return nil, err
	}
	uc.log.Info(ctx, "poll created", logger.String("poll_id", p.ID), logger.String("lesson_id", lessonID))
	return p, nil
}

func (uc *useCase) Close(ctx context.Context, mentorID, pollID string) (*entity.PollResults, error) {
	if err := shared.ValidateID(pollID, "poll"); err != nil {
		return nil, err
	}
	p, err := uc.repo.GetByID(ctx, pollID)
	if err != nil {
		return nil, err
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, p.LessonID); err != nil {
		return nil, err
	}
	if err := uc.repo.Close(ctx, pollID); err != nil {
		return nil, err
	}
	return uc.Results(ctx, pollID, "")
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
	if !p.IsActive {
		return apperr.BadRequest("poll is closed")
	}
	if optionIndex < 0 || optionIndex >= len(p.Options) {
		return apperr.BadRequest("invalid option index")
	}
	if err := uc.repo.Vote(ctx, pollID, voterIdentity, optionIndex); err != nil {
		uc.log.Error(ctx, "poll.Vote: db error", logger.String("poll_id", pollID), logger.SafeString("err", err.Error()))
		return err
	}
	return nil
}

// Results — natijalar. `tokenRoom` — chaqiruvchining LiveKit room-token'idagi xona
// (bo'sh bo'lsa tekshirilmaydi: bu HOST yo'li, u allaqachon egalik bo'yicha tekshirilgan).
//
// Nega token kerak: avval bu endpoint UMUMAN ochiq edi — poll ID'ni bilgan har kim
// (masalan sinfdosh, yoki ID'ni chatdan ko'rgan begona) natijani o'qiy olardi.
// Ovoz berish esa allaqachon token talab qilardi, ya'ni himoya nomutanosib edi.
func (uc *useCase) Results(ctx context.Context, pollID, tokenRoom string) (*entity.PollResults, error) {
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
	counts, err := uc.repo.Counts(ctx, pollID, len(p.Options))
	if err != nil {
		return nil, err
	}
	total := 0
	for _, c := range counts {
		total += c
	}
	return &entity.PollResults{Poll: p, Counts: counts, Total: total}, nil
}

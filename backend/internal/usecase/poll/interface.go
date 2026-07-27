package poll

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UseCase interface {
	Create(ctx context.Context, mentorID, lessonID, question string, options []string) (*entity.Poll, error)
	Close(ctx context.Context, mentorID, pollID string) (*entity.PollResults, error)
	ListByLesson(ctx context.Context, mentorID, lessonID string) ([]*entity.Poll, error)
	// Vote — ishtirokchi ovoz beradi (voterIdentity va tokenRoom room-token'dan olinadi).
	// tokenRoom poll'ning darsiga bog'lanadi — begona darsning tokeni bilan ovoz berib bo'lmaydi.
	Vote(ctx context.Context, pollID, voterIdentity, tokenRoom string, optionIndex int) error
	// Results — so'rovnoma natijalari (ochiq).
	Results(ctx context.Context, pollID string) (*entity.PollResults, error)
}

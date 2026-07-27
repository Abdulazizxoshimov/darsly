package repository

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type PollRepository interface {
	Create(ctx context.Context, p *entity.Poll) error
	GetByID(ctx context.Context, id string) (*entity.Poll, error)
	ListByLesson(ctx context.Context, lessonID string) ([]*entity.Poll, error)
	Close(ctx context.Context, id string) error
	// Vote — bitta ishtirokchi bitta ovoz (upsert: qayta ovoz variantni yangilaydi).
	Vote(ctx context.Context, pollID, voterIdentity string, optionIndex int) error
	// Counts — har variant uchun ovozlar soni.
	Counts(ctx context.Context, pollID string, numOptions int) ([]int, error)
}

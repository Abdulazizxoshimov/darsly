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
	// Publish — natijani e'lon qiladi (№7) va e'lon vaqtini qaytaradi.
	//
	// IDEMPOTENT: `COALESCE(results_published_at, NOW())` — mentor tugmani ikki
	// marta bossa e'lon vaqti surilib ketmasin. E'lon bir tomonlama amal:
	// qaytarib olish yo'q (o'quvchi allaqachon ko'rgan, "yashirdim" degan
	// da'vo yolg'on bo'lardi).
	Publish(ctx context.Context, pollID string) (*entity.Poll, error)
}

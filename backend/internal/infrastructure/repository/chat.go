package repository

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

type ChatRepository interface {
	Create(ctx context.Context, m *entity.ChatMessage) error
	// ListByLesson — eng yangi xabarlarni qaytaradi (created_at DESC). before!=nil bo'lsa
	// faqat undan eski xabarlar (kursor-paginatsiya). limit sahifa hajmi.
	ListByLesson(ctx context.Context, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error)
}

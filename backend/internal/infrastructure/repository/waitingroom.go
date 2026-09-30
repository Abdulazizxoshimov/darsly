package repository

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

type WaitingRoomRepository interface {
	Create(ctx context.Context, req *entity.WaitingRoomRequest) error
	GetByID(ctx context.Context, id string) (*entity.WaitingRoomRequest, error)
	ListPending(ctx context.Context, lessonID string) ([]*entity.WaitingRoomRequest, error)
	// ListPendingByMentor — mentorning BARCHA darslaridagi kutayotgan so'rovlar.
	// WS reconnect'da snapshot yetkazish uchun (uzilishда yo'qolgan push'lar).
	ListPendingByMentor(ctx context.Context, mentorID string) ([]*entity.WaitingRoomRequest, error)
	// CountPendingByLesson — darsdagi kutayotgan so'rovlar soni (ochiq endpoint spam cap'i uchun).
	CountPendingByLesson(ctx context.Context, lessonID string) (int, error)
	// TransitionFromPending atomik ravishda status'ni pending'dan boshqasiga o'tkazadi.
	// Faqat hozirgi status pending bo'lsa true qaytadi (TOCTOU race'dan himoya).
	TransitionFromPending(ctx context.Context, id, newStatus string, decidedAt time.Time) (bool, error)
}

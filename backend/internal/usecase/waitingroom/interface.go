package waitingroom

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UseCase interface {
	// CreateRequest — kutish so'rovi yaratadi va mentorga WS orqali bildiradi.
	CreateRequest(ctx context.Context, lesson *entity.Lesson, requesterName string) (*entity.WaitingRoomRequest, error)
	// ListPending — mentor uchun kutayotgan so'rovlar (ownership tekshiruvi bilan).
	ListPending(ctx context.Context, mentorID, lessonID string) ([]*entity.WaitingRoomRequest, error)
	// Admit — so'rovni qabul qiladi, participant token generatsiya qiladi va guestga WS orqali yuboradi.
	Admit(ctx context.Context, mentorID, requestID string) (*entity.RoomToken, error)
	// AdmitAll — darsning BARCHA kutayotgan so'rovlarini qabul qiladi
	// (ommaviy «Hammasini kiritish»). Qisman muvaffaqiyat mumkin — natija
	// sonlar bilan qaytadi.
	AdmitAll(ctx context.Context, mentorID, lessonID string) (*entity.AdmitAllResp, error)
	// Reject — so'rovni rad etadi va guestga WS orqali bildiradi.
	Reject(ctx context.Context, mentorID, requestID string) error
	// Status — guest so'rovining hozirgi holati (public; admitted bo'lsa token bilan).
	Status(ctx context.Context, requestID string) (*entity.WaitingRoomStatusResp, error)
	// DeliverCurrentStatus — guest WS ulanganda, agar qaror allaqachon chiqqan bo'lsa, darhol yuboradi.
	DeliverCurrentStatus(ctx context.Context, requestID string)
	// DeliverPendingSnapshot — mentor WS (re)connect'da barcha kutayotgan so'rovlarni
	// qayta yuboradi (uzilish paytida yo'qolgan real-time push'larni qoplaydi).
	DeliverPendingSnapshot(ctx context.Context, mentorID string)
}

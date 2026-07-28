package roomstate

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

// LiveKit — roomstate talab qiladigan LiveKit operatsiyalari (DIP).
type LiveKit interface {
	Enabled() bool
	SendData(ctx context.Context, room string, data []byte) error
	// VerifyToken guest'ni room-token orqali autentifikatsiya qiladi
	// (JWT'siz ishtirokchilar uchun yagona yo'l).
	VerifyToken(token string) (identity, name, room string, err error)
}

// UseCase — dars xonasining o'tkinchi holati (qo'l ko'tarish, reaksiya).
//
// Bu holat Postgres'da EMAS, Redis'da: u darsning umri bilan cheklangan va
// dars tugagach hech kimga kerak emas. Chat esa aksincha — DB'da (tarix, yozuv,
// moderatsiya), shuning uchun u alohida domen bo'lib qoladi.
type UseCase interface {
	// SetHand — ishtirokchi qo'lini ko'taradi/tushiradi va xonaga tarqatadi.
	SetHand(ctx context.Context, lessonID, identity, name string, raised bool) error
	// LowerHand — HOST boshqa ishtirokchining qo'lini tushiradi (egalik tekshiriladi).
	LowerHand(ctx context.Context, mentorID, lessonID, identity string) error
	// LowerAll — HOST barcha qo'llarni tushiradi.
	LowerAll(ctx context.Context, mentorID, lessonID string) error
	// State — joriy holat (kech kirgan klient shu bilan tiklanadi).
	State(ctx context.Context, lessonID string) (*entity.RoomState, error)
	// Reaction — emoji tarqatadi (saqlanmaydi). Server tomonda tezlik cheklovi bor.
	Reaction(ctx context.Context, lessonID, identity, name, emoji string) error
	// Clear — dars yakunlanganda holatni tozalaydi.
	Clear(ctx context.Context, lessonID string) error
}

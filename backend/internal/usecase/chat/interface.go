package chat

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

// LiveKit — chat usecase talab qiladigan LiveKit operatsiyalari (DIP).
type LiveKit interface {
	Enabled() bool
	SendData(ctx context.Context, room string, data []byte) error
	// SendDataTo — faqat ko'rsatilgan ishtirokchilarga (shaxsiy xabar).
	SendDataTo(ctx context.Context, room string, data []byte, identities []string) error
}

// UseCase — dars ichidagi chat.
//
// Ikki kirish yo'li bor va bu ATAYLAB:
//   - `Send`/`History` — HOST yo'li (JWT + egalik). Ustoz dars tugagach ham
//     tarixni o'qiy oladi, ya'ni xonaga ulangan bo'lishi shart emas.
//   - `SendFromRoom`/`HistoryForRoom` — XONA yo'li (LiveKit room-token). Guest'da
//     JWT yo'q; room-token esa faqat dars davomida yaroqli — to'g'ri cheklov.
type UseCase interface {
	// Send — host xabar yuboradi: saqlaydi va tarqatadi. to bo'sh bo'lsa — hammaga.
	Send(ctx context.Context, mentorID, lessonID, body, to string) (*entity.ChatMessage, error)
	// History — dars chat tarixi (host), eng yangidan eskiga. before!=nil → kursor
	// (undan eski xabarlar). Sahifa hajmi limit (default/max 50).
	History(ctx context.Context, mentorID, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error)

	// SendFromRoom — xonadagi ISHTIROKCHI xabar yuboradi (room-token bilan
	// autentifikatsiya qilingan identity). Tezlik cheklovi shu yerda.
	SendFromRoom(ctx context.Context, lessonID, identity, name, body, to string) (*entity.ChatMessage, error)
	// HistoryForRoom — xonadagi ishtirokchi uchun tarix: ommaviy xabarlar +
	// faqat O'ZI ishtirok etgan shaxsiy yozishmalar.
	HistoryForRoom(ctx context.Context, lessonID, identity string, before *time.Time, limit int) ([]*entity.ChatMessage, error)
}

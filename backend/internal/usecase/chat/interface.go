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
}

type UseCase interface {
	// Send — host xabar yuboradi: saqlaydi va LiveKit data-channel orqali xonaga tarqatadi.
	Send(ctx context.Context, mentorID, lessonID, body string) (*entity.ChatMessage, error)
	// History — dars chat tarixi (host), eng yangidan eskiga. before!=nil → kursor
	// (undan eski xabarlar). Sahifa hajmi limit (default/max 50).
	History(ctx context.Context, mentorID, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error)
}

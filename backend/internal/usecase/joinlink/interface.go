package joinlink

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UseCase interface {
	// Preview — slug orqali darsning ochiq ma'lumotini qaytaradi (parolsiz, ekran ko'rsatish uchun).
	Preview(ctx context.Context, slug string) (*entity.LessonPublic, error)
	// Join — parol/qulf/status tekshiruvidan o'tkazib, keyingi qadamni qaytaradi.
	Join(ctx context.Context, slug, clientIP string, req *entity.JoinLessonReq) (*entity.JoinLessonResp, error)
}

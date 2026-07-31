package repository

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

// BlocklistRepository — mentor darajasidagi doimiy qora ro'yxat (№4).
type BlocklistRepository interface {
	// Add — yozuv qo'shadi. Ism bo'yicha dublikat (bir mentor, bir ism)
	// jimgina e'tiborsiz qoldiriladi (ON CONFLICT DO NOTHING).
	Add(ctx context.Context, e *entity.BlocklistEntry) error
	// IsBlocked — displayName shu mentorning qora ro'yxatidami
	// (katta-kichik harf farqsiz). Bo'sh ism hech qachon mos kelmaydi.
	IsBlocked(ctx context.Context, mentorID, displayName string) (bool, error)
	// ListByMentor — mentorning barcha yozuvlari (yangi birinchi).
	ListByMentor(ctx context.Context, mentorID string) ([]*entity.BlocklistEntry, error)
	// Delete — mentor o'z yozuvini o'chiradi (unban). Begona yozuv NotFound.
	Delete(ctx context.Context, mentorID, id string) error
}

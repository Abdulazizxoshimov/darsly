package repository

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

type LessonRepository interface {
	Create(ctx context.Context, lesson *entity.Lesson) error
	GetByID(ctx context.Context, id string) (*entity.Lesson, error)
	GetBySlug(ctx context.Context, slug string) (*entity.Lesson, error)
	SlugExists(ctx context.Context, slug string) (bool, error)
	ListByMentor(ctx context.Context, mentorID string, filter *entity.LessonFilter) ([]*entity.Lesson, int, error)
	Update(ctx context.Context, lesson *entity.Lesson) error
	SoftDelete(ctx context.Context, id string) error

	// Eslatma yuborilmagan, [from, to] oralig'ida boshlanadigan rejalashtirilgan darslar.
	ListUpcomingUnreminded(ctx context.Context, from, to time.Time) ([]*entity.Lesson, error)
	// ClaimReminder atomik ravishda darsni "eslatildi" deb belgilaydi. Faqat hali
	// belgilanmagan bo'lsa true qaytadi — ko'p instansda dublikat eslatma oldini oladi.
	ClaimReminder(ctx context.Context, id string) (bool, error)
}

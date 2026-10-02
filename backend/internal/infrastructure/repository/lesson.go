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
	// Update — to'liq qator yozadi. Ishlab chiqarish kodida ISHLATILMAYDI (lost-update
	// xavfi); qisman o'zgarish uchun UpdateFields, lifecycle uchun ClaimStart/ClaimEnd.
	Update(ctx context.Context, lesson *entity.Lesson) error
	// UpdateFields faqat patch'dagi to'ldirilgan maydonlarni yangilaydi.
	UpdateFields(ctx context.Context, id string, patch *entity.LessonPatch) error
	// ClaimStart atomik ravishda darsni `live` qiladi (scheduled yoki live bo'lsa).
	// started_at faqat birinchi marta yoziladi. Dars tugagan/bekor qilingan bo'lsa false.
	ClaimStart(ctx context.Context, id string, startedAt time.Time) (bool, error)
	SoftDelete(ctx context.Context, id string) error
	// CancelByMentor mentorning barcha o'chirilmagan darslarini bekor qiladi
	// (deleted_at + status='cancelled'). Mentor hisobi o'chirilganda join-linklar
	// ishlashda davom etmasligi uchun.
	CancelByMentor(ctx context.Context, mentorID string) error

	// Eslatma yuborilmagan, [from, to] oralig'ida boshlanadigan rejalashtirilgan darslar.
	ListUpcomingUnreminded(ctx context.Context, from, to time.Time) ([]*entity.Lesson, error)
	// ClaimReminder atomik ravishda darsni "eslatildi" deb belgilaydi. Faqat hali
	// belgilanmagan bo'lsa true qaytadi — ko'p instansda dublikat eslatma oldini oladi.
	ClaimReminder(ctx context.Context, id string) (bool, error)
	// UnclaimReminder — reminder belgisini qaytaradi (Notify uzil-kesil xato
	// bo'lsa keyingi tick qayta urinsin, audit realtime #11).
	UnclaimReminder(ctx context.Context, id string) error

	// ListLive — hozir `live` holatidagi darslar (avto-yakun ishchisi uchun).
	ListLive(ctx context.Context) ([]*entity.Lesson, error)
	// ClaimEnd atomik ravishda darsni yakunlaydi: faqat dars hamon `live` bo'lsa
	// true qaytadi. `ClaimReminder` bilan bir xil naqsh — avto-yakun ko'p
	// instansda ham DUBLIKAT bo'lmasligi kerak, aks holda bitta dars uchun
	// yozuvni to'xtatish/xonani o'chirish ikki marta bajarilardi.
	ClaimEnd(ctx context.Context, id string, endedAt time.Time) (bool, error)
}

package repository

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

type RecordingRepository interface {
	Create(ctx context.Context, rec *entity.Recording) error
	GetByID(ctx context.Context, id string) (*entity.Recording, error)
	GetByEgressID(ctx context.Context, egressID string) (*entity.Recording, error)
	ListByLesson(ctx context.Context, lessonID string) ([]*entity.Recording, error)
	UpdateStatus(ctx context.Context, id, status string) error
	MarkReady(ctx context.Context, egressID, objectKey string, durationSec int, sizeBytes int64, endedAt time.Time) error
	MarkFailed(ctx context.Context, egressID string, endedAt time.Time) error

	// ── Retention (30 kunlik saqlash) — `worker.RetentionWorker` ishlatadi ──

	// ListExpired — saqlash muddati o'tgan `ready` yozuvlar
	// (tugash vaqti `endedBefore` dan oldin).
	ListExpired(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error)
	// ClaimExpire atomik ravishda yozuvni `expired` deb belgilaydi (faqat hamon
	// `ready` bo'lsa true). Ko'p instansda bitta faylni ikki marta o'chirishning
	// oldini oladi — `ClaimReminder` bilan bir xil naqsh.
	ClaimExpire(ctx context.Context, id string, deletedAt time.Time) (bool, error)
	// ListExpiringUnwarned — muddati yaqinlashgan, lekin ogohlantirish hali
	// yuborilmagan `ready` yozuvlar.
	ListExpiringUnwarned(ctx context.Context, endedBefore time.Time, limit uint64) ([]*entity.Recording, error)
	// ClaimRetentionWarning atomik ravishda "ogohlantirildi" deb belgilaydi
	// (faqat hali belgilanmagan bo'lsa true) — dublikat bildirishnoma bo'lmaydi.
	ClaimRetentionWarning(ctx context.Context, id string) (bool, error)

	// Qayta kodlash navbati (CRF) — `worker.TranscodeWorker` ishlatadi.
	EnqueueTranscode(ctx context.Context, egressID string) error
	ClaimTranscode(ctx context.Context) (*entity.Recording, error)
	FinishTranscode(ctx context.Context, id string, newSize, originalSize int64) error
	FailTranscode(ctx context.Context, id string) error
	RequeueStaleTranscodes(ctx context.Context, olderThan time.Time) (int64, error)
}

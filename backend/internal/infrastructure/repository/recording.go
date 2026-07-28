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

	// Qayta kodlash navbati (CRF) — `worker.TranscodeWorker` ishlatadi.
	EnqueueTranscode(ctx context.Context, egressID string) error
	ClaimTranscode(ctx context.Context) (*entity.Recording, error)
	FinishTranscode(ctx context.Context, id string, newSize, originalSize int64) error
	FailTranscode(ctx context.Context, id string) error
	RequeueStaleTranscodes(ctx context.Context, olderThan time.Time) (int64, error)
}

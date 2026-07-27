package recording

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
)

// LiveKit — recording usecase talab qiladigan LiveKit operatsiyalari (DIP).
type LiveKit interface {
	Enabled() bool
	StartRoomRecording(ctx context.Context, roomName, objectKey string, s3 livekit.S3Config) (string, error)
	StopRecording(ctx context.Context, egressID string) error
}

type UseCase interface {
	// StartRecording — mentor yozib olishni boshlaydi (LiveKit Egress → MinIO).
	StartRecording(ctx context.Context, mentorID, lessonID string) (*entity.Recording, error)
	// StopRecording — yozib olishni to'xtatadi.
	StopRecording(ctx context.Context, mentorID, recordingID string) error
	// ListByLesson — dars yozuvlari (mentor).
	ListByLesson(ctx context.Context, mentorID, lessonID string) ([]*entity.Recording, error)
	// DownloadURL — tayyor yozuv uchun vaqtinchalik MinIO presigned havolasi.
	DownloadURL(ctx context.Context, mentorID, recordingID string) (*entity.RecordingDownload, error)
	// HandleEgress — LiveKit Egress webhook'idan kelgan yakuniy holatni qayta ishlaydi.
	HandleEgress(ctx context.Context, egressID string, completed bool, objectKey string, durationSec int, sizeBytes int64) error
}

package recording

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/audit"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/metrics"
	"github.com/zoom/darsly/internal/usecase/shared"
)

const downloadTTL = time.Hour

type useCase struct {
	repo       repository.RecordingRepository
	lessonRepo repository.LessonRepository
	livekit    LiveKit
	minio      minio.Client
	s3         livekit.S3Config
	log        logger.Logger
}

func New(repo repository.RecordingRepository, lessonRepo repository.LessonRepository, lk LiveKit, mc minio.Client, s3 livekit.S3Config, log logger.Logger) UseCase {
	return &useCase{repo: repo, lessonRepo: lessonRepo, livekit: lk, minio: mc, s3: s3, log: log}
}

func (uc *useCase) StartRecording(ctx context.Context, mentorID, lessonID string) (*entity.Recording, error) {
	// Avval egalik tekshiriladi — video servis o'chirilgan bo'lsa ham begona dars
	// yozuvi 403 bo'lib qolsin (500 bilan niqoblanmasin).
	l, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID)
	if err != nil {
		return nil, err
	}
	if !uc.livekit.Enabled() {
		return nil, apperr.Internal(fmt.Errorf("video service (LiveKit) is not configured"))
	}
	if l.Status != entity.LessonStatusLive {
		return nil, apperr.BadRequest("lesson is not live")
	}

	recID := uuid.NewString()
	objectKey := fmt.Sprintf("recordings/%s/%s.mp4", lessonID, recID)
	roomName := "lesson_" + lessonID

	// Egress chaqiruvi klient so'rovi hayotidan ajratilgan va cheklangan (WriteTimeout=30s'dan past),
	// shunda mentor uzilib qolsa ham yozib olish boshlanishi buzilmaydi.
	egressCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	egressID, err := uc.livekit.StartRoomRecording(egressCtx, roomName, objectKey, uc.s3)
	if err != nil {
		metrics.LiveKitErrors.WithLabelValues("egress_start").Inc()
		uc.log.Error(ctx, "recording.Start: egress failed", logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil, apperr.Internal(fmt.Errorf("could not start recording: %w", err))
	}

	now := time.Now().UTC()
	rec := &entity.Recording{
		ID:        recID,
		LessonID:  lessonID,
		EgressID:  egressID,
		ObjectKey: objectKey,
		Status:    entity.RecordingStatusRecording,
		StartedAt: now,
		CreatedAt: now,
	}
	if err := uc.repo.Create(ctx, rec); err != nil {
		uc.log.Error(ctx, "recording.Start: db error", logger.SafeString("err", err.Error()))
		// Kompensatsiya: DB yozib bo'lmadi — egressni to'xtatamiz, aks holda orphan yozuv qoladi.
		if stopErr := uc.livekit.StopRecording(egressCtx, egressID); stopErr != nil {
			uc.log.Error(ctx, "recording.Start: orphan egress stop failed", logger.String("egress_id", egressID), logger.SafeString("err", stopErr.Error()))
		}
		return nil, err
	}

	metrics.RecordingsStarted.Inc()
	audit.Record(ctx, uc.log, "recording.start", mentorID,
		logger.String("recording_id", recID), logger.String("lesson_id", lessonID), logger.String("egress_id", egressID))
	return rec, nil
}

func (uc *useCase) StopRecording(ctx context.Context, mentorID, recordingID string) error {
	// Yaroqsiz UUID DB'ga yetmasin (22P02 → 500). lessonID'ni OwnedLesson tekshiradi.
	if err := shared.ValidateID(recordingID, "recording"); err != nil {
		return err
	}
	rec, err := uc.repo.GetByID(ctx, recordingID)
	if err != nil {
		return err
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, rec.LessonID); err != nil {
		return err
	}
	if rec.Status != entity.RecordingStatusRecording {
		return apperr.BadRequest("recording is not active")
	}

	if err := uc.livekit.StopRecording(ctx, rec.EgressID); err != nil {
		uc.log.Error(ctx, "recording.Stop: egress stop failed", logger.String("egress_id", rec.EgressID), logger.SafeString("err", err.Error()))
		return apperr.Internal(fmt.Errorf("could not stop recording: %w", err))
	}
	// Yakuniy holat webhook orqali keladi; hozircha "processing".
	if err := uc.repo.UpdateStatus(ctx, rec.ID, entity.RecordingStatusProcessing); err != nil {
		return err
	}
	audit.Record(ctx, uc.log, "recording.stop", mentorID, logger.String("recording_id", rec.ID))
	return nil
}

func (uc *useCase) ListByLesson(ctx context.Context, mentorID, lessonID string) ([]*entity.Recording, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	return uc.repo.ListByLesson(ctx, lessonID)
}

func (uc *useCase) DownloadURL(ctx context.Context, mentorID, recordingID string) (*entity.RecordingDownload, error) {
	if err := shared.ValidateID(recordingID, "recording"); err != nil {
		return nil, err
	}
	rec, err := uc.repo.GetByID(ctx, recordingID)
	if err != nil {
		return nil, err
	}
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, rec.LessonID); err != nil {
		return nil, err
	}
	if rec.Status != entity.RecordingStatusReady {
		return nil, apperr.BadRequest("recording is not ready yet")
	}

	url, err := uc.minio.PresignedURL(ctx, rec.ObjectKey, downloadTTL)
	if err != nil {
		uc.log.Error(ctx, "recording.Download: presign failed", logger.SafeString("err", err.Error()))
		return nil, apperr.Internal(fmt.Errorf("could not generate download link: %w", err))
	}
	return &entity.RecordingDownload{
		URL:         url,
		ExpiresInS:  int(downloadTTL.Seconds()),
		DurationSec: rec.DurationSec,
		SizeBytes:   rec.SizeBytes,
	}, nil
}

func (uc *useCase) HandleEgress(ctx context.Context, egressID string, completed bool, objectKey string, durationSec int, sizeBytes int64) error {
	rec, err := uc.repo.GetByEgressID(ctx, egressID)
	if err != nil {
		// Noma'lum egress (bizniki emas yoki allaqachon o'chirilgan) — retry foydasiz,
		// qabul qilamiz (webhook 200 oladi).
		uc.log.Warn(ctx, "recording.HandleEgress: unknown egress", logger.String("egress_id", egressID))
		return nil
	}
	now := time.Now().UTC()
	if !completed {
		if err := uc.repo.MarkFailed(ctx, egressID, now); err != nil {
			uc.log.Error(ctx, "recording.HandleEgress: mark failed error", logger.SafeString("err", err.Error()))
			return err // tranzient DB xatosi — LiveKit qayta yuborsin
		}
		metrics.EgressResults.WithLabelValues("failed").Inc()
		uc.log.Warn(ctx, "recording failed", logger.String("recording_id", rec.ID), logger.String("egress_id", egressID))
		return nil
	}
	key := objectKey
	if key == "" {
		key = rec.ObjectKey // fallback: so'ralgan yo'l
	}
	if err := uc.repo.MarkReady(ctx, egressID, key, durationSec, sizeBytes, now); err != nil {
		// DB tranzient bo'lishi mumkin — non-200 qaytaramiz, LiveKit hodisani qayta yuboradi
		// (aks holda tayyor yozuv abadiy "processing"da qolardi).
		uc.log.Error(ctx, "recording.HandleEgress: mark ready failed", logger.SafeString("err", err.Error()))
		return err
	}
	metrics.EgressResults.WithLabelValues("ready").Inc()
	uc.log.Info(ctx, "recording ready", logger.String("recording_id", rec.ID), logger.String("object_key", key))
	return nil
}

package chat

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

type useCase struct {
	repo       repository.ChatRepository
	lessonRepo repository.LessonRepository
	userRepo   repository.UserRepository
	livekit    LiveKit
	log        logger.Logger
}

func New(repo repository.ChatRepository, lessonRepo repository.LessonRepository, userRepo repository.UserRepository, lk LiveKit, log logger.Logger) UseCase {
	return &useCase{repo: repo, lessonRepo: lessonRepo, userRepo: userRepo, livekit: lk, log: log}
}

func (uc *useCase) Send(ctx context.Context, mentorID, lessonID, body string) (*entity.ChatMessage, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}

	senderName := "Host"
	if m, err := uc.userRepo.GetByID(ctx, mentorID); err == nil {
		senderName = m.FullName
	}

	msg := &entity.ChatMessage{
		ID:             uuid.NewString(),
		LessonID:       lessonID,
		SenderIdentity: mentorID,
		SenderName:     senderName,
		Body:           body,
		CreatedAt:      time.Now().UTC(),
	}
	if err := uc.repo.Create(ctx, msg); err != nil {
		uc.log.Error(ctx, "chat.Send: db error", logger.String("lesson_id", lessonID), logger.SafeString("err", err.Error()))
		return nil, err
	}

	// Real-vaqt yetkazish — LiveKit data-channel orqali xonadagi hammага.
	if uc.livekit.Enabled() {
		if data, mErr := json.Marshal(msg); mErr == nil {
			if err := uc.livekit.SendData(ctx, shared.RoomName(lessonID), data); err != nil {
				uc.log.Warn(ctx, "chat.Send: livekit broadcast failed", logger.SafeString("err", err.Error()))
			}
		}
	}

	uc.log.Info(ctx, "chat message sent", logger.String("lesson_id", lessonID))
	return msg, nil
}

func (uc *useCase) History(ctx context.Context, mentorID, lessonID string, before *time.Time, limit int) ([]*entity.ChatMessage, error) {
	if _, err := shared.OwnedLesson(ctx, uc.lessonRepo, mentorID, lessonID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	return uc.repo.ListByLesson(ctx, lessonID, before, limit)
}

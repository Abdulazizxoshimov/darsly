package lesson

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/usecase/shared"
)

type useCase struct {
	repo   repository.LessonRepository
	hasher hasher.Hasher
	log    logger.Logger
}

func New(repo repository.LessonRepository, h hasher.Hasher, log logger.Logger) UseCase {
	return &useCase{repo: repo, hasher: h, log: log}
}

func (uc *useCase) Create(ctx context.Context, mentorID string, req *entity.CreateLessonReq) (*entity.Lesson, error) {
	slug, err := uc.uniqueSlug(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	duration := req.DurationMin
	if duration == 0 {
		duration = 60
	}
	l := &entity.Lesson{
		ID:                   uuid.NewString(),
		MentorID:             mentorID,
		Title:                req.Title,
		Description:          req.Description,
		ScheduledAt:          req.ScheduledAt,
		DurationMin:          duration,
		RecurrenceRule:       req.RecurrenceRule,
		JoinSlug:             slug,
		IsRecordingEnabled:   req.IsRecordingEnabled,
		IsWaitingRoomEnabled: req.IsWaitingRoomEnabled,
		Status:               entity.LessonStatusScheduled,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if req.Passcode != nil && *req.Passcode != "" {
		hashed, err := uc.hasher.Hash(*req.Passcode)
		if err != nil {
			return nil, fmt.Errorf("lesson.Create hash passcode: %w", err)
		}
		l.PasscodeHash = &hashed
	}

	if err := uc.repo.Create(ctx, l); err != nil {
		uc.log.Error(ctx, "lesson.Create: db error", logger.String("mentor_id", mentorID), logger.SafeString("err", err.Error()))
		return nil, err
	}
	l.HasPasscode = l.PasscodeHash != nil

	uc.log.Info(ctx, "lesson created", logger.String("id", l.ID), logger.String("slug", slug))
	return l, nil
}

func (uc *useCase) GetByID(ctx context.Context, mentorID, id string) (*entity.Lesson, error) {
	// shared.OwnedLesson: ID formatini tekshiradi (yaroqsiz UUID → 404, 500 emas),
	// so'ng yuklaydi va egalikni tekshiradi (avval bu mantiq shu yerda takrorlangan edi).
	return shared.OwnedLesson(ctx, uc.repo, mentorID, id)
}

func (uc *useCase) ListByMentor(ctx context.Context, mentorID string, filter *entity.LessonFilter) ([]*entity.Lesson, int, error) {
	return uc.repo.ListByMentor(ctx, mentorID, filter)
}

func (uc *useCase) Update(ctx context.Context, mentorID, id string, req *entity.UpdateLessonReq) (*entity.Lesson, error) {
	l, err := shared.OwnedLesson(ctx, uc.repo, mentorID, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		l.Title = *req.Title
	}
	if req.Description != nil {
		l.Description = req.Description
	}
	if req.ScheduledAt != nil {
		l.ScheduledAt = req.ScheduledAt
	}
	if req.DurationMin != nil {
		l.DurationMin = *req.DurationMin
	}
	if req.RecurrenceRule != nil {
		l.RecurrenceRule = req.RecurrenceRule
	}
	if req.IsLocked != nil {
		l.IsLocked = *req.IsLocked
	}
	if req.IsRecordingEnabled != nil {
		l.IsRecordingEnabled = *req.IsRecordingEnabled
	}
	if req.IsWaitingRoomEnabled != nil {
		l.IsWaitingRoomEnabled = *req.IsWaitingRoomEnabled
	}
	if req.Status != nil {
		l.Status = *req.Status
	}

	// Parol boshqaruvi: RemovePasscode > Passcode
	switch {
	case req.RemovePasscode:
		l.PasscodeHash = nil
	case req.Passcode != nil && *req.Passcode != "":
		hashed, err := uc.hasher.Hash(*req.Passcode)
		if err != nil {
			return nil, fmt.Errorf("lesson.Update hash passcode: %w", err)
		}
		l.PasscodeHash = &hashed
	}

	if err := uc.repo.Update(ctx, l); err != nil {
		uc.log.Error(ctx, "lesson.Update: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return nil, err
	}
	l.HasPasscode = l.PasscodeHash != nil

	uc.log.Info(ctx, "lesson updated", logger.String("id", id))
	return l, nil
}

func (uc *useCase) Delete(ctx context.Context, mentorID, id string) error {
	if _, err := shared.OwnedLesson(ctx, uc.repo, mentorID, id); err != nil {
		return err
	}
	if err := uc.repo.SoftDelete(ctx, id); err != nil {
		uc.log.Error(ctx, "lesson.Delete: db error", logger.String("id", id), logger.SafeString("err", err.Error()))
		return err
	}
	uc.log.Info(ctx, "lesson deleted", logger.String("id", id))
	return nil
}

// uniqueSlug o'qishga qulay join-slug generatsiya qiladi (collision tekshiruvi bilan).
// Namuna: "kav-teng-25iy".
func (uc *useCase) uniqueSlug(ctx context.Context) (string, error) {
	for range 6 {
		slug := generateSlug()
		exists, err := uc.repo.SlugExists(ctx, slug)
		if err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		uc.log.Warn(ctx, "lesson slug collision, retrying", logger.String("slug", slug))
	}
	return "", apperr.Internal(fmt.Errorf("could not generate unique join_slug after retries"))
}

// safeAlphabet — chalkash belgilarsiz (0/o, 1/l/i yo'q).
const safeAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// generateSlug "xxx-xxxx-xxx" ko'rinishida slug qaytaradi.
func generateSlug() string {
	groups := []int{3, 4, 3}
	buf := make([]byte, 0, 12)
	for gi, n := range groups {
		if gi > 0 {
			buf = append(buf, '-')
		}
		for range n {
			buf = append(buf, safeAlphabet[randIndex(len(safeAlphabet))])
		}
	}
	return string(buf)
}

func randIndex(n int) int {
	b := make([]byte, 1)
	// crypto/rand xato bermaydi deb faraz qilinadi; xato bo'lsa 0 qaytadi.
	if _, err := rand.Read(b); err != nil {
		return 0
	}
	return int(b[0]) % n
}

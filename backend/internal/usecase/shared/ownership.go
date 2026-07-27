// Package shared — usecase'lar orasida umumiy yordamchilar.
package shared

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// OwnedLesson darsni yuklaydi va mentorID egaligini tekshiradi.
// Bir xil "topilmasa NotFound, egasi bo'lmasa Forbidden" mantig'i barcha usecase'larda
// takrorlanmasin (DRY). Repo GetByID allaqachon NotFound qaytaradi.
func OwnedLesson(ctx context.Context, repo repository.LessonRepository, mentorID, lessonID string) (*entity.Lesson, error) {
	// Yaroqsiz UUID DB'ga yetmasin (22P02 → 500 + Sentry shovqini). Qarang: ValidateID.
	if err := ValidateID(lessonID, "lesson"); err != nil {
		return nil, err
	}
	l, err := repo.GetByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if l.MentorID != mentorID {
		return nil, apperr.Forbidden("you do not own this lesson")
	}
	return l, nil
}

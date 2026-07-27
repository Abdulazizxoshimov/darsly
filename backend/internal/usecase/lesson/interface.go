package lesson

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

type UseCase interface {
	Create(ctx context.Context, mentorID string, req *entity.CreateLessonReq) (*entity.Lesson, error)
	GetByID(ctx context.Context, mentorID, id string) (*entity.Lesson, error)
	ListByMentor(ctx context.Context, mentorID string, filter *entity.LessonFilter) ([]*entity.Lesson, int, error)
	Update(ctx context.Context, mentorID, id string, req *entity.UpdateLessonReq) (*entity.Lesson, error)
	Delete(ctx context.Context, mentorID, id string) error
}

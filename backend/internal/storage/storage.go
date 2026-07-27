package storage

import (
	"github.com/zoom/darsly/internal/infrastructure/repository"
	pgRepo "github.com/zoom/darsly/internal/infrastructure/repository/postgres"
	"github.com/zoom/darsly/internal/pkg/postgres"
)

// Storage barcha repository implementatsiyalarini bir joyda saqlaydi.
// Yangi domen qo'shilganda shu yerga repository qo'shiladi.
type Storage struct {
	User         repository.UserRepository
	Auth         repository.AuthRepository
	Lesson       repository.LessonRepository
	WaitingRoom  repository.WaitingRoomRepository
	Recording    repository.RecordingRepository
	Notification repository.NotificationRepository
	Chat         repository.ChatRepository
	Poll         repository.PollRepository
}

func New(pg *postgres.Postgres) *Storage {
	return &Storage{
		User:         pgRepo.NewUserRepo(pg),
		Auth:         pgRepo.NewAuthRepo(pg),
		Lesson:       pgRepo.NewLessonRepo(pg),
		WaitingRoom:  pgRepo.NewWaitingRoomRepo(pg),
		Recording:    pgRepo.NewRecordingRepo(pg),
		Notification: pgRepo.NewNotificationRepo(pg),
		Chat:         pgRepo.NewChatRepo(pg),
		Poll:         pgRepo.NewPollRepo(pg),
	}
}

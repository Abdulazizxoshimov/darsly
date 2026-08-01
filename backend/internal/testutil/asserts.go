package testutil

import (
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	"github.com/zoom/darsly/internal/infrastructure/repository"
	"github.com/zoom/darsly/internal/infrastructure/telegram"
	"github.com/zoom/darsly/internal/pkg/token"
	"github.com/zoom/darsly/internal/usecase/chat"
	"github.com/zoom/darsly/internal/usecase/recording"
	"github.com/zoom/darsly/internal/usecase/room"
)

// Kompilyatsiya vaqtida faqe'lar haqiqiy interfeyslarni qanoatlantirishini tekshiradi.
var (
	_ repository.UserRepository         = (*FakeUserRepo)(nil)
	_ repository.AuthRepository         = (*FakeAuthRepo)(nil)
	_ repository.LessonRepository       = (*FakeLessonRepo)(nil)
	_ repository.WaitingRoomRepository  = (*FakeWaitingRepo)(nil)
	_ repository.NotificationRepository = (*FakeNotifRepo)(nil)
	_ repository.RecordingRepository    = (*FakeRecordingRepo)(nil)
	_ repository.ChatRepository         = (*FakeChatRepo)(nil)
	_ repository.PollRepository         = (*FakePollRepo)(nil)
	_ token.Maker                       = (*FakeTokenMaker)(nil)
	_ redis.Cache                       = (*FakeCache)(nil)
	_ minio.Client                      = (*FakeMinio)(nil)
	_ room.UseCase                      = (*FakeRoomUC)(nil)
	_ room.LiveKit                      = (*FakeLiveKit)(nil)
	_ chat.LiveKit                      = (*FakeLiveKit)(nil)
	_ recording.LiveKit                 = (*FakeLiveKit)(nil)
	_ repository.TelegramRepository     = (*FakeTelegramRepo)(nil)
	// FakeTelegram BUTUN `telegram.Client` sirtini qoplaydi — shu bilan bot
	// ishchisini ham, `recording.Telegram` (tiklash) ni ham sinash mumkin.
	_ telegram.Client    = (*FakeTelegram)(nil)
	_ recording.Telegram = (*FakeTelegram)(nil)
)

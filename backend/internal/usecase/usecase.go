package usecase

import (
	"time"

	emailpkg "github.com/zoom/darsly/internal/infrastructure/email"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/infrastructure/minio"
	"github.com/zoom/darsly/internal/infrastructure/redis"
	ws "github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/pkg/hasher"
	"github.com/zoom/darsly/internal/pkg/logger"
	"github.com/zoom/darsly/internal/pkg/token"
	"github.com/zoom/darsly/internal/storage"
	"github.com/zoom/darsly/internal/usecase/auth"
	"github.com/zoom/darsly/internal/usecase/chat"
	"github.com/zoom/darsly/internal/usecase/joinlink"
	"github.com/zoom/darsly/internal/usecase/lesson"
	"github.com/zoom/darsly/internal/usecase/notification"
	"github.com/zoom/darsly/internal/usecase/poll"
	"github.com/zoom/darsly/internal/usecase/recording"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/roomstate"
	"github.com/zoom/darsly/internal/usecase/user"
	"github.com/zoom/darsly/internal/usecase/waitingroom"
)

// UseCases barcha usecase interfeyslarini bir joyda saqlaydi.
// Yangi domen qo'shilganda shu yerga qo'shiladi.
type UseCases struct {
	Auth         auth.UseCase
	User         user.UseCase
	Lesson       lesson.UseCase
	Room         room.UseCase
	RoomState    roomstate.UseCase
	WaitingRoom  waitingroom.UseCase
	Recording    recording.UseCase
	Notification notification.UseCase
	Chat         chat.UseCase
	Poll         poll.UseCase
	JoinLink     joinlink.UseCase
}

// Deps — UseCases uchun tashqi bog'liqliklar.
type Deps struct {
	Store           *storage.Storage
	TokenMaker      token.Maker
	Hasher          hasher.Hasher
	Minio           minio.Client
	Cache           redis.Cache
	Log             logger.Logger
	Hub             *ws.Hub
	EmailSender     emailpkg.Sender
	LiveKit         *livekit.Client
	RecordingS3     livekit.S3Config
	RefreshTTL      time.Duration
	FrontendBaseURL string
}

func New(d Deps) *UseCases {
	// Tartib muhim: `recording` `room`ga bog'liq emas, `room` esa majburiy
	// yozib olish uchun unga bog'liq (room.Recorder). Shuning uchun avval
	// recording yasaladi va room'ga uzatiladi.
	recordingUC := recording.New(d.Store.Recording, d.Store.Lesson, d.LiveKit, d.Minio, d.RecordingS3, d.Log)
	// roomstate `room`dan OLDIN yasaladi: `room` unga bog'liq (ruxsat berilganda
	// qo'lni tushirish, dars tugaganda tozalash), teskarisi esa yo'q.
	roomStateUC := roomstate.New(d.Store.Lesson, d.LiveKit, d.Cache, d.Log)
	roomUC := room.New(d.Store.Lesson, d.Store.User, d.LiveKit, d.Cache, d.Log, recordingUC, roomStateUC)
	waitingUC := waitingroom.New(d.Store.WaitingRoom, d.Store.Lesson, roomUC, d.Hub, d.Cache, d.Log)
	return &UseCases{
		Auth:         auth.New(d.Store.User, d.Store.Auth, d.TokenMaker, d.Hasher, 24*time.Hour, d.RefreshTTL, d.EmailSender, d.FrontendBaseURL, d.Log),
		User:         user.New(d.Store.User, d.Hasher, d.TokenMaker, d.Log),
		Lesson:       lesson.New(d.Store.Lesson, d.Hasher, d.Log),
		Room:         roomUC,
		RoomState:    roomStateUC,
		WaitingRoom:  waitingUC,
		Recording:    recordingUC,
		Notification: notification.New(d.Store.Notification, d.Hub, d.Log),
		Chat:         chat.New(d.Store.Chat, d.Store.Lesson, d.Store.User, d.LiveKit, d.Cache, d.Log),
		Poll:         poll.New(d.Store.Poll, d.Store.Lesson, d.Log),
		JoinLink:     joinlink.New(d.Store.Lesson, d.Store.User, d.Hasher, d.Cache, roomUC, waitingUC, d.Log),
	}
}

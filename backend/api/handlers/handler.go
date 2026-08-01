package handlers

import (
	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/usecase/archive"
	"github.com/zoom/darsly/internal/usecase/auth"
	"github.com/zoom/darsly/internal/usecase/chat"
	"github.com/zoom/darsly/internal/usecase/joinlink"
	"github.com/zoom/darsly/internal/usecase/lesson"
	"github.com/zoom/darsly/internal/usecase/notification"
	"github.com/zoom/darsly/internal/usecase/poll"
	"github.com/zoom/darsly/internal/usecase/recording"
	"github.com/zoom/darsly/internal/usecase/room"
	"github.com/zoom/darsly/internal/usecase/roomstate"
	"github.com/zoom/darsly/internal/usecase/telegram"
	"github.com/zoom/darsly/internal/usecase/user"
	"github.com/zoom/darsly/internal/usecase/waitingroom"
)

// Handler barcha usecase'larni HTTP qatlamiga ulaydigan yagona struct.
// Yangi domen qo'shilganda shu yerga maydon qo'shiladi (app/wire.go dan to'ldiriladi).
type Handler struct {
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
	// Telegram — dars arxivi integratsiyasi (bog'lash oqimi).
	// Sozlanmagan bo'lsa `Status().Enabled=false` qaytadi (nil emas).
	Telegram telegram.UseCase
	Archive  archive.UseCase

	Hub     *websocket.Hub
	LiveKit *livekit.Client

	// Config flag'lari
	AllowOpenRegistration bool
	FrontendBaseURL       string
	// AppConfig — ochiq /app-config javobi (startup'da config'dan yig'iladi).
	AppConfig entity.AppConfig
}

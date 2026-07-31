package app

import (
	"github.com/zoom/darsly/api/handlers"
	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/infrastructure/livekit"
	"github.com/zoom/darsly/internal/infrastructure/websocket"
	"github.com/zoom/darsly/internal/pkg/config"
	"github.com/zoom/darsly/internal/usecase"
)

// BuildHandler wired usecase'larni HTTP Handler struct'iga bog'laydi.
// Production server (Run) va integration test harness bir xil ishlatadi.
func BuildHandler(uc *usecase.UseCases, hub *websocket.Hub, lk *livekit.Client, cfg *config.Config) *handlers.Handler {
	return &handlers.Handler{
		Auth:                  uc.Auth,
		User:                  uc.User,
		Lesson:                uc.Lesson,
		Room:                  uc.Room,
		RoomState:             uc.RoomState,
		WaitingRoom:           uc.WaitingRoom,
		Recording:             uc.Recording,
		Notification:          uc.Notification,
		Chat:                  uc.Chat,
		Poll:                  uc.Poll,
		JoinLink:              uc.JoinLink,
		Hub:                   hub,
		LiveKit:               lk,
		AllowOpenRegistration: cfg.App.AllowOpenRegistration,
		FrontendBaseURL:       cfg.App.FrontendBaseURL,
		// Mobil versiya siyosati env'dan bir marta o'qiladi (DB'ga tegmaydi).
		AppConfig: entity.AppConfig{
			AllowOpenRegistration: cfg.App.AllowOpenRegistration,
			Android: entity.AppPlatformConfig{
				MinVersion:    cfg.Mobile.AndroidMinVersion,
				LatestVersion: cfg.Mobile.AndroidLatestVersion,
				APKURL:        cfg.Mobile.AndroidAPKURL,
				ForceUpdate:   cfg.Mobile.AndroidForceUpdate,
				ReleaseNotes:  cfg.Mobile.AndroidReleaseNotes,
			},
		},
	}
}

package room

import (
	"context"

	"github.com/zoom/darsly/internal/entity"
)

// LiveKit — room usecase talab qiladigan LiveKit operatsiyalari (DIP: konkret
// klient emas, interfeys). `*livekit.Client` buni structural qondiradi; test'da mock.
type LiveKit interface {
	Enabled() bool
	WSURL() string
	AccessToken(roomName, identity, displayName string, isHost bool) (string, error)
	EnsureRoom(ctx context.Context, name string, emptyTimeoutSec uint32) error
	DeleteRoom(ctx context.Context, name string) error
	ListParticipantViews(ctx context.Context, room string) ([]entity.RoomParticipant, error)
	MuteParticipant(ctx context.Context, room, identity string, audioOnly bool) error
	RemoveParticipant(ctx context.Context, room, identity string) error
	SetParticipantPublish(ctx context.Context, room, identity string, canPublish bool) error
}

type UseCase interface {
	// HostToken — mentor (egasi) uchun host tokeni; xonani yaratadi va darsni "live" qiladi.
	HostToken(ctx context.Context, mentorID, lessonID string) (*entity.RoomToken, error)
	// ParticipantToken — validatsiyadan o'tgan ishtirokchi uchun token (joinlink chaqiradi).
	ParticipantToken(ctx context.Context, lesson *entity.Lesson, identity, displayName string) (*entity.RoomToken, error)
	// EndLesson — darsni yakunlaydi va xonani yopadi (barchani uzadi).
	EndLesson(ctx context.Context, mentorID, lessonID string) error

	// ── Host boshqaruvi (mentor/co-host) ──
	// ListParticipants — xonadagi ishtirokchilar (real-vaqt, LiveKit'dan).
	ListParticipants(ctx context.Context, mentorID, lessonID string) ([]entity.RoomParticipant, error)
	// MuteParticipant — ishtirokchini mute qiladi (audioOnly=true → faqat mikrofon).
	MuteParticipant(ctx context.Context, mentorID, lessonID, identity string, audioOnly bool) error
	// MuteAll — host'dan tashqari hammani mute qiladi.
	MuteAll(ctx context.Context, mentorID, lessonID string) error
	// RemoveParticipant — ishtirokchini xonadan chiqarib yuboradi (kick).
	RemoveParticipant(ctx context.Context, mentorID, lessonID, identity string) error
	// SetSpeakPermission — studentga media publish (so'zlash) ruxsatini beradi/qaytaradi.
	// Webinar modeli: student default publish qila olmaydi; host "qo'l ko'targanda" ruxsat beradi.
	SetSpeakPermission(ctx context.Context, mentorID, lessonID, identity string, canPublish bool) error
}

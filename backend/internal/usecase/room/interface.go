package room

import (
	"context"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

// LiveKit — room usecase talab qiladigan LiveKit operatsiyalari (DIP: konkret
// klient emas, interfeys). `*livekit.Client` buni structural qondiradi; test'da mock.
type LiveKit interface {
	Enabled() bool
	// ClientWSURL — klientga beriladigan signaling manzili. requestHost —
	// klientning API so'rovidagi Host ("auto" rejim uchun; bo'sh bo'lsa default).
	ClientWSURL(requestHost string) string
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
	// EnforceJoin — `participant_joined` webhook'i: chiqarilgan (dars ban'i) yoki
	// mentor qora ro'yxatidagi (displayName bo'yicha) ishtirokchi qaytib kirgan
	// bo'lsa uni uzadi. Batafsil izoh implementatsiyada.
	EnforceJoin(ctx context.Context, roomName, identity, displayName string)
	// EnforceAudioPolicy — `track_published` (audio) webhook'i: dars ovoz
	// sozlamalarini server tomonda qo'llaydi — mute_on_entry (birinchi publish'da
	// bir marta) va allow_self_unmute=false (har publish'da qayta mute).
	EnforceAudioPolicy(ctx context.Context, roomName, identity string)

	// ── Avto-yakun (PRODUCT.md «Dars hayoti») ──
	// NoteRoomEmpty — `participant_left` (oxirgi ishtirokchi) yoki `room_finished`
	// webhook'i: bo'sh xona grace hisobi shu lahzadan boshlanadi.
	NoteRoomEmpty(ctx context.Context, roomName string)
	// NoteRoomOccupied — `participant_joined` webhook'i: bo'shlik hisobi bekor qilinadi.
	NoteRoomOccupied(ctx context.Context, roomName string)
	// SweepAutoEnd — jonli darslarni tekshiradi va yakunlash shartiga tushganini
	// yakunlaydi (4 soatlik limit yoki bo'sh xona grace'i). Yakunlangan darslar soni.
	// Idempotent: atomik `ClaimEnd` tufayli dublikat yakun bo'lmaydi.
	SweepAutoEnd(ctx context.Context, maxDuration, emptyGrace time.Duration) int

	// ── Host boshqaruvi (mentor/co-host) ──
	// ListParticipants — xonadagi ishtirokchilar (real-vaqt, LiveKit'dan).
	ListParticipants(ctx context.Context, mentorID, lessonID string) ([]entity.RoomParticipant, error)
	// MuteParticipant — ishtirokchini mute qiladi (audioOnly=true → faqat mikrofon).
	MuteParticipant(ctx context.Context, mentorID, lessonID, identity string, audioOnly bool) error
	// MuteAll — host'dan tashqari hammani mute qiladi. allowSelfUnmute berilsa
	// (nil emas) dars bayrog'i ham shu qiymatga yangilanadi (Zoom checkbox'i).
	MuteAll(ctx context.Context, mentorID, lessonID string, allowSelfUnmute *bool) error
	// RemoveParticipant — ishtirokchini xonadan chiqarib yuboradi (kick).
	// scope: "lesson" (default, faqat shu dars) yoki "mentor" (doimiy qora ro'yxat).
	RemoveParticipant(ctx context.Context, mentorID, lessonID, identity, scope string) error
	// SetSpeakPermission — studentning media publish (so'zlash) ruxsatini boshqaradi
	// (masalan tartib buzarga kamera+mikrofonni butunlay yopish va qaytarish).
	SetSpeakPermission(ctx context.Context, mentorID, lessonID, identity string, canPublish bool) error

	// ── Mentor qora ro'yxati (doimiy ban) ──
	// ListBlocklist — mentorning doimiy qora ro'yxati.
	ListBlocklist(ctx context.Context, mentorID string) ([]*entity.BlocklistEntry, error)
	// Unblock — qora ro'yxat yozuvini o'chiradi (unban).
	Unblock(ctx context.Context, mentorID, entryID string) error
}

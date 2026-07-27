package entity

import "time"

// Lesson — rejalashtirilgan yoki jonli video-dars.
type Lesson struct {
	ID                   string     `json:"id"`
	MentorID             string     `json:"mentor_id"`
	Title                string     `json:"title"`
	Description          *string    `json:"description,omitempty"`
	ScheduledAt          *time.Time `json:"scheduled_at,omitempty"`
	DurationMin          int        `json:"duration_min"`
	RecurrenceRule       *string    `json:"recurrence_rule,omitempty"` // iCal RRULE (haftalik takrorlanish)
	JoinSlug             string     `json:"join_slug"`
	PasscodeHash         *string    `json:"-"`
	HasPasscode          bool       `json:"has_passcode"` // hisoblanadigan (persist qilinmaydi)
	IsLocked             bool       `json:"is_locked"`
	IsRecordingEnabled   bool       `json:"is_recording_enabled"`
	IsWaitingRoomEnabled bool       `json:"is_waiting_room_enabled"`
	Status               string     `json:"status"` // scheduled | live | ended | cancelled
	StartedAt            *time.Time `json:"started_at,omitempty"`
	EndedAt              *time.Time `json:"ended_at,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
	DeletedAt            *time.Time `json:"-"`
}

// Dars statuslari
const (
	LessonStatusScheduled = "scheduled"
	LessonStatusLive      = "live"
	LessonStatusEnded     = "ended"
	LessonStatusCancelled = "cancelled"
)

type CreateLessonReq struct {
	Title                string     `json:"title"                   validate:"required,min=2,max=255"`
	Description          *string    `json:"description"             validate:"omitempty,max=2000"`
	ScheduledAt          *time.Time `json:"scheduled_at"`
	DurationMin          int        `json:"duration_min"            validate:"omitempty,min=5,max=1440"`
	RecurrenceRule       *string    `json:"recurrence_rule"         validate:"omitempty,max=255"`
	Passcode             *string    `json:"passcode"                validate:"omitempty,min=4,max=20"`
	IsRecordingEnabled   bool       `json:"is_recording_enabled"`
	IsWaitingRoomEnabled bool       `json:"is_waiting_room_enabled"`
}

type UpdateLessonReq struct {
	Title                *string    `json:"title"                   validate:"omitempty,min=2,max=255"`
	Description          *string    `json:"description"             validate:"omitempty,max=2000"`
	ScheduledAt          *time.Time `json:"scheduled_at"`
	DurationMin          *int       `json:"duration_min"            validate:"omitempty,min=5,max=1440"`
	RecurrenceRule       *string    `json:"recurrence_rule"         validate:"omitempty,max=255"`
	Passcode             *string    `json:"passcode"                validate:"omitempty,min=4,max=20"` // yangi parol o'rnatish
	RemovePasscode       bool       `json:"remove_passcode"`                                           // parolni olib tashlash
	IsLocked             *bool      `json:"is_locked"`
	IsRecordingEnabled   *bool      `json:"is_recording_enabled"`
	IsWaitingRoomEnabled *bool      `json:"is_waiting_room_enabled"`
	Status               *string    `json:"status"                  validate:"omitempty,oneof=scheduled live ended cancelled"`
}

type LessonFilter struct {
	Filter
	Status string `form:"status" json:"status"`
}

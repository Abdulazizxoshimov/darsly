package entity

import "time"

// Poll — dars ichidagi so'rovnoma/viktorina.
type Poll struct {
	ID        string     `json:"id"`
	LessonID  string     `json:"lesson_id"`
	Question  string     `json:"question"`
	Options   []string   `json:"options"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	ClosedAt  *time.Time `json:"closed_at,omitempty"`
}

// PollResults — so'rovnoma natijalari.
type PollResults struct {
	Poll   *Poll `json:"poll"`
	Counts []int `json:"counts"` // har variant uchun ovozlar soni
	Total  int   `json:"total"`
}

type CreatePollReq struct {
	Question string   `json:"question" validate:"required,min=1,max=500"`
	Options  []string `json:"options"  validate:"required,min=2,max=10,dive,min=1,max=200"`
}

// VoteReq — ovoz berish. Token — guest'ning LiveKit access token'i (room-auth).
type VoteReq struct {
	Token       string `json:"token"        validate:"required"`
	OptionIndex int    `json:"option_index" validate:"min=0"`
}

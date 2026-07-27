package entity

import "time"

// ChatMessage — dars ichidagi chat xabari (tarix uchun saqlanadi).
// Real-vaqt yetkazish LiveKit data-channel orqali; backend tarixni saqlaydi.
type ChatMessage struct {
	ID             string    `json:"id"`
	LessonID       string    `json:"lesson_id"`
	SenderIdentity string    `json:"sender_identity"`
	SenderName     string    `json:"sender_name"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}

type SendChatReq struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
}

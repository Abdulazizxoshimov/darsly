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
	// ToIdentity — shaxsiy xabar qabul qiluvchisi. nil = xonaga (ommaviy).
	// Yetkazish ham shunga qarab: ommaviy xabar butun xonaga, shaxsiysi esa
	// FAQAT ikki tomonga (`destination_identities`) yuboriladi — ya'ni maxfiylik
	// klientning "ko'rsatmaslik" xushmuomalaligiga tayanmaydi.
	ToIdentity *string `json:"to_identity,omitempty"`
}

type SendChatReq struct {
	Body string `json:"body" validate:"required,min=1,max=2000"`
	// To — shaxsiy xabar qabul qiluvchisi (LiveKit identity). Bo'sh = hammaga.
	To string `json:"to"`
}

// SendRoomChatReq — ochiq yo'l (guest): room-token bilan.
// Token'ga `validate:"required"` qo'yilmagan — [HandReq] izohidagi sabab bilan (401).
type SendRoomChatReq struct {
	Token string `json:"token"`
	Body  string `json:"body" validate:"required,min=1,max=2000"`
	To    string `json:"to"`
}

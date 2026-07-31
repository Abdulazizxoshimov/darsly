package entity

import "time"

// ChatFile — chat xabariga ilova qilingan fayl (MinIO'da).
//
// `URL` DB'da SAQLANMAYDI: u har o'qishda presigned havola sifatida qaytadan
// imzolanadi (`ChatFileURLTTL`). Bazaga yozilgan havola bir soatdan keyin
// o'lik bo'lardi va "tarixdagi eski fayllar ochilmaydi" degan jimgina
// nosozlikni berardi.
type ChatFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Mime string `json:"mime"`
	// URL — vaqtinchalik presigned havola (javob yasalayotganda hisoblanadi).
	URL string `json:"url,omitempty"`
	// ExpiresInS — havola necha soniyadan keyin o'ladi (klient keshini shunga moslaydi).
	ExpiresInS int `json:"expires_in_s,omitempty"`
	// Key — MinIO obyekt kaliti. `json:"-"` — TASHQARIGA CHIQMAYDI: bucket ichidagi
	// nomlash sxemasi hujumchiga boshqa fayllarni taxmin qilish yo'lini berardi.
	Key string `json:"-"`
}

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
	// File — ilova qilingan fayl (nil = oddiy matnli xabar).
	File *ChatFile `json:"file,omitempty"`
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

// ChatDeletedEvent — moderatsiya hodisasi (LiveKit data-channel).
//
// Klient shu ID li xabar pufagini ro'yxatdan olib tashlaydi. Xabar MAZMUNI
// yuborilmaydi — o'chirishning butun maqsadi uni tarqatmaslik edi.
// `kind` qiymati klientlar bilan shartnoma: `frontend/src/livekit/messaging.js`,
// `mobile/.../RoomDataParser.kt`.
type ChatDeletedEvent struct {
	Kind      string `json:"kind"` // doim "chat_deleted"
	ID        string `json:"id"`
	LessonID  string `json:"lesson_id"`
	DeletedBy string `json:"deleted_by"`
}

// NewChatDeletedEvent — [ChatDeletedEvent] konstruktori (kind'ni qo'lda yozish
// xatosi bo'lmasin).
func NewChatDeletedEvent(id, lessonID, deletedBy string) ChatDeletedEvent {
	return ChatDeletedEvent{Kind: "chat_deleted", ID: id, LessonID: lessonID, DeletedBy: deletedBy}
}

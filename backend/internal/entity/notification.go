package entity

import "time"

// Notification — foydalanuvchiga bildirishnoma (WS push + saqlanadi).
type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	LessonID  *string    `json:"lesson_id,omitempty"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// Bildirishnoma turlari
const (
	NotificationTypeLessonReminder = "lesson_reminder"
	NotificationTypeWaitingRoom    = "waiting_room"
	NotificationTypeSystem         = "system"
	// NotificationTypeRecordingExpiring — yozuv saqlash muddati tugayapti
	// (retention ogohlantirishi, PRODUCT.md №5). Mentor ulgurib yuklab olsin.
	NotificationTypeRecordingExpiring = "recording_expiring"
	// NotificationTypeTelegramFailed — yozuvni Telegram arxiviga yuborib
	// bo'lmadi (3 urinishdan keyin). Server nusxasi SAQLANIB QOLADI —
	// bildirishnoma matni buni ataylab ta'kidlaydi, aks holda mentor
	// "video yo'qoldi" deb tushunardi.
	NotificationTypeTelegramFailed = "telegram_upload_failed"
	// NotificationTypeRecordingArchived — yozuv serverdan Telegram arxiviga
	// ko'chirildi (30 kun o'tdi). Yo'qolgani emas — ochilganda tiklanadi.
	NotificationTypeRecordingArchived = "recording_archived"
)

type NotificationFilter struct {
	Filter
	UnreadOnly bool `form:"unread" json:"unread"`
}

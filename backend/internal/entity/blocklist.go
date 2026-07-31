package entity

import "time"

// BlocklistEntry — mentor darajasidagi doimiy qora ro'yxat yozuvi (№4).
//
// O'quvchida akkaunt yo'q va LiveKit identity har join'da yangi, shuning uchun
// doimiy moslik kaliti — DisplayName (katta-kichik harf farqsiz). Identity esa
// chiqarilgan paytdagi qiymat: joriy token oynasida (30 min) qaytib kirishni
// kesish va audit uchun saqlanadi.
type BlocklistEntry struct {
	ID          string    `json:"id"`
	MentorID    string    `json:"-"`
	Identity    string    `json:"identity"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// Ban qamrovi (kick paytida mentor tanlaydi).
const (
	// BanScopeLesson — faqat shu darsdan (Redis, dars bilan tugaydi). Default.
	BanScopeLesson = "lesson"
	// BanScopeMentor — mentorning HAMMA darslaridan doimiy (mentor_blocklist).
	BanScopeMentor = "mentor"
)

// RemoveParticipantReq — kick so'rovi tanasi (ixtiyoriy; bo'sh = scope "lesson").
type RemoveParticipantReq struct {
	Scope string `json:"scope" validate:"omitempty,oneof=lesson mentor"`
}

// MuteAllReq — "hammani o'chirish" so'rovi tanasi (ixtiyoriy).
// AllowSelfUnmute berilsa dars bayrog'i ham shu qiymatga yangilanadi
// (Zoom'dagi "Allow participants to unmute themselves" checkbox'i).
type MuteAllReq struct {
	AllowSelfUnmute *bool `json:"allow_self_unmute"`
}

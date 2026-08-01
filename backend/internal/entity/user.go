package entity

import "time"

// Mahsulot qat'iy qiymatlari: vaqt mintaqasi va til interfeysda SOZLANMAYDI
// (mobil va web profilidan olib tashlangan). Sabab: ilova butunlay o'zbekcha
// va foydalanuvchilar O'zbekistonda; erkin maydon bo'lganda noto'g'ri mintaqa
// dars vaqtlarini surib ko'rsatardi. Avvalgi default `UTC` edi.
const (
	DefaultTimezone = "Asia/Tashkent"
	DefaultLanguage = "uz"
)

type User struct {
	ID           string  `json:"id"`
	Email        string  `json:"email"`
	PasswordHash string  `json:"-"`
	FullName     string  `json:"full_name"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	Color        string  `json:"color"`
	Role         string  `json:"role"`
	Timezone     string  `json:"timezone"`
	Language     string  `json:"language"`
	IsActive     bool    `json:"is_active"`
	// TelegramUserID — bog'langan Telegram akkaunt (bot mentorni shu orqali
	// taniydi: dars tugagach «qaysi guruhga yuboray?» so'rovi kimga ketishi va
	// tugmani bosgan odam kimligi shundan aniqlanadi). nil = bog'lanmagan.
	// `json:"-"` — ID tashqariga chiqmaydi; holat `TelegramLinkStatus` da.
	TelegramUserID   *int64     `json:"-"`
	TelegramUsername *string    `json:"telegram_username,omitempty"`
	TelegramLinkedAt *time.Time `json:"telegram_linked_at,omitempty"`
	LastLoginAt      *time.Time `json:"last_login_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"-"`
}

type CreateUserReq struct {
	Email    string `json:"email"     validate:"required,email"`
	Password string `json:"password"  validate:"required,min=8,max=72"`
	FullName string `json:"full_name" validate:"required,min=2,max=255"`
	Role     string `json:"role"      validate:"omitempty,oneof=admin mentor student guest"`
}

type UpdateUserReq struct {
	FullName  *string `json:"full_name"  validate:"omitempty,min=2,max=255"`
	AvatarURL *string `json:"avatar_url" validate:"omitempty,url"`
	Color     *string `json:"color"      validate:"omitempty,len=7"`
	Timezone  *string `json:"timezone"   validate:"omitempty,max=64"`
	Language  *string `json:"language"   validate:"omitempty,max=8"`
	Role      *string `json:"role"       validate:"omitempty,oneof=admin mentor student guest"`
}

type ChangePasswordReq struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password"     validate:"required,min=8,max=72"`
}

// AdminResetPasswordReq — mentor boshqa foydalanuvchi parolini joriy-parolsiz
// tiklaganida (admin-reset) ishlatiladi. Joriy parol talab qilinmaydi.
type AdminResetPasswordReq struct {
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
}

type UserFilter struct {
	Filter
	Role     string `form:"role"      json:"role"`
	IsActive *bool  `form:"is_active" json:"is_active,omitempty"`
}

// UserShort is a lightweight user representation used in nested responses.
type UserShort struct {
	ID        string  `json:"id"`
	FullName  string  `json:"full_name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Color     string  `json:"color"`
	Email     string  `json:"email"`
}

// ToShort — ro'yxat javoblari uchun yengil ko'rinish (ortiqcha maydonlar va
// email'dan boshqa PII oshkor bo'lmaydi; past internetда payload kichrayadi).
func (u *User) ToShort() UserShort {
	return UserShort{
		ID:        u.ID,
		FullName:  u.FullName,
		AvatarURL: u.AvatarURL,
		Color:     u.Color,
		Email:     u.Email,
	}
}

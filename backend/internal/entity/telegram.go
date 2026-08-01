package entity

import "time"

// TelegramChat — bot a'zo bo'lgan guruh.
//
// Nega DB'da saqlanadi: Bot API'da "bot qaysi guruhlarda" degan metod YO'Q.
// Yagona manba — bot guruhga qo'shilganda/chiqarilganda keladigan
// `my_chat_member` yangilanishi. Uni ushlab shu jadvalga yozamiz, aks holda
// dars tugagach mentorga tanlash uchun ro'yxat bera olmaymiz.
type TelegramChat struct {
	ChatID int64  `json:"chat_id"`
	Title  string `json:"title"`
	Type   string `json:"type"`
	// MentorID — botni guruhga qo'shgan mentor (bog'langan bo'lsa). nil →
	// noma'lum (bot guruhga bog'lanmagan akkaunt tomonidan qo'shilgan).
	MentorID  *string   `json:"-"`
	IsActive  bool      `json:"is_active"`
	AddedAt   time.Time `json:"added_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TelegramLink — «Telegram bilan bog'lash» oqimining birinchi qadami.
//
// Mentor ilovada tugmani bosadi → server bir martalik kod beradi → mentor
// botga `/start <kod>` yuboradi. Nega kod: Telegram bizga foydalanuvchining
// kimligini aytmaydi, faqat o'z ID sini beradi; kod ikkalasini bog'laydigan
// yagona ishonchli ko'prik.
type TelegramLink struct {
	Code string `json:"code"`
	// DeepLink — bosish bilan botni ochib `/start <kod>` yuboradigan havola
	// (`https://t.me/<bot>?start=<kod>`). Bot username noma'lum bo'lsa bo'sh.
	DeepLink string `json:"deep_link,omitempty"`
	// ExpiresInS — kod muddati. Qisqa: kod aslida bir martalik parol.
	ExpiresInS int `json:"expires_in_s"`
}

// TelegramLinkStatus — mentorning Telegram bog'lanish holati.
type TelegramLinkStatus struct {
	// Enabled — integratsiya UMUMAN sozlanganmi (TELEGRAM_BOT_TOKEN bor).
	// false bo'lsa klient «Telegram bilan bog'lash» bo'limini ko'rsatmasin.
	Enabled  bool       `json:"enabled"`
	Linked   bool       `json:"linked"`
	Username *string    `json:"telegram_username,omitempty"`
	LinkedAt *time.Time `json:"linked_at,omitempty"`
	// Chats — mentor botni qo'shgan faol guruhlar.
	Chats []*TelegramChat `json:"chats,omitempty"`
}

// RecordingRestore — `POST /recordings/:id/restore` javobi.
//
// Tiklash 30-60 soniya davom etadi, ya'ni HTTP so'rovini ushlab turib
// bo'lmaydi (nginx/Caddy timeout, mobil tarmoq uzilishi). Shuning uchun
// endpoint 202 bilan darhol qaytadi va klient `GET /recordings/:id` ni poll
// qiladi.
type RecordingRestore struct {
	Status string `json:"status"`
	// PollAfterS — klient qancha kutib qayta so'rasin (thundering-herd yo'q).
	PollAfterS int `json:"poll_after_s"`
}

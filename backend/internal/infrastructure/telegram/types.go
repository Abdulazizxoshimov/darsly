package telegram

// Bot API turlarining MINIMAL to'plami.
//
// Nega tayyor kutubxona emas: bizga Bot API'ning ~10 metodi kerak, tayyor
// kutubxonalar esa butun API sirtini (to'lovlar, stikerlar, inline rejim)
// olib keladi va ular bilan birga o'z tranzport/retry siyosatini ham. Bizda
// esa aniq talab bor — 429 `retry_after` ni hurmat qilish va 2 GB fayllarni
// oqim (stream) bilan yuklash. Shu ikkisi tayyor yechimlarda odatda
// boshqacha ishlaydi, ya'ni baribir ustidan yozishga to'g'ri kelardi.
//
// Faqat BIZ O'QIYDIGAN maydonlar e'lon qilingan: qolganlari `json` tomonidan
// e'tiborsiz qoldiriladi va Telegram API o'sganda hech narsa buzilmaydi.

// User — Telegram foydalanuvchisi (yoki botning o'zi, `getMe`).
type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

// Chat — shaxsiy chat yoki guruh. `Type`: private | group | supergroup | channel.
type Chat struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// Message — yuborilgan yoki qabul qilingan xabar.
type Message struct {
	MessageID int64  `json:"message_id"`
	From      *User  `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Video     *Video `json:"video"`
	Document  *Doc   `json:"document"`
}

// Video — yuborilgan video (yuklashdan keyin `file_id` shu yerda keladi).
type Video struct {
	FileID   string `json:"file_id"`
	FileSize int64  `json:"file_size"`
	Duration int    `json:"duration"`
}

// Doc — hujjat (chat transkripti shu ko'rinishda ketadi).
type Doc struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

// File — `getFile` natijasi.
//
// ⚠️ `FilePath` ikki xil bo'ladi va butun yuklab olish mantig'i shunga tayanadi:
//   - rasmiy api.telegram.org: NISBIY yo'l (`videos/file_123.mp4`) →
//     `https://api.telegram.org/file/bot<token>/<path>` dan yuklanadi;
//   - Local Bot API Server (`--local`): ABSOLYUT lokal yo'l
//     (`/var/lib/telegram-bot-api/<bot_id>/videos/file_123.mp4`) → fayl
//     to'g'ridan-to'g'ri diskdan o'qiladi (HTTP orqali umuman yuklanmaydi).
//
// Ikkinchi holat aynan nima uchun local server kerakligining sababi: 1.9 GB
// videoni HTTP orqali qayta yuklab olish o'rniga uni ulashilgan volume'dan
// nusxalaymiz.
type File struct {
	FileID   string `json:"file_id"`
	FileSize int64  `json:"file_size"`
	FilePath string `json:"file_path"`
}

// Update — long-polling'dan keladigan yangilanish.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
	// MyChatMember — botning O'ZI guruhga qo'shilgan/chiqarilgan hodisa.
	// Bot qaysi guruhlarda ekanini bilishning YAGONA yo'li (Bot API'da
	// "mening guruhlarim" metodi yo'q).
	MyChatMember *ChatMemberUpdated `json:"my_chat_member"`
}

// CallbackQuery — inline tugma bosilishi.
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message"`
	Data    string   `json:"data"`
}

// ChatMemberUpdated — a'zolik holati o'zgarishi.
type ChatMemberUpdated struct {
	Chat          Chat       `json:"chat"`
	From          User       `json:"from"`
	NewChatMember ChatMember `json:"new_chat_member"`
	OldChatMember ChatMember `json:"old_chat_member"`
}

// ChatMember — a'zolik. `Status`: creator | administrator | member |
// restricted | left | kicked.
type ChatMember struct {
	Status string `json:"status"`
	User   User   `json:"user"`
}

// IsMember — bot hamon guruhda va yozish huquqiga egami.
func (m ChatMember) IsMember() bool {
	switch m.Status {
	case "creator", "administrator", "member":
		return true
	}
	return false
}

// InlineKeyboard — xabar tagidagi tugmalar (mentor guruh tanlaydi).
type InlineKeyboard struct {
	InlineKeyboard [][]InlineButton `json:"inline_keyboard"`
}

// InlineButton — bitta tugma. `CallbackData` 64 baytdan oshmasligi kerak
// (Telegram cheklovi) — shuning uchun kodlarimiz qisqa: `s:<rec8>:<chat>`.
type InlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// apiResponse — Bot API javobining umumiy qobig'i.
type apiResponse struct {
	OK          bool        `json:"ok"`
	Description string      `json:"description"`
	ErrorCode   int         `json:"error_code"`
	Parameters  *respParams `json:"parameters"`
	RawResult   rawJSON     `json:"result"`
}

type respParams struct {
	// RetryAfter — 429'da qancha kutish kerakligi (soniya). Telegram bu qiymatni
	// HURMAT QILISHNI talab qiladi; e'tiborsiz qoldirilsa bot vaqtincha
	// bloklanadi.
	RetryAfter int `json:"retry_after"`
	// MigrateToChatID — guruh supergroup'ga aylanganda yangi ID.
	MigrateToChatID int64 `json:"migrate_to_chat_id"`
}

// rawJSON — `result` maydonini keyinroq (kutilgan turga) unmarshal qilish uchun.
type rawJSON []byte

func (r *rawJSON) UnmarshalJSON(b []byte) error {
	*r = append((*r)[:0], b...)
	return nil
}

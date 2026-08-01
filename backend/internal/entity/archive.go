package entity

import "time"

// LessonArchive — o'tgan darsning YAGONA so'rovdagi to'liq surati
// (PRODUCT.md «Dars arxivi va Telegram saqlash», ish №20).
//
// # Nega bitta endpoint, uchta emas
//
// Arxiv sahifasi bir vaqtning o'zida video, chat va materiallarni ko'rsatadi.
// Klient buni uchta so'rov bilan yig'ganda uch xil yuklanish holati, uch xil
// xato yo'li va — eng yomoni — chat bilan videoning MOS KELMASLIK ehtimoli
// paydo bo'lardi (`offset_sec` boshqa so'rovda kelgan `started_at` ga tayanadi).
// Bitta javobda esa vaqt tayanchi ham, uning bo'yicha hisoblangan siljishlar
// ham bir xil o'qishdan keladi.
type LessonArchive struct {
	Lesson *Lesson `json:"lesson"`
	// Recording — yozuv topilmasa `null` (klient «yozuv yo'q» deb ko'rsatadi).
	Recording *ArchiveRecording `json:"recording"`
	// Chat — ESKIDAN YANGIGA (o'qish tartibi), o'chirilganlarsiz.
	Chat []ArchiveChatMessage `json:"chat"`
	// Materials — darsda ulashilgan fayllar (chat fayl xabarlaridan yig'iladi).
	Materials []ArchiveMaterial `json:"materials"`
}

// ArchiveRecording — arxiv sahifasi uchun yozuv (pleyer + holat).
//
// `entity.Recording` ning o'zi qaytarilmaydi: unda `egress_id` kabi ichki
// maydonlar bor va pleyerga kerak bo'lgan HAVOLA yo'q.
type ArchiveRecording struct {
	ID          string `json:"id"`
	Status      string `json:"status"`
	DurationSec int    `json:"duration_sec"`
	SizeBytes   int64  `json:"size_bytes"`
	// URL — presigned havola; faqat `ready` holatda to'ldiriladi.
	//
	// `expired` holatda ATAYLAB `null`, status esa o'zgarmaydi: fayl serverdan
	// o'chirilgan, lekin Telegram nusxasi bor va uni qaytarib olish oqimi
	// (PRODUCT.md, 30 kun + 1 kunlik kesh) aynan shu statusga tayanadi.
	// Bu yerda `expired` ni `null` recording'ga aylantirish klientdan
	// "qaytarib olish mumkin" ma'lumotini yashirardi.
	URL *string `json:"url"`
	// ExpiresAt — yozuv qachon serverdan o'chadi (retention). Hisoblanadigan
	// maydon — `entity.Recording.ExpiresAt` izohiga qara.
	ExpiresAt *time.Time `json:"expires_at"`
}

// ArchiveChatMessage — arxivdagi chat xabari + videoga bog'lash uchun siljish.
type ArchiveChatMessage struct {
	ID             string  `json:"id"`
	SenderIdentity string  `json:"sender_identity"`
	SenderName     string  `json:"sender_name"`
	Body           string  `json:"body"`
	ToIdentity     *string `json:"to_identity"`
	// File — ilova qilingan fayl (presigned havola bilan), yo'q bo'lsa `null`.
	File      *ChatFile `json:"file"`
	CreatedAt time.Time `json:"created_at"`
	// OffsetSec — xabar dars boshlangandan necha soniya keyin yozilgani.
	// Web pleyerda vaqt belgisini bosganda videoni shu nuqtaga sakratish uchun.
	// Hech qachon manfiy emas (dars boshlanishidan oldingi xabar → 0).
	OffsetSec int `json:"offset_sec"`
}

// ArchiveMaterial — darsda ulashilgan fayl (chatdagi fayl xabaridan).
type ArchiveMaterial struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Mime string `json:"mime"`
	// URL — vaqtinchalik presigned havola (imzolash yiqilsa bo'sh satr).
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

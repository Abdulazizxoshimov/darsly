package entity

import "time"

// Recording — dars yozuvi (LiveKit Egress → MinIO).
type Recording struct {
	ID          string     `json:"id"`
	LessonID    string     `json:"lesson_id"`
	EgressID    string     `json:"egress_id"`
	ObjectKey   string     `json:"-"` // MinIO ichidagi yo'l (ichki)
	Status      string     `json:"status"`
	DurationSec int        `json:"duration_sec"`
	SizeBytes   int64      `json:"size_bytes"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	// ExpiresAt — yozuv qachon avtomatik o'chiriladi (retention, PRODUCT.md №5).
	//
	// HISOBLANADIGAN maydon, DB'da saqlanmaydi: `ended_at + RECORDING_RETENTION_DAYS`.
	// Nega ustun emas — muddat sozlama (env) bilan boshqariladi va uni
	// o'zgartirganda MAVJUD yozuvlar ham yangi qoidaga bo'ysunishi kerak;
	// ustunga yozib qo'yilsa eski qatorlar eski muddat bilan qotib qolardi.
	// Faqat `ready` yozuvlarda to'ldiriladi (qolganlarida o'chiriladigan narsa yo'q).
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// ContentOffsetSec — fayl boshidan KESILGAN soniyalar (transkoddagi qora va
	// jim "o'lik" qism). Ya'ni faylning t=0 lahzasi haqiqiy vaqtda
	// `StartedAt + ContentOffsetSec` ga to'g'ri keladi.
	//
	// Nega saqlanadi: arxivda chat xabari bosilganda video o'sha lahzaga
	// sakraydi. Busiz butun chat kesilgan miqdorga siljib ketardi — 13 s kesish
	// = har bir xabar 13 s noto'g'ri joyga olib borardi. Klientga berilmaydi
	// (ichki hisob): tashqariga chiqadigan narsa allaqachon to'g'rilangan
	// `offset_sec`.
	ContentOffsetSec int `json:"-"`

	// ── Telegram arxivi (PRODUCT.md «Dars arxivi va Telegram saqlash») ───────
	//
	// `TelegramSentAt` — eng muhim maydon: server nusxasi FAQAT u to'lgan
	// bo'lsa o'chiriladi. Ya'ni video Telegramda tasdiqlanmaguncha diskda
	// qolaveradi (30 kun o'tgan bo'lsa ham).

	// TelegramFileID — Telegram fayl identifikatori. Yozuvni qaytarib olish
	// (restore) AYNAN shu orqali: `getFile` → yuklab olish → MinIO.
	TelegramFileID *string `json:"-"`
	// TelegramMessageID/TelegramChatID — arxiv guruhidagi xabar manzili
	// (mentorga «qayerda saqlangan» deb ko'rsatish va keyinchalik forward uchun).
	TelegramMessageID *int64     `json:"telegram_message_id,omitempty"`
	TelegramChatID    *int64     `json:"-"`
	TelegramSentAt    *time.Time `json:"telegram_sent_at,omitempty"`
	// TelegramError — oxirgi urinishdagi xato (mentorga ko'rsatiladi: nega
	// hali arxivda yo'q). Butun xato matni emas, qisqartirilgan.
	TelegramError *string `json:"telegram_error,omitempty"`
	// TelegramAttempts — nechta urinish bo'lgan (3 dan keyin to'xtaydi va
	// mentorga bildirishnoma ketadi).
	TelegramAttempts int `json:"telegram_attempts,omitempty"`
	// CachedUntil — Telegramdan TIKLANGAN nusxa qachongacha MinIO'da turadi.
	// nil → asl nusxa (kesh emas).
	CachedUntil *time.Time `json:"cached_until,omitempty"`
}

// InTelegram — yozuv Telegramda tasdiqlanganmi (server nusxasini o'chirish sharti).
func (r *Recording) InTelegram() bool { return r != nil && r.TelegramSentAt != nil }

// PlaybackZero — video faylining t=0 lahzasi HAQIQIY vaqtda qachon.
//
// ## Nega kerak
// Arxivda chat xabari bosilganda video o'sha lahzaga sakraydi. Sakrash
// `xabar_vaqti − nol_nuqta` bilan hisoblanadi va nol nuqta **fayl boshi**
// bo'lishi shart, dars boshi emas. Ular ikki sababdan farq qiladi:
//
//  1. Egress darsdan KEYINROQ boshlanadi (birinchi trek e'lon qilinganda);
//  2. Transkod fayl boshidagi qora va jim qismni kesadi ([ContentOffsetSec]).
//
// Ikkalasi ham hisobga olinmasa xabarlar o'nlab soniyaga siljib ketardi —
// ya'ni «sakrash» funksiyasi ishlagandek ko'rinib, aslida noto'g'ri joyga
// olib borardi.
//
// ok=false — yozuv yo'q yoki uning boshlanish vaqti noma'lum. Bunday holda
// chaqiruvchi darsning boshlanish vaqtiga tushadi (taxminiy, lekin yo'qdan
// yaxshi).
func (r *Recording) PlaybackZero() (time.Time, bool) {
	if r == nil || r.StartedAt.IsZero() {
		return time.Time{}, false
	}
	return r.StartedAt.Add(time.Duration(r.ContentOffsetSec) * time.Second), true
}

// ServerExpiry — yozuv qachon SERVERDAN o'chadi (klientga ko'rsatiladigan sana).
//
// ## Nega alohida sof funksiya
// Bu hisob ilgari IKKI joyda takrorlangan edi — `recording` va `arxiv`
// usecase'larida — va ular allaqachon ajralib ketgan: arxiv Telegram shartini
// ham, kesh nusxasini ham bilmasdi va tiklangan yozuvga O'TMISHDAGI sanani
// qaytarardi. Bunday takror har doim shu tarzda tugaydi, shuning uchun qoida
// endi bitta joyda va test ostida.
//
// Qoidalar:
//   - faqat `ready` yozuvda ma'no bor (boshqasida ko'rsatiladigan sana yo'q);
//   - `retention <= 0` → cheksiz saqlash, sana yo'q;
//   - Telegram YOQILGAN va yozuv u yerda TASDIQLANMAGAN bo'lsa sana yo'q:
//     bunday yozuv muddat bo'yicha o'chirilmaydi (kafolat), «X kundan keyin
//     o'chadi» deyish esa mentorni bekorga shoshirardi;
//   - Telegramdan TIKLANGAN nusxa `cached_until` bilan boshqariladi.
func ServerExpiry(r *Recording, retention time.Duration, telegramEnabled bool) *time.Time {
	if r == nil || retention <= 0 || r.Status != RecordingStatusReady {
		return nil
	}
	if telegramEnabled && !r.InTelegram() {
		return nil
	}
	if r.CachedUntil != nil {
		return r.CachedUntil
	}
	// Tugash vaqti noma'lum bo'lsa yaratilish vaqti — retention ishchisidagi
	// `COALESCE(ended_at, created_at)` bilan bir xil, aks holda UI va ishchi
	// turli sanalarni ko'rsatardi.
	base := r.CreatedAt
	if r.EndedAt != nil {
		base = *r.EndedAt
	}
	exp := base.Add(retention)
	return &exp
}

// Yozuv statuslari
const (
	RecordingStatusRecording  = "recording"  // egress faol
	RecordingStatusProcessing = "processing" // to'xtatildi, yuklanmoqda
	RecordingStatusReady      = "ready"      // MinIO'da tayyor
	RecordingStatusFailed     = "failed"
	// RecordingStatusExpired — yozuv BUTUNLAY yo'qolgan: MinIO'dagi fayl
	// o'chirilgan va Telegramda ham nusxasi yo'q.
	//
	// ⚠️ Telegram arxivi joriy etilgandan keyin bu holatga YO'L QO'YILMAYDI:
	// retention faqat `telegram_sent_at` to'lgan yozuvni o'chiradi va u
	// `archived` bo'ladi. `expired` faqat eski qatorlar uchun qoldi.
	RecordingStatusExpired = "expired"
	// RecordingStatusArchived — serverda YO'Q, Telegramda BOR.
	//
	// Mentor uchun bu «yo'qolgan» emas: `POST /recordings/:id/restore` bilan
	// qaytarib olinadi (30-60 s) va bir kun keshda turadi.
	RecordingStatusArchived = "archived"
	// RecordingStatusRestoring — hozir Telegramdan yuklab olinmoqda.
	//
	// Alohida holat kerak, chunki tiklash uzoq: endpoint darhol 202 qaytaradi
	// va klient shu statusni poll qiladi. `archived` da qoldirilsa klient
	// «bosdim, hech narsa bo'lmadi» deb yana bosaverardi.
	RecordingStatusRestoring = "restoring"
)

// Qayta kodlash (CRF) holatlari — `recordings.transcode_status`.
//
// Nega alohida holat: yozuv `ready` bo'lishi bilan ustoz uni yuklab ola oladi,
// qayta kodlash esa fonda va keyinroq bo'ladi. Ikkalasini bitta ustunga
// tiqish "tayyor, lekin hali tayyor emas" degan chalkash holat yasardi.
const (
	TranscodePending = "pending"
	TranscodeRunning = "running"
	TranscodeDone    = "done"
	TranscodeFailed  = "failed"
	TranscodeSkipped = "skipped"
)

// RecordingDownload — vaqtinchalik yuklab olish havolasi.
type RecordingDownload struct {
	URL         string `json:"url"`
	ExpiresInS  int    `json:"expires_in_s"`
	DurationSec int    `json:"duration_sec"`
	SizeBytes   int64  `json:"size_bytes"`
}

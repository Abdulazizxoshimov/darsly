package chat

import (
	"fmt"
	"html"
	"strings"
	"time"

	"github.com/zoom/darsly/internal/entity"
)

// Transkript formatlari (`?format=` qiymatlari).
const (
	FormatTXT  = "txt"
	FormatHTML = "html"
)

// Tashkent — transkriptdagi BARCHA vaqtlar shu mintaqada ko'rsatiladi.
//
// # Nega UTC emas va nega serverning mahalliy vaqti ham emas
//
// Xabarlar DB'da UTC saqlanadi (to'g'ri), lekin transkript ODAM o'qiydigan
// hujjat: ustoz "dars 14:30 da boshlangan edi" deb eslaydi, "09:30 UTC" deb
// emas. Serverning `time.Local` iga tayanish esa transkriptni JOYLASHUVGA
// bog'lardi — konteyner UTC'da ishlaydi va shu bilan har bir transkript
// besh soatga siljigan bo'lardi (jimgina, hech qanday xatosiz).
//
// Mintaqa qattiq belgilangan, chunki mahsulot O'zbekiston uchun (PRODUCT.md)
// va foydalanuvchida mintaqa maydoni yo'q. Ko'p mintaqali bo'lganda bu
// yagona o'zgaruvchi parametrga aylanadi.
var Tashkent = loadTashkent()

// loadTashkent — tzdata bo'lmagan muhitda ham to'g'ri ishlaydigan yuklash.
//
// `scratch`/`distroless` konteynerlarida IANA bazasi yo'q va
// `time.LoadLocation` xato qaytaradi. O'zbekiston yil bo'yi UTC+5 da (yozgi
// vaqt YO'Q), shuning uchun zaxira sifatida qat'iy siljish AYNAN bir xil
// natija beradi — ya'ni bu zaxira "taxminan to'g'ri" emas, to'liq to'g'ri.
func loadTashkent() *time.Location {
	if loc, err := time.LoadLocation("Asia/Tashkent"); err == nil {
		return loc
	}
	return time.FixedZone("+05", 5*60*60)
}

// TranscriptData — transkript render qilish uchun kerakli HAMMA narsa.
//
// Sof funksiyalar ([RenderTXT], [RenderHTML]) faqat shu strukturaga tayanadi:
// DB, HTTP, soat va konfiguratsiya yo'q. Shuning uchun formatni testlash
// "xabarlar ro'yxati → kutilgan satr" ga aylanadi va uni buzish uchun butun
// stekni ko'tarish shart emas.
type TranscriptData struct {
	// LessonTitle — sarlavhada ko'rsatiladi (foydalanuvchi matni: HTML'da escape).
	LessonTitle string
	// LessonDate — darsning boshlanish sanasi (sarlavhadagi «Sana»).
	LessonDate time.Time
	// MentorName — dars egasi.
	MentorName string
	// HostIdentity — kimning xabarlari «Siz» deb ko'rsatilishi.
	// Transkriptni ustoz yuklab oladi, ya'ni bu uning identity'si (= mentorID).
	HostIdentity string
	// Messages — ESKIDAN YANGIGA tartibda.
	Messages []*entity.ChatMessage
}

const (
	// transcriptSelf — host xabarlarining ko'rsatiladigan nomi.
	transcriptSelf = "Siz"
	// transcriptPrivate — shaxsiy xabar belgisi.
	transcriptPrivate = " (shaxsiy)"
	// transcriptFileMark — fayl ilovasi belgisi.
	transcriptFileMark = "📎 "
	// transcriptEmpty — chatda birorta xabar bo'lmaganda.
	transcriptEmpty = "(Chatda xabar bo'lmagan)"
	// transcriptRule — sarlavha bilan xabarlar orasidagi ajratgich.
	transcriptRule = "─────────────────────────────"
)

// isHost — xabar ustozning o'zinikimi.
func (d TranscriptData) isHost(m *entity.ChatMessage) bool {
	return m.SenderIdentity != "" && m.SenderIdentity == d.HostIdentity
}

// senderName — muallif nomi, shaxsiy belgisisiz.
//
// TXT va HTML IKKALASI ham shu yerdan oladi: ikki nusxa bo'lganda bir
// formatda «Siz», ikkinchisida ustozning to'liq ismi chiqib qolardi.
func (d TranscriptData) senderName(m *entity.ChatMessage) string {
	if d.isHost(m) {
		return transcriptSelf
	}
	if m.SenderName == "" {
		// Ism bo'lmasa identity: xabar egasiz qolgandan ko'ra texnik ID yaxshi.
		return m.SenderIdentity
	}
	return m.SenderName
}

// displayName — TXT uchun: nom + shaxsiy belgisi.
func (d TranscriptData) displayName(m *entity.ChatMessage) string {
	if m.ToIdentity != nil {
		return d.senderName(m) + transcriptPrivate
	}
	return d.senderName(m)
}

// messageText — xabar mazmuni: matn, fayl, yoki ikkalasi.
//
// Fayl xabarining o'zi bo'sh tanaga ega bo'lishi mumkin (izohsiz yuborilgan) —
// shunda faqat fayl nomi chiqadi, aks holda ikkalasi tire bilan qo'shiladi.
func messageText(m *entity.ChatMessage) string {
	body := strings.TrimSpace(m.Body)
	if m.File == nil || m.File.Name == "" {
		return body
	}
	file := transcriptFileMark + m.File.Name
	if body == "" {
		return file
	}
	return file + " — " + body
}

// RenderTXT — Zoom uslubidagi sodda matnli transkript. SOF funksiya.
func RenderTXT(d TranscriptData) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Dars: %s\n", d.LessonTitle)
	fmt.Fprintf(&b, "Sana: %s\n", d.LessonDate.In(Tashkent).Format("02.01.2006 15:04"))
	fmt.Fprintf(&b, "Mentor: %s\n", d.MentorName)
	b.WriteString(transcriptRule + "\n")

	if len(d.Messages) == 0 {
		b.WriteString(transcriptEmpty + "\n")
		return b.String()
	}
	for _, m := range d.Messages {
		fmt.Fprintf(&b, "%s  %s: %s\n",
			m.CreatedAt.In(Tashkent).Format("15:04:05"), d.displayName(m), txtBody(messageText(m)))
	}
	return b.String()
}

// txtIndent — vaqt ustuni kengligi ("15:04:05" + ikki bo'sh joy).
const txtIndent = "          "

// txtBody — ko'p qatorli xabarni davomiy qatorlarga surib yozadi.
//
// # Nega shunchaki qo'yib yubormaymiz
//
// Har bir haqiqiy satr `HH:MM:SS  Ism:` bilan boshlanadi. Xabar tanasidagi
// yangi qator esa ustunga TEKIS tushardi va o'quvchi tanasiga
// `14:59:00  Ustoz: darsdan chiqing` deb yozib, transkriptda ustoz aytgandek
// ko'rinadigan qator yasay olardi. Transkript — nizoda dalil bo'ladigan
// hujjat, shuning uchun davomiy qatorlar surilgan holda yoziladi va soxta
// satr darhol ko'rinadi.
func txtBody(s string) string {
	if !strings.ContainsAny(s, "\n\r") {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.ReplaceAll(s, "\n", "\n"+txtIndent)
}

// transcriptCSS — HTML transkriptning butun uslubi (Jonly brendi).
//
// INLINE va tashqi resurssiz: fayl ustozning kompyuterida, internetsiz,
// Telegram ilovasidan ochilishi mumkin. Bitta `<link>` yoki shrift havolasi
// ham o'sha holatda hujjatni buzardi.
const transcriptCSS = `
:root { color-scheme: dark; }
body {
  margin: 0; padding: 32px 16px;
  background: #05080c; color: #eef1f6;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Arial, sans-serif;
  font-size: 15px; line-height: 1.55;
}
.wrap { max-width: 760px; margin: 0 auto; }
.head {
  background: #0a1018; border: 1px solid #1b2431; border-radius: 14px;
  padding: 20px 22px; margin-bottom: 20px;
}
.brand {
  font-size: 12px; letter-spacing: .16em; text-transform: uppercase;
  color: #19d3a2; font-weight: 700; margin-bottom: 10px;
}
h1 { margin: 0 0 12px; font-size: 21px; line-height: 1.3; }
.meta { margin: 0; color: #97a3b3; font-size: 13px; }
.meta span { margin-right: 16px; white-space: nowrap; }
.msg {
  display: flex; gap: 12px; align-items: baseline;
  padding: 9px 12px; border-radius: 10px;
}
.msg + .msg { margin-top: 2px; }
.msg:nth-child(odd) { background: #070b12; }
.msg.self { background: rgba(25, 211, 162, .08); }
.time { color: #6b7688; font-size: 12px; font-variant-numeric: tabular-nums; flex: none; }
.who { color: #cdd5e0; font-weight: 600; flex: none; }
.msg.self .who { color: #19d3a2; }
.private {
  color: #97a3b3; font-weight: 500; font-size: 12px;
  border: 1px solid #2c3849; border-radius: 999px; padding: 1px 7px; margin-left: 6px;
}
.body { white-space: pre-wrap; word-break: break-word; min-width: 0; }
.file { color: #4ae3bc; }
.empty { color: #6b7688; text-align: center; padding: 32px 0; }
.foot { color: #6b7688; font-size: 12px; text-align: center; margin-top: 24px; }
@media print { body { background: #fff; color: #111; } .head, .msg { background: #fff; } }
`

// RenderHTML — bir faylli, offline ochiladigan HTML transkript. SOF funksiya.
//
// # XSS
// Ism ham, xabar matni ham, fayl nomi ham, dars sarlavhasi ham FOYDALANUVCHI
// KIRITGAN matn. Hujjat brauzerda ochiladi, ya'ni escape qilinmagan har qanday
// maydon o'quvchi yozgan `<script>` ni ustozning brauzerida ishga tushirardi
// (`file://` kelib chiqishida, ustozning boshqa fayllariga yaqin joyda).
// Shuning uchun BU FAYLDA hech qanday satr `html.EscapeString` siz chiqmaydi.
func RenderHTML(d TranscriptData) string {
	esc := html.EscapeString
	var b strings.Builder
	b.WriteString(`<!DOCTYPE html>` + "\n")
	b.WriteString(`<html lang="uz"><head><meta charset="utf-8">` + "\n")
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">` + "\n")
	fmt.Fprintf(&b, "<title>%s — chat transkripti</title>\n", esc(d.LessonTitle))
	fmt.Fprintf(&b, "<style>%s</style>\n</head>\n<body>\n<div class=\"wrap\">\n", transcriptCSS)

	b.WriteString(`<div class="head"><div class="brand">Jonly</div>` + "\n")
	fmt.Fprintf(&b, "<h1>%s</h1>\n", esc(d.LessonTitle))
	fmt.Fprintf(&b, "<p class=\"meta\"><span>%s</span><span>Mentor: %s</span></p>\n",
		esc(d.LessonDate.In(Tashkent).Format("02.01.2006 15:04")), esc(d.MentorName))
	b.WriteString("</div>\n")

	if len(d.Messages) == 0 {
		fmt.Fprintf(&b, "<p class=\"empty\">%s</p>\n", esc(transcriptEmpty))
	}
	for _, m := range d.Messages {
		cls := "msg"
		if d.isHost(m) {
			cls = "msg self"
		}
		fmt.Fprintf(&b, "<div class=\"%s\"><span class=\"time\">%s</span><span class=\"who\">%s</span>",
			cls, esc(m.CreatedAt.In(Tashkent).Format("15:04:05")), esc(d.senderName(m)))
		if m.ToIdentity != nil {
			b.WriteString(`<span class="private">shaxsiy</span>`)
		}
		b.WriteString(`<span class="body">`)
		if m.File != nil && m.File.Name != "" {
			// Havola YO'Q — presigned URL bir soatda o'ladi va faylga "o'lik"
			// havola qo'yish uni butunlay yo'q qilingandek ko'rsatardi.
			// Materiallar ilovada qoladi (PRODUCT.md), transkriptda esa nomi.
			fmt.Fprintf(&b, `<span class="file">%s%s</span>`, transcriptFileMark, esc(m.File.Name))
			if strings.TrimSpace(m.Body) != "" {
				fmt.Fprintf(&b, " — %s", esc(strings.TrimSpace(m.Body)))
			}
		} else {
			b.WriteString(esc(strings.TrimSpace(m.Body)))
		}
		b.WriteString("</span></div>\n")
	}

	fmt.Fprintf(&b, "<p class=\"foot\">Jonly — %d ta xabar</p>\n", len(d.Messages))
	b.WriteString("</div>\n</body>\n</html>\n")
	return b.String()
}

// safeFilename — dars sarlavhasidan XAVFSIZ fayl nomi yasaydi.
//
// Nom `Content-Disposition` sarlavhasiga tushadi. Sarlavha esa ustoz yozgan
// ixtiyoriy matn: qo'shtirnoq, nuqtali vergul yoki yangi qator undagi
// sarlavhani "sindirib", boshqa HTTP sarlavhasini qo'shish yo'lini ochardi.
// Shuning uchun oq ro'yxat: faqat ASCII harf/raqam va tire. Kirill/lotin
// diakritikasi tushib qoladi — bu ataylab: fayl nomining chiroyliligi
// sarlavha in'yeksiyasi xavfiga arzimaydi.
func safeFilename(title string, date time.Time, format string) string {
	const maxSlug = 40
	var b strings.Builder
	prevDash := true // boshida tire bo'lmasin
	for _, r := range strings.ToLower(title) {
		if b.Len() >= maxSlug {
			break
		}
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		slug = "dars"
	}
	ext := FormatTXT
	if format == FormatHTML {
		ext = FormatHTML
	}
	return fmt.Sprintf("%s-chat-%s.%s", slug, date.In(Tashkent).Format("2006-01-02"), ext)
}

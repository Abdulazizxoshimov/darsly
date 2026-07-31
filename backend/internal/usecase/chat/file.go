package chat

import (
	"bytes"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// MaxChatFileBytes — bitta ilovaning eng katta hajmi (20 MB).
//
// Nega 20: dars materiallari (slayd PDF, rasm, doc) shu chegaraga bemalol
// sig'adi, video esa YO'Q — va bu ataylab. Video ulashish uchun yozuv
// mexanizmi bor; chatga 200 MB'lik fayl qo'yish 4 CPU'lik VPS'da butun
// darsning tarmog'ini yeb qo'yadi.
const MaxChatFileBytes int64 = 20 << 20

// ChatFileURLTTL — presigned havolaning umri. Recording'dagi bilan bir xil
// tartibda: klient sahifani yangilaganda yangi havola oladi, eskisi esa
// nusxalanib tarqalsa ham tez o'ladi.
const ChatFileURLTTL = time.Hour

// Yuklash tezligi cheklovi: bitta identity daqiqasiga 5 ta fayl.
//
// Matnli chatdan (5 s da 5 ta) ancha qattiq va bu ataylab: bu yerda narx
// bayt — 20 MB × cheksiz urinish diskni to'ldiradi. Cheklov USTOZGA HAM
// tegadi (matnli chatdan farqli): u ham bo'lsa-da, diskni to'ldirish
// e'tiborsizlik bilan ham sodir bo'ladi (klient qayta-qayta urinsa).
const (
	uploadWindow = time.Minute
	uploadMax    = 5
)

// allowedChatFiles — RUXSAT ETILGAN turlar: kengaytma → (kanonik MIME, sniff qilinganda
// kutiladigan turlar).
//
// # Nega KENGAYTMA bo'yicha, klient yuborgan Content-Type bo'yicha emas
//
// Klientning `Content-Type` maydoni oddiy matn — uni har kim istalgan qiymatga
// qo'yadi. Shuning uchun ishonchli manba emas.
//
// # Nega bundan tashqari mazmun ham sniff qilinadi
//
// Kengaytmaning o'zi ham foydalanuvchi nazoratida: `virus.exe` ni `rasm.png`
// deb yuborish mumkin. `http.DetectContentType` esa haqiqiy baytlarga qaraydi.
// Ikkalasi mos kelmasa fayl rad etiladi.
//
// # Nega bu XSS'ni ham yopadi
//
// MinIO obyektni biz saqlashda bergan `Content-Type` bilan qaytaradi. Biz esa
// klientnikini emas, shu jadvaldagi KANONIK qiymatni beramiz — ya'ni HTML/JS
// yuklab, uni brauzerda `text/html` sifatida ochtirib bo'lmaydi.
var allowedChatFiles = map[string]struct {
	mime  string
	sniff []string
}{
	// Rasm — sniff aniq va ishonchli.
	".jpg":  {"image/jpeg", []string{"image/jpeg"}},
	".jpeg": {"image/jpeg", []string{"image/jpeg"}},
	".png":  {"image/png", []string{"image/png"}},
	".gif":  {"image/gif", []string{"image/gif"}},
	".webp": {"image/webp", []string{"image/webp"}},

	".pdf": {"application/pdf", []string{"application/pdf"}},

	// OOXML (docx/xlsx/pptx) — ichida ZIP, shuning uchun sniff "application/zip".
	".docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", []string{"application/zip"}},
	".xlsx": {"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", []string{"application/zip"}},
	".pptx": {"application/vnd.openxmlformats-officedocument.presentationml.presentation", []string{"application/zip"}},

	// Eski Office (OLE konteyner) — DetectContentType uni tanimaydi.
	".doc": {"application/msword", []string{"application/octet-stream"}},
	".xls": {"application/vnd.ms-excel", []string{"application/octet-stream"}},
	".ppt": {"application/vnd.ms-powerpoint", []string{"application/octet-stream"}},

	// Matn. `text/plain` sifatida qaytarilishi xavfsiz (HTML sifatida talqin
	// qilinmaydi), lekin sniff HTML topsa rad etamiz — quyida alohida tekshiruv.
	".txt": {"text/plain; charset=utf-8", []string{"text/plain"}},
	".csv": {"text/csv; charset=utf-8", []string{"text/plain"}},
}

// sniffLen — `http.DetectContentType` ishlatadigan bayt soni.
const sniffLen = 512

// validatedFile — tekshiruvdan o'tgan yuklama.
type validatedFile struct {
	name string // tozalangan fayl nomi (yo'lsiz)
	ext  string
	mime string // KANONIK MIME (MinIO'ga shu yoziladi)
	body io.Reader
	size int64
}

// validateFile — nom/hajm/tur tekshiruvi va mazmun sniff'i.
//
// Qaytgan `body` — MUSTAQIL o'quvchi: sniff uchun o'qib olingan bosh qism
// qaytadan ulanadi, ya'ni chaqiruvchi faylni to'liq oqim sifatida yuklaydi
// (butun 20 MB xotiraga olinmaydi).
func validateFile(name string, size int64, r io.Reader) (*validatedFile, error) {
	// Nomdagi yo'lni tashlaymiz: klient "../../etc/passwd" yubora oladi va u
	// obyekt kalitiga tushib ketardi. Kalit baribir UUID'dan yasaladi, lekin
	// nom tarixda ko'rinadi va klientda ham ishlatiladi.
	name = filepath.Base(strings.TrimSpace(strings.ReplaceAll(name, "\\", "/")))
	if name == "" || name == "." || name == "/" {
		return nil, apperr.BadRequest("file name is required")
	}
	if len(name) > 200 {
		name = name[len(name)-200:]
	}

	if size <= 0 {
		return nil, apperr.BadRequest("file is empty")
	}
	if size > MaxChatFileBytes {
		return nil, apperr.BadRequest("file is too large (max 20 MB)")
	}

	ext := strings.ToLower(filepath.Ext(name))
	spec, ok := allowedChatFiles[ext]
	if !ok {
		return nil, apperr.BadRequest("file type is not allowed (image, pdf, office document or text only)")
	}

	head := make([]byte, sniffLen)
	n, err := io.ReadFull(r, head)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, apperr.BadRequest("could not read file")
	}
	head = head[:n]

	detected := http.DetectContentType(head)
	if !sniffMatches(detected, spec.sniff) {
		return nil, apperr.BadRequest("file content does not match its extension")
	}

	return &validatedFile{
		name: name,
		ext:  ext,
		mime: spec.mime,
		body: io.MultiReader(bytes.NewReader(head), r),
		size: size,
	}, nil
}

// sniffMatches — aniqlangan tur kutilganlardan biriga mos keladimi.
// `DetectContentType` charset qo'shib qaytaradi ("text/plain; charset=utf-8"),
// shuning uchun prefiks bo'yicha solishtiriladi.
func sniffMatches(detected string, want []string) bool {
	base := detected
	if i := strings.IndexByte(base, ';'); i >= 0 {
		base = strings.TrimSpace(base[:i])
	}
	for _, w := range want {
		if base == w {
			return true
		}
	}
	return false
}

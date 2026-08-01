// Package telegram — Telegram Bot API klienti (dars yozuvlarini arxivlash).
//
// # Nega umuman kerak
//
// Mahsulot qarori (PRODUCT.md, 2026-08-01): server nusxasi 30 kun turadi,
// Telegramdagi nusxa esa ABADIY. Ya'ni Telegram bu yerda "bonus tarqatish
// kanali" emas — u asosiy uzoq muddatli saqlagich.
//
// # Local Bot API Server
//
// Oddiy bot `sendVideo` bilan 50 MB gacha yubora oladi; 1 soatlik dars esa
// 75-250 MB, 2.5 soatliki ~1 GB gacha. Shuning uchun serverda Telegramning
// O'Z dasturi — Local Bot API Server (`--local`) ishlaydi va chegara 2 GB
// bo'ladi. `TELEGRAM_API_URL` shu serverga ko'rsatadi; bo'sh bo'lsa rasmiy
// api.telegram.org ishlatiladi va 50 MB chegara amal qiladi.
//
// # Sozlanmagan bo'lsa — JIMGINA O'CHIQ
//
// `TELEGRAM_BOT_TOKEN` bo'sh bo'lsa [New] `nop` klient qaytaradi: `Enabled()`
// false, barcha metodlar tushunarli xato beradi va HECH BIR ishchi ishga
// tushmaydi. Yozib olish esa avvalgidek ishlayveradi. Bu Sentry bilan bir xil
// naqsh — integratsiya yo'qligi nosozlik emas, sozlama.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// DefaultAPIURL — rasmiy Bot API (50 MB yuklash chegarasi bilan).
const DefaultAPIURL = "https://api.telegram.org"

// CloudUploadLimit — rasmiy api.telegram.org dagi `sendVideo` chegarasi.
const CloudUploadLimit int64 = 50 * 1024 * 1024

// Config — telegram klienti sozlamalari (barchasi `.env` dan).
type Config struct {
	// BotToken — @BotFather bergan token. BO'SH → integratsiya butunlay o'chiq.
	BotToken string
	// APIURL — Local Bot API Server manzili (`http://telegram-bot-api:8081`).
	// Bo'sh → rasmiy api.telegram.org.
	APIURL string
	// ArchiveChatID — video HAR DOIM avtomatik tushadigan guruh. 0 → avtomatik
	// arxivlash o'chiq (mentor qo'lda tanlagan guruhga yuborish ishlayveradi).
	ArchiveChatID int64
	// MaxUploadBytes — yuklash chegarasi. 0 → rejimga qarab (local 2000 MB,
	// cloud 50 MB).
	MaxUploadBytes int64
	// FileRoot — Local Bot API Server fayllarni saqlaydigan katalog, backend
	// konteynerida ULASHILGAN volume sifatida ko'rinadi. Bo'sh bo'lsa fayl
	// HTTP orqali yuklab olinadi (sekinroq, lekin ishlaydi).
	FileRoot string
}

// Client — usecase/worker qatlami ko'radigan sirt (DIP: ular `telegram`
// paketining ichki tuzilishini bilmaydi, faqat shu interfeysni).
type Client interface {
	// Enabled — integratsiya sozlanganmi. false bo'lsa hech narsa chaqirilmaydi.
	Enabled() bool
	// Local — Local Bot API Server rejimidami (2 GB chegara).
	Local() bool
	// ArchiveChatID — avtomatik arxiv guruhi (0 → yo'q).
	ArchiveChatID() int64
	// MaxUploadBytes — bitta fayl uchun chegara.
	MaxUploadBytes() int64
	// BotUsername — deep-link (`t.me/<bot>?start=<kod>`) yasash uchun.
	// [Client.Init] chaqirilgunicha bo'sh bo'lishi mumkin.
	BotUsername() string

	// Init — `getMe` bilan botni tekshiradi va username'ni keshlaydi.
	// Ishga tushishda bir marta; xato — integratsiya sozlanmagan degani.
	Init(ctx context.Context) error

	SendMessage(ctx context.Context, chatID int64, text string, kb *InlineKeyboard) (*Message, error)
	EditMessageText(ctx context.Context, chatID, messageID int64, text string, kb *InlineKeyboard) error
	AnswerCallbackQuery(ctx context.Context, id, text string) error

	// SendVideo — yozuvni yuboradi. `path` — lokal fayl (ishchi uni MinIO'dan
	// vaqtinchalik katalogga tushirgan bo'ladi).
	SendVideo(ctx context.Context, chatID int64, path, caption string, durationSec int) (*Message, error)
	// SendVideoByFileID — ALLAQACHON Telegramda bo'lgan videoni boshqa chatga
	// yuboradi.
	//
	// ⭐ Fayl QAYTA YUKLANMAYDI. Telegram bir marta yuklangan faylni `file_id`
	// bo'yicha istalgan chatga bir zumda ko'chiradi. Arxivdan o'quvchilar
	// guruhiga ulashish shu sabab bir necha millisekund, 1 GB qayta yuklash
	// esa o'nlab daqiqa bo'lardi.
	SendVideoByFileID(ctx context.Context, chatID int64, fileID, caption string) (*Message, error)
	// SendDocument — chat transkripti (TXT/HTML) uchun.
	SendDocument(ctx context.Context, chatID int64, filename string, body io.Reader, size int64, caption string) (*Message, error)

	// GetFile — `file_id` bo'yicha fayl ma'lumoti (tiklash uchun).
	GetFile(ctx context.Context, fileID string) (*File, error)
	// DownloadFile — faylni `dst` ga yozadi. Local rejimda ulashilgan
	// volume'dan NUSXALANADI (tarmoqsiz), aks holda HTTP orqali yuklanadi.
	DownloadFile(ctx context.Context, f *File, dst string) error

	// GetUpdates — long-polling. `offset` — oxirgi qayta ishlangan
	// `update_id + 1`; `timeoutS` — server javobni shuncha soniya ushlab turadi.
	GetUpdates(ctx context.Context, offset int64, timeoutS int) ([]Update, error)
}

// RetryAfterError — Telegram 429 bilan javob berdi.
//
// Alohida tur kerak: chaqiruvchi (yuklash ishchisi) buni "vaqtinchalik" deb
// biladi va urinishlar hisobini OSHIRMAYDI — aks holda band soatda uch
// urinish bir necha sekundda sarflanib, yozuv "muvaffaqiyatsiz" deb
// belgilanardi.
type RetryAfterError struct {
	After time.Duration
}

func (e *RetryAfterError) Error() string {
	return fmt.Sprintf("telegram: rate limited, retry after %s", e.After)
}

// IsRetryAfter — xato 429 mi (va qancha kutish kerak).
func IsRetryAfter(err error) (time.Duration, bool) {
	var e *RetryAfterError
	if errors.As(err, &e) {
		return e.After, true
	}
	return 0, false
}

// APIError — Bot API mazmunli xato qaytardi (`ok: false`).
type APIError struct {
	Code        int
	Description string
	Method      string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("telegram %s: %d %s", e.Method, e.Code, e.Description)
}

// Permanent — xato qayta urinishga arzimaydimi.
//
// 400/403 odatda konfiguratsiya xatosi (bot guruhdan chiqarilgan, chat ID
// noto'g'ri, fayl juda katta). Bularni uch marta qaytarish faqat log to'ldiradi
// va mentorga xabar berishni kechiktiradi.
func (e *APIError) Permanent() bool {
	return e.Code == 400 || e.Code == 401 || e.Code == 403 || e.Code == 404
}

// IsPermanent — xato tuzatilmaguncha qayta urinish foydasizmi.
func IsPermanent(err error) bool {
	var e *APIError
	if errors.As(err, &e) {
		return e.Permanent()
	}
	return false
}

type client struct {
	token   string
	apiURL  string
	base    string // <apiURL>/bot<token>
	fileURL string // <apiURL>/file/bot<token>
	local   bool

	archiveChat int64
	maxUpload   int64
	fileRoot    string

	username string

	// http — UZOQ so'rovlar uchun (1.9 GB yuklash soatlab ketishi mumkin).
	// Global `Timeout` ATAYLAB qo'yilmagan: muddat har chaqiruvda `context`
	// bilan beriladi. Aks holda katta yuklash o'rtasida uzilib, butun
	// integratsiya "ishlamaydi" bo'lib qolardi.
	http *http.Client
}

// New — klient yasaydi. Token bo'sh bo'lsa `nop` (integratsiya o'chiq).
func New(cfg Config) Client {
	if strings.TrimSpace(cfg.BotToken) == "" {
		return nopClient{}
	}
	apiURL := strings.TrimRight(strings.TrimSpace(cfg.APIURL), "/")
	local := apiURL != "" && apiURL != DefaultAPIURL
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	max := cfg.MaxUploadBytes
	if max <= 0 {
		max = CloudUploadLimit
		if local {
			max = 1900 * 1024 * 1024
		}
	}
	// Local server 2000 MB dan ko'pini qabul qilmaydi — chegarani ustidan
	// so'ramaymiz (aks holda soatlab yuklab, oxirida 413 olardik).
	if local && max > 2000*1024*1024 {
		max = 2000 * 1024 * 1024
	}
	// Cloud rejimda chegarani oshirib bo'lmaydi: bu Telegram tomonidagi qat'iy
	// limit, sozlama bilan aylanib o'tilmaydi.
	if !local && max > CloudUploadLimit {
		max = CloudUploadLimit
	}
	return &client{
		token:       cfg.BotToken,
		apiURL:      apiURL,
		base:        apiURL + "/bot" + cfg.BotToken,
		fileURL:     apiURL + "/file/bot" + cfg.BotToken,
		local:       local,
		archiveChat: cfg.ArchiveChatID,
		maxUpload:   max,
		fileRoot:    strings.TrimRight(cfg.FileRoot, "/"),
		http:        &http.Client{},
	}
}

func (c *client) Enabled() bool         { return true }
func (c *client) Local() bool           { return c.local }
func (c *client) ArchiveChatID() int64  { return c.archiveChat }
func (c *client) MaxUploadBytes() int64 { return c.maxUpload }
func (c *client) BotUsername() string   { return c.username }

func (c *client) Init(ctx context.Context) error {
	var me User
	if err := c.call(ctx, "getMe", nil, &me); err != nil {
		return err
	}
	c.username = me.Username
	return nil
}

// ─── Tranzport ───────────────────────────────────────────────────────────────

// call — JSON so'rov (fayl yuklashdan tashqari hamma metod).
func (c *client) call(ctx context.Context, method string, payload any, out any) error {
	var body io.Reader
	contentType := ""
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("telegram %s: marshal: %w", method, err)
		}
		body = bytes.NewReader(b)
		contentType = "application/json"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/"+method, body)
	if err != nil {
		return fmt.Errorf("telegram %s: request: %w", method, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.do(req, method, out)
}

// do — javobni o'qiydi va Bot API xato qobig'ini tarjima qiladi.
//
// Qayta urinish BU YERDA EMAS: 429 va tranzient xatolar chaqiruvchiga
// qaytariladi. Sabab — bizda urinish siyosati ikki xil: yuklash ishchisi
// uchun 1m/5m/15m (DB'da saqlanadi, deploy'dan omon qoladi), long-polling
// uchun esa oddiy qisqa kutish. Tranzportga bitta "universal" retry qo'yilsa
// ikkalasi ham noto'g'ri bo'lardi.
func (c *client) do(req *http.Request, method string, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram %s: %w", method, err)
	}
	defer resp.Body.Close()

	// Javob kichik (fayl mazmuni emas) — lekin buzuq server cheksiz oqim
	// yuborishi mumkin, shuning uchun cheklangan o'qish.
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("telegram %s: read: %w", method, err)
	}

	var ar apiResponse
	if err := json.Unmarshal(raw, &ar); err != nil {
		return fmt.Errorf("telegram %s: bad response (http %d): %w", method, resp.StatusCode, err)
	}
	if !ar.OK {
		if ar.Parameters != nil && ar.Parameters.RetryAfter > 0 {
			return &RetryAfterError{After: time.Duration(ar.Parameters.RetryAfter) * time.Second}
		}
		code := ar.ErrorCode
		if code == 0 {
			code = resp.StatusCode
		}
		return &APIError{Code: code, Description: ar.Description, Method: method}
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(ar.RawResult, out); err != nil {
		return fmt.Errorf("telegram %s: decode result: %w", method, err)
	}
	return nil
}

// ─── Xabarlar ────────────────────────────────────────────────────────────────

func (c *client) SendMessage(ctx context.Context, chatID int64, text string, kb *InlineKeyboard) (*Message, error) {
	payload := map[string]any{
		"chat_id": chatID,
		"text":    text,
		// HTML emas, oddiy matn: dars sarlavhasi foydalanuvchi kiritgan satr va
		// undagi `<` belgisi butun xabarni yuborilmaydigan qilib qo'yardi
		// (Telegram parse xatosi). Formatlashdan voz kechish arzonroq.
		"disable_web_page_preview": true,
	}
	if kb != nil {
		payload["reply_markup"] = kb
	}
	var m Message
	if err := c.call(ctx, "sendMessage", payload, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *client) EditMessageText(ctx context.Context, chatID, messageID int64, text string, kb *InlineKeyboard) error {
	payload := map[string]any{
		"chat_id":    chatID,
		"message_id": messageID,
		"text":       text,
	}
	if kb != nil {
		payload["reply_markup"] = kb
	}
	return c.call(ctx, "editMessageText", payload, nil)
}

func (c *client) AnswerCallbackQuery(ctx context.Context, id, text string) error {
	return c.call(ctx, "answerCallbackQuery", map[string]any{
		"callback_query_id": id,
		"text":              text,
	}, nil)
}

func (c *client) GetUpdates(ctx context.Context, offset int64, timeoutS int) ([]Update, error) {
	payload := map[string]any{
		"offset":  offset,
		"timeout": timeoutS,
		// Faqat bizga kerak bo'lgan turlar. `my_chat_member` DEFAULT
		// yuborilmaydi — uni ataylab so'rash SHART, aks holda bot qaysi
		// guruhlarga qo'shilganini hech qachon bilmaydi (jimgina bo'sh ro'yxat).
		"allowed_updates": []string{"message", "callback_query", "my_chat_member"},
	}
	var ups []Update
	if err := c.call(ctx, "getUpdates", payload, &ups); err != nil {
		return nil, err
	}
	return ups, nil
}

// ─── Fayl yuklash ────────────────────────────────────────────────────────────

func (c *client) SendVideo(ctx context.Context, chatID int64, path, caption string, durationSec int) (*Message, error) {
	st, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("telegram sendVideo: stat: %w", err)
	}
	if st.Size() > c.maxUpload {
		// Qayta urinish foydasiz — hajm o'zgarmaydi. `APIError` 400 bilan
		// qaytaramiz, ishchi uni "doimiy" deb bilib darhol mentorga xabar beradi.
		return nil, &APIError{
			Code:   400,
			Method: "sendVideo",
			Description: fmt.Sprintf("fayl juda katta: %.0f MB (chegara %.0f MB)",
				float64(st.Size())/(1024*1024), float64(c.maxUpload)/(1024*1024)),
		}
	}
	fields := map[string]string{
		"chat_id": strconv.FormatInt(chatID, 10),
		"caption": caption,
		// supports_streaming — Telegram klientida yuklab olmasdan ko'rish
		// mumkin bo'lsin. Dars yozuvi uchun bu asosiy foydalanish stsenariysi.
		"supports_streaming": "true",
	}
	if durationSec > 0 {
		fields["duration"] = strconv.Itoa(durationSec)
	}
	var m Message
	if err := c.upload(ctx, "sendVideo", "video", filepath.Base(path), path, nil, st.Size(), fields, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// SendVideoByFileID — `file_id` bilan qayta yuborish (yuklashsiz).
//
// `multipart` EMAS, oddiy JSON: fayl mazmuni umuman uzatilmaydi, faqat ID.
func (c *client) SendVideoByFileID(ctx context.Context, chatID int64, fileID, caption string) (*Message, error) {
	var m Message
	err := c.call(ctx, "sendVideo", map[string]any{
		"chat_id":            chatID,
		"video":              fileID,
		"caption":            caption,
		"supports_streaming": true,
	}, &m)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *client) SendDocument(ctx context.Context, chatID int64, filename string, body io.Reader, size int64, caption string) (*Message, error) {
	if size > c.maxUpload {
		return nil, &APIError{Code: 400, Method: "sendDocument", Description: "fayl juda katta"}
	}
	fields := map[string]string{
		"chat_id": strconv.FormatInt(chatID, 10),
		"caption": caption,
	}
	var m Message
	if err := c.upload(ctx, "sendDocument", "document", filename, "", body, size, fields, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// upload — multipart yuklash OQIM bilan.
//
// ⚠️ `io.Pipe` ATAYLAB: oddiy `multipart.Writer` + `bytes.Buffer` yondashuvi
// butun faylni XOTIRAGA oladi. 1.9 GB yozuv bilan bu 4 GB RAM li VPS'da
// backend'ni OOM bilan o'ldiradi — ya'ni arxivlash dars yozib olishni ham
// buzardi. Pipe bilan xotira sarfi bufer o'lchamida qoladi.
//
// `srcPath` berilsa fayl diskdan o'qiladi, aks holda `src` oqimi ishlatiladi.
func (c *client) upload(
	ctx context.Context,
	method, field, filename, srcPath string,
	src io.Reader,
	_ int64,
	fields map[string]string,
	out any,
) error {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		// Yozish xatosi o'quvchi tomonga uzatiladi — `http.Client` so'rovni
		// uzadi va biz aniq sababni ko'ramiz (jimgina yarim fayl ketmaydi).
		err := func() error {
			for k, v := range fields {
				if err := mw.WriteField(k, v); err != nil {
					return err
				}
			}
			part, err := mw.CreateFormFile(field, filename)
			if err != nil {
				return err
			}
			r := src
			if srcPath != "" {
				f, err := os.Open(srcPath)
				if err != nil {
					return err
				}
				defer f.Close()
				r = f
			}
			if _, err := io.Copy(part, r); err != nil {
				return err
			}
			return mw.Close()
		}()
		_ = pw.CloseWithError(err)
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/"+method, pr)
	if err != nil {
		_ = pr.CloseWithError(err)
		return fmt.Errorf("telegram %s: request: %w", method, err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	return c.do(req, method, out)
}

// ─── Fayl yuklab olish (tiklash) ─────────────────────────────────────────────

func (c *client) GetFile(ctx context.Context, fileID string) (*File, error) {
	var f File
	if err := c.call(ctx, "getFile", map[string]any{"file_id": fileID}, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func (c *client) DownloadFile(ctx context.Context, f *File, dst string) error {
	if f == nil || f.FilePath == "" {
		return fmt.Errorf("telegram: bo'sh file_path")
	}
	// 1-yo'l: Local Bot API Server bilan ULASHILGAN volume. Fayl allaqachon
	// bizning diskda — tarmoqqa chiqmasdan nusxalaymiz. 1 GB video uchun bu
	// bir necha soniya vs. bir necha daqiqa farq qiladi.
	if p := c.localPath(f.FilePath); p != "" {
		return copyFile(p, dst)
	}
	// 2-yo'l: HTTP. Rasmiy API'da `file_path` nisbiy; local rejimda ham
	// (volume ulanmagan bo'lsa) server shu yo'l bilan berishga harakat qiladi.
	rel := strings.TrimPrefix(f.FilePath, "/")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.fileURL+"/"+url.PathEscape(rel), nil)
	if err != nil {
		return fmt.Errorf("telegram download: request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("telegram download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return &APIError{Code: resp.StatusCode, Method: "download", Description: "fayl olinmadi"}
	}
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("telegram download: create: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("telegram download: copy: %w", err)
	}
	return out.Sync()
}

// localPath — `file_path` bizning diskda o'qiladigan faylga to'g'ri kelsa
// shu yo'lni qaytaradi, aks holda bo'sh satr.
//
// Ikki holat qo'llab-quvvatlanadi:
//   - absolyut yo'l (Local Bot API Server `--local` rejimida shunday beradi)
//     va u BIZNING konteynerda ham mavjud (ulashilgan volume);
//   - nisbiy yo'l + `TELEGRAM_FILE_ROOT` sozlangan.
func (c *client) localPath(p string) string {
	if filepath.IsAbs(p) {
		if _, err := os.Stat(p); err == nil {
			return p
		}
		// Volume boshqa nuqtaga ulangan bo'lishi mumkin (masalan server
		// `/var/lib/telegram-bot-api`, bizda `/tgfiles`). Root sozlangan bo'lsa
		// yo'lning oxirgi qismlarini unga ulab ko'ramiz.
		if c.fileRoot != "" {
			if i := strings.Index(p, "/telegram-bot-api/"); i >= 0 {
				cand := filepath.Join(c.fileRoot, p[i+len("/telegram-bot-api/"):])
				if _, err := os.Stat(cand); err == nil {
					return cand
				}
			}
		}
		return ""
	}
	if c.fileRoot == "" {
		return ""
	}
	cand := filepath.Join(c.fileRoot, p)
	if _, err := os.Stat(cand); err == nil {
		return cand
	}
	return ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("telegram: local file: %w", err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("telegram: create: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("telegram: copy: %w", err)
	}
	return out.Sync()
}

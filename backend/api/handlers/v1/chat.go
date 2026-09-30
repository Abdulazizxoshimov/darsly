package v1

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/usecase/chat"
)

// parseBefore — kursor-paginatsiya parametri (RFC3339). Bo'sh bo'lsa nil.
func parseBefore(c *gin.Context) (*time.Time, bool) {
	raw := c.Query("before")
	if raw == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		hs.BadRequest(c, "before must be RFC3339 timestamp")
		return nil, false
	}
	return &t, true
}

// SendChat godoc
// @Summary      Chat xabari yuborish (host) — saqlanadi + LiveKit orqali tarqaladi
// @Tags         chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string             true  "Lesson ID"
// @Param        body  body  entity.SendChatReq  true  "Xabar (to — shaxsiy uchun)"
// @Success      201  {object}  object{data=entity.ChatMessage}
// @Router       /api/v1/lessons/{id}/chat [post]
func SendChat(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		var req entity.SendChatReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		msg, err := h.Chat.Send(c.Request.Context(), mentorID, c.Param("id"), req.Body, req.To)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, msg)
	}
}

// ChatHistory godoc
// @Summary      Dars chat tarixi (host) — eng yangidan eskiga, kursor-paginatsiya
// @Tags         chat
// @Produce      json
// @Security     BearerAuth
// @Param        id      path   string  true   "Lesson ID"
// @Param        before  query  string  false  "Kursor: shu vaqtdan (RFC3339) eski xabarlar"
// @Param        limit   query  int     false  "Sahifa hajmi (max 50)"
// @Success      200  {object}  object{data=[]entity.ChatMessage}
// @Router       /api/v1/lessons/{id}/chat [get]
func ChatHistory(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		before, ok := parseBefore(c)
		if !ok {
			return
		}
		limit, _ := strconv.Atoi(c.Query("limit"))

		items, err := h.Chat.History(c.Request.Context(), mentorID, c.Param("id"), before, limit)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, items)
	}
}

// ChatTranscript godoc
// @Summary      Chat transkriptini yuklab olish (host) — TXT yoki HTML
// @Description  Zoom uslubidagi chat fayli: vaqt belgilari Toshkent vaqtida,
// @Description  shaxsiy xabarlar va fayl ilovalari belgilangan. HTML varianti
// @Description  bir faylli va offline ochiladi (tashqi resurs yo'q).
// @Tags         chat
// @Produce      plain
// @Security     BearerAuth
// @Param        id      path   string  true   "Lesson ID"
// @Param        format  query  string  false  "txt (default) yoki html"
// @Success      200  {string}  string  "fayl (Content-Disposition: attachment)"
// @Router       /api/v1/lessons/{id}/chat/transcript [get]
func ChatTranscript(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		t, err := h.Chat.Transcript(c.Request.Context(), mentorID, c.Param("id"), c.Query("format"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		// Fayl nomi usecase'da tozalangan (faqat ASCII harf/raqam/tire) —
		// `entity.ChatTranscript` izohida sabab. Shuning uchun bu yerda uni
		// qo'shtirnoq ichiga qo'yish xavfsiz.
		c.Header("Content-Disposition", `attachment; filename="`+t.Filename+`"`)
		// Transkript o'zgarishi mumkin (yangi xabar, moderatsiya) — keshlanmasin.
		c.Header("Cache-Control", "no-store")
		c.Data(http.StatusOK, t.ContentType, t.Body)
	}
}

// SendRoomChat godoc
// @Summary  Xonadan chat xabari (ochiq — LiveKit room-token bilan)
// @Tags     chat
// @Accept   json
// @Produce  json
// @Param    lessonID  path  string                  true  "Lesson ID"
// @Param    body      body  entity.SendRoomChatReq  true  "Token + xabar (to — shaxsiy uchun)"
// @Success  201  {object}  object{data=entity.ChatMessage}
// @Router   /api/v1/rooms/{lessonID}/chat [post]
func SendRoomChat(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.SendRoomChatReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		lessonID := c.Param("lessonID")
		identity, name, ok := roomTokenIdentity(c, h, req.Token, lessonID)
		if !ok {
			return
		}
		msg, err := h.Chat.SendFromRoom(c.Request.Context(), lessonID, identity, name, req.Body, req.To)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, msg)
	}
}

// DeleteChatMessage godoc
// @Summary      Chat xabarini o'chirish (moderatsiya — faqat dars egasi)
// @Description  Yumshoq o'chirish: xabar tarixdan butunlay yo'qoladi va jonli
// @Description  xonadagi klientlarga `chat_deleted` hodisasi yuboriladi.
// @Tags         chat
// @Security     BearerAuth
// @Param        id         path  string  true  "Lesson ID"
// @Param        messageID  path  string  true  "Chat message ID"
// @Success      204
// @Router       /api/v1/lessons/{id}/chat/{messageID} [delete]
func DeleteChatMessage(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		if err := h.Chat.Delete(c.Request.Context(), mentorID, c.Param("id"), c.Param("messageID")); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// Yuklash hamroh maydonlarining chegaralari (entity.SendChatReq bilan mos).
const (
	maxChatBodyRunes = 2000
	maxChatToLen     = 128
)

// chatUpload — multipart so'rovdan faylni va hamroh maydonlarni oladi.
//
// Bitta joyda: host va xona yo'llari AYNAN bir xil shaklni kutadi, ikki nusxa
// esa vaqt o'tib bir-biridan uzilib qolardi (masalan biri `caption`, ikkinchisi
// `body` deb nomlanib).
//
// Fayl OQIM sifatida uzatiladi (`multipart.File` — `io.Reader`), ya'ni 20 MB
// xotiraga to'liq olinmaydi; gin `MaxMultipartMemory` dan oshgan qismni diskka
// yozadi va `Close` uni tozalaydi.
func chatUpload(c *gin.Context) (chat.FileUpload, string, string, bool) {
	fh, err := c.FormFile("file")
	if err != nil {
		// Hajm chegarasidan oshsa `http.MaxBytesReader` shu yerda xato beradi —
		// klient uchun bu ham 400 (so'rov noto'g'ri), 500 emas.
		hs.BadRequest(c, "file is required (multipart field \"file\", max 20 MB)")
		return chat.FileUpload{}, "", "", false
	}
	f, err := fh.Open()
	if err != nil {
		hs.BadRequest(c, "could not read uploaded file")
		return chat.FileUpload{}, "", "", false
	}
	// Hamroh maydonlar validatsiyasi (JSON yo'lidagi `validate` teglari bilan bir xil chegara):
	// multipart'da ular teg orqali tekshirilmaydi, 20 MB'lik caption bo'lishi mumkin edi.
	body, to := c.PostForm("body"), c.PostForm("to")
	if utf8.RuneCountInString(body) > maxChatBodyRunes {
		hs.BadRequest(c, "body is too long (max 2000 characters)")
		return chat.FileUpload{}, "", "", false
	}
	if len(to) > maxChatToLen {
		hs.BadRequest(c, "to is too long (max 128)")
		return chat.FileUpload{}, "", "", false
	}
	// Handler `Close` ni o'z zimmasiga oladi: usecase oqimni faqat O'QIYDI va
	// uning hayot davri HTTP so'rovga tegishli (qatlam chegarasi toza qoladi).
	c.Set(ctxUploadCloser, f)
	return chat.FileUpload{Name: fh.Filename, Size: fh.Size, Reader: f}, body, to, true
}

// ctxUploadCloser — ochilgan multipart faylni so'rov oxirida yopish uchun kalit.
const ctxUploadCloser = "chat_upload_file"

// ChatUploadPathSuffix / ChatUploadBodyLimit — router'dagi tana-hajmi
// middleware'i uchun (`api/router.go`). Shu yerda turadi, chunki chegara
// mahsulot qoidasidan (`chat.MaxChatFileBytes`) kelib chiqadi va router'da
// mustaqil raqam sifatida yozilsa ikkisi bir-biridan uzilib qolardi.
const (
	ChatUploadPathSuffix = "/chat/upload"
	// +1 MB — multipart chegaralari va qo'shimcha maydonlar uchun zaxira.
	ChatUploadBodyLimit = chat.MaxChatFileBytes + (1 << 20)
)

// closeUpload — [chatUpload] ochgan oqimni yopadi (defer bilan chaqiriladi).
func closeUpload(c *gin.Context) {
	if v, ok := c.Get(ctxUploadCloser); ok {
		if f, ok := v.(multipart.File); ok {
			_ = f.Close()
		}
	}
}

// UploadChatFile godoc
// @Summary      Chatda fayl ulashish (host) — MinIO + presigned havola
// @Tags         chat
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        id    path      string  true   "Lesson ID"
// @Param        file  formData  file    true   "Fayl (max 20 MB; rasm/pdf/ofis/matn)"
// @Param        body  formData  string  false  "Izoh"
// @Param        to    formData  string  false  "Shaxsiy uchun qabul qiluvchi identity"
// @Success      201  {object}  object{data=entity.ChatMessage}
// @Router       /api/v1/lessons/{id}/chat/upload [post]
func UploadChatFile(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer closeUpload(c)
		mentorID := c.GetString(middleware.CtxUserID)
		f, body, to, ok := chatUpload(c)
		if !ok {
			return
		}
		msg, err := h.Chat.Upload(c.Request.Context(), mentorID, c.Param("id"), f, body, to)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, msg)
	}
}

// UploadRoomChatFile godoc
// @Summary  Xonadan fayl ulashish (ochiq — LiveKit room-token bilan)
// @Tags     chat
// @Accept   multipart/form-data
// @Produce  json
// @Param    lessonID  path      string  true   "Lesson ID"
// @Param    token     query     string  false  "LiveKit room token (afzal — tana o'qilmasdan tekshiriladi)"
// @Param    token     formData  string  false  "LiveKit room token (fallback)"
// @Param    file      formData  file    true   "Fayl (max 20 MB)"
// @Param    body      formData  string  false  "Izoh"
// @Param    to        formData  string  false  "Shaxsiy uchun qabul qiluvchi identity"
// @Success  201  {object}  object{data=entity.ChatMessage}
// @Router   /api/v1/rooms/{lessonID}/chat/upload [post]
func UploadRoomChatFile(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer closeUpload(c)
		lessonID := c.Param("lessonID")
		// TOKEN AVVAL, TANA KEYIN.
		//
		// `?token=` afzal ko'riladi, chunki uni o'qish TANANI PARSE QILMAYDI:
		// `c.PostForm` esa butun multipart'ni o'qib bo'ladi (10 MB gacha
		// xotira, qolgani vaqtinchalik faylga). Ya'ni tokenni faqat tanadan
		// olsak, autentifikatsiyadan o'tmagan kishi ham serverni 20 MB yozishga
		// majburlay olardi. Query bilan esa yaroqsiz token darhol 401 oladi va
		// bayt ham o'qilmaydi.
		//
		// Query'da token qoldirish yangi sizish emas: `GET /rooms/:id/chat` va
		// `/state` allaqachon shu naqshda ishlaydi.
		//
		// Tana maydoni FALLBACK sifatida qoladi (klientlar birdaniga
		// ko'chmasligi uchun) — u holda himoya IP tezlik cheklovi bo'lib qoladi.
		token := c.Query("token")
		if token == "" {
			token = c.PostForm("token")
		}
		identity, name, ok := roomTokenIdentity(c, h, token, lessonID)
		if !ok {
			return
		}
		f, body, to, ok := chatUpload(c)
		if !ok {
			return
		}
		msg, err := h.Chat.UploadFromRoom(c.Request.Context(), lessonID, identity, name, f, body, to)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, msg)
	}
}

// RoomChatHistory godoc
// @Summary  Xona chat tarixi (ochiq — room-token). Ommaviy + o'z shaxsiy yozishmalari.
// @Tags     chat
// @Produce  json
// @Param    lessonID  path   string  true   "Lesson ID"
// @Param    token     query  string  true   "LiveKit room token"
// @Param    before    query  string  false  "Kursor (RFC3339)"
// @Param    limit     query  int     false  "Sahifa hajmi (max 50)"
// @Success  200  {object}  object{data=[]entity.ChatMessage}
// @Router   /api/v1/rooms/{lessonID}/chat [get]
func RoomChatHistory(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		lessonID := c.Param("lessonID")
		identity, _, ok := roomTokenIdentity(c, h, c.Query("token"), lessonID)
		if !ok {
			return
		}
		before, ok := parseBefore(c)
		if !ok {
			return
		}
		limit, _ := strconv.Atoi(c.Query("limit"))

		items, err := h.Chat.HistoryForRoom(c.Request.Context(), lessonID, identity, before, limit)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, items)
	}
}

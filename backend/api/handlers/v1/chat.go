package v1

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
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

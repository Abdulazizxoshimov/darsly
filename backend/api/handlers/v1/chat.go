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

// SendChat godoc
// @Summary      Chat xabari yuborish (host) — saqlanadi + LiveKit orqali tarqaladi
// @Tags         chat
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string             true  "Lesson ID"
// @Param        body  body  entity.SendChatReq  true  "Xabar"
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
		msg, err := h.Chat.Send(c.Request.Context(), mentorID, c.Param("id"), req.Body)
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

		var before *time.Time
		if raw := c.Query("before"); raw != "" {
			t, err := time.Parse(time.RFC3339, raw)
			if err != nil {
				hs.BadRequest(c, "before must be RFC3339 timestamp")
				return
			}
			before = &t
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

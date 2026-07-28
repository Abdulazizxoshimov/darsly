package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
)

// CreatePoll godoc
// @Summary  So'rovnoma yaratish (host)
// @Tags     poll
// @Security BearerAuth
// @Param    id    path  string               true  "Lesson ID"
// @Param    body  body  entity.CreatePollReq  true  "So'rovnoma"
// @Success  201  {object}  object{data=entity.Poll}
// @Router   /api/v1/lessons/{id}/polls [post]
func CreatePoll(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		var req entity.CreatePollReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		p, err := h.Poll.Create(c.Request.Context(), mentorID, c.Param("id"), req.Question, req.Options)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, p)
	}
}

// ListPolls godoc
// @Summary  Dars so'rovnomalari (host)
// @Tags     poll
// @Security BearerAuth
// @Param    id  path  string  true  "Lesson ID"
// @Success  200  {object}  object{data=[]entity.Poll}
// @Router   /api/v1/lessons/{id}/polls [get]
func ListPolls(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		items, err := h.Poll.ListByLesson(c.Request.Context(), mentorID, c.Param("id"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, items)
	}
}

// ClosePoll godoc
// @Summary  So'rovnomani yopish + natijalar (host)
// @Tags     poll
// @Security BearerAuth
// @Param    id  path  string  true  "Poll ID"
// @Success  200  {object}  object{data=entity.PollResults}
// @Router   /api/v1/polls/{id}/close [post]
func ClosePoll(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		res, err := h.Poll.Close(c.Request.Context(), mentorID, c.Param("id"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, res)
	}
}

// VotePoll godoc
// @Summary  So'rovnomaga ovoz berish (ochiq — LiveKit room-token bilan)
// @Tags     poll
// @Param    id    path  string          true  "Poll ID"
// @Param    body  body  entity.VoteReq   true  "Token + variant"
// @Success  204
// @Router   /api/v1/polls/{id}/vote [post]
func VotePoll(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.VoteReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		// Guest'ni LiveKit room-token orqali autentifikatsiya qilamiz.
		if h.LiveKit == nil || !h.LiveKit.Enabled() {
			hs.Error(c, nil)
			return
		}
		identity, _, room, err := h.LiveKit.VerifyToken(req.Token)
		if err != nil || identity == "" {
			hs.Unauthorized(c, "invalid room token")
			return
		}
		if err := h.Poll.Vote(c.Request.Context(), c.Param("id"), identity, room, req.OptionIndex); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// PollResults godoc
// @Summary  So'rovnoma natijalari — LiveKit room-token talab qilinadi
// @Tags     poll
// @Param    id     path   string  true  "Poll ID"
// @Param    token  query  string  true  "LiveKit room token"
// @Success  200  {object}  object{data=entity.PollResults}
// @Router   /api/v1/polls/{id}/results [get]
func PollResults(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Natija endi ochiq EMAS: xonada bo'lganlik isboti kerak. Ovoz berish
		// allaqachon token talab qilardi — natijani o'qish esa qilmasdi.
		token := c.Query("token")
		if token == "" {
			hs.Unauthorized(c, "room token is required")
			return
		}
		if h.LiveKit == nil || !h.LiveKit.Enabled() {
			hs.Error(c, nil)
			return
		}
		identity, _, room, err := h.LiveKit.VerifyToken(token)
		if err != nil || identity == "" {
			hs.Unauthorized(c, "invalid room token")
			return
		}
		res, err := h.Poll.Results(c.Request.Context(), c.Param("id"), room)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, res)
	}
}

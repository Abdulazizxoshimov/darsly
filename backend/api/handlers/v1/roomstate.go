package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
	"github.com/zoom/darsly/internal/usecase/shared"
)

// roomTokenIdentity — ochiq (JWT'siz) endpointlar uchun autentifikatsiya.
//
// Guest'da JWT yo'q, lekin uning LiveKit room-token'i bor va u imzolangan.
// Token'ning XONASI so'ralayotgan dars bilan mos kelishi SHART — aks holda
// bir darsning tokeni bilan boshqa darsning holatiga ta'sir qilish mumkin edi.
// Bu naqsh so'rovnomada allaqachon ishlatilgan (`VotePoll`).
func roomTokenIdentity(c *gin.Context, h *handlers.Handler, token, lessonID string) (identity, name string, ok bool) {
	if token == "" {
		hs.Unauthorized(c, "room token is required")
		return "", "", false
	}
	if h.LiveKit == nil || !h.LiveKit.Enabled() {
		hs.AbortError(c, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "video service is not available")
		return "", "", false
	}
	identity, name, room, err := h.LiveKit.VerifyToken(token)
	if err != nil || identity == "" {
		hs.Unauthorized(c, "invalid room token")
		return "", "", false
	}
	if room != shared.RoomName(lessonID) {
		hs.Forbidden(c, "room token is not valid for this lesson")
		return "", "", false
	}
	return identity, name, true
}

// SetHand godoc
// @Summary  Qo'l ko'tarish/tushirish (ochiq — LiveKit room-token bilan)
// @Tags     roomstate
// @Accept   json
// @Param    lessonID  path  string          true  "Lesson ID"
// @Param    body      body  entity.HandReq  true  "Token + holat"
// @Success  204
// @Router   /api/v1/rooms/{lessonID}/hand [post]
func SetHand(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.HandReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		lessonID := c.Param("lessonID")
		identity, name, ok := roomTokenIdentity(c, h, req.Token, lessonID)
		if !ok {
			return
		}
		if err := h.RoomState.SetHand(c.Request.Context(), lessonID, identity, name, req.Raised); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// SendReaction godoc
// @Summary  Emoji reaksiya (ochiq — LiveKit room-token bilan). Saqlanmaydi.
// @Tags     roomstate
// @Accept   json
// @Param    lessonID  path  string              true  "Lesson ID"
// @Param    body      body  entity.ReactionReq  true  "Token + emoji"
// @Success  204
// @Router   /api/v1/rooms/{lessonID}/reaction [post]
func SendReaction(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.ReactionReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		lessonID := c.Param("lessonID")
		identity, name, ok := roomTokenIdentity(c, h, req.Token, lessonID)
		if !ok {
			return
		}
		if err := h.RoomState.Reaction(c.Request.Context(), lessonID, identity, name, req.Emoji); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// GetRoomState godoc
// @Summary  Xona holati — ko'tarilgan qo'llar (kech kirgan klient shu bilan tiklanadi)
// @Tags     roomstate
// @Produce  json
// @Param    lessonID  path   string  true  "Lesson ID"
// @Param    token     query  string  true  "LiveKit room token"
// @Success  200  {object}  object{data=entity.RoomState}
// @Router   /api/v1/rooms/{lessonID}/state [get]
func GetRoomState(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		lessonID := c.Param("lessonID")
		identity, _, ok := roomTokenIdentity(c, h, c.Query("token"), lessonID)
		if !ok {
			return
		}
		st, err := h.RoomState.State(c.Request.Context(), lessonID, identity)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, st)
	}
}

// LowerHand godoc
// @Summary  Ishtirokchining qo'lini tushirish (host)
// @Tags     roomstate
// @Accept   json
// @Security BearerAuth
// @Param    id    path  string                true  "Lesson ID"
// @Param    body  body  entity.LowerHandReq   true  "Ishtirokchi identity"
// @Success  204
// @Router   /api/v1/lessons/{id}/hands/lower [post]
func LowerHand(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		var req entity.LowerHandReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		if err := h.RoomState.LowerHand(c.Request.Context(), mentorID, c.Param("id"), req.Identity); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// LowerAllHands godoc
// @Summary  Barcha qo'llarni tushirish (host)
// @Tags     roomstate
// @Security BearerAuth
// @Param    id  path  string  true  "Lesson ID"
// @Success  204
// @Router   /api/v1/lessons/{id}/hands/lower-all [post]
func LowerAllHands(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		if err := h.RoomState.LowerAll(c.Request.Context(), mentorID, c.Param("id")); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
)

// GetRoomToken godoc
// @Summary      Host (mentor) uchun LiveKit tokeni — xonani ochadi
// @Tags         room
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      200  {object}  object{data=entity.RoomToken}
// @Failure      403  {object}  object{code=string,message=string}
// @Router       /api/v1/lessons/{id}/token [post]
func GetRoomToken(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		lessonID := c.Param("id")
		rt, err := h.Room.HostToken(c.Request.Context(), mentorID, lessonID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, rt)
	}
}

// ListParticipants godoc
// @Summary      Xonadagi ishtirokchilar (host)
// @Tags         room
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      200  {object}  object{data=[]entity.RoomParticipant}
// @Router       /api/v1/lessons/{id}/participants [get]
func ListParticipants(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		items, err := h.Room.ListParticipants(c.Request.Context(), mentorID, c.Param("id"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, items)
	}
}

// MuteParticipant godoc
// @Summary      Ishtirokchini mute qilish (host)
// @Tags         room
// @Security     BearerAuth
// @Param        id        path  string  true  "Lesson ID"
// @Param        identity  path  string  true  "Participant identity"
// @Success      204
// @Router       /api/v1/lessons/{id}/participants/{identity}/mute [post]
func MuteParticipant(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		if err := h.Room.MuteParticipant(c.Request.Context(), mentorID, c.Param("id"), c.Param("identity"), true); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// RemoveParticipant godoc
// @Summary      Ishtirokchini chiqarib yuborish / kick (host)
// @Tags         room
// @Accept       json
// @Security     BearerAuth
// @Param        id        path  string  true  "Lesson ID"
// @Param        identity  path  string  true  "Participant identity"
// @Param        body      body  entity.RemoveParticipantReq  false  "scope: lesson (default) | mentor (doimiy qora ro'yxat)"
// @Success      204
// @Router       /api/v1/lessons/{id}/participants/{identity}/remove [post]
func RemoveParticipant(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		var req entity.RemoveParticipantReq
		_ = c.ShouldBindJSON(&req) // body ixtiyoriy: bo'sh = scope "lesson" (eski klientlar)
		if err := h.Room.RemoveParticipant(c.Request.Context(), mentorID, c.Param("id"), c.Param("identity"), req.Scope); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// MuteAll godoc
// @Summary      Hammani mute qilish (host) — ixtiyoriy allow_self_unmute bayrog'i bilan
// @Tags         room
// @Accept       json
// @Security     BearerAuth
// @Param        id    path  string            true   "Lesson ID"
// @Param        body  body  entity.MuteAllReq false  "allow_self_unmute berilsa dars bayrog'i ham yangilanadi"
// @Success      204
// @Router       /api/v1/lessons/{id}/mute-all [post]
func MuteAll(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		var req entity.MuteAllReq
		_ = c.ShouldBindJSON(&req) // body ixtiyoriy (eski klientlar bo'sh yuboradi)
		if err := h.Room.MuteAll(c.Request.Context(), mentorID, c.Param("id"), req.AllowSelfUnmute); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// ListBlocklist godoc
// @Summary  Mentorning doimiy qora ro'yxati (kick scope=mentor yozuvlari)
// @Tags     room
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  object{data=[]entity.BlocklistEntry}
// @Router   /api/v1/blocklist [get]
func ListBlocklist(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		items, err := h.Room.ListBlocklist(c.Request.Context(), mentorID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, items)
	}
}

// Unblock godoc
// @Summary  Qora ro'yxat yozuvini o'chirish (unban)
// @Tags     room
// @Security BearerAuth
// @Param    id  path  string  true  "Blocklist entry ID"
// @Success  204
// @Router   /api/v1/blocklist/{id} [delete]
func Unblock(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		if err := h.Room.Unblock(c.Request.Context(), mentorID, c.Param("id")); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// AllowSpeak godoc
// @Summary  Studentga so'zlash (media publish) ruxsatini berish (host)
// @Tags     room
// @Security BearerAuth
// @Param    id        path  string  true  "Lesson ID"
// @Param    identity  path  string  true  "Participant identity"
// @Success  204
// @Router   /api/v1/lessons/{id}/participants/{identity}/allow-speak [post]
func AllowSpeak(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		if err := h.Room.SetSpeakPermission(c.Request.Context(), mentorID, c.Param("id"), c.Param("identity"), true); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// RevokeSpeak godoc
// @Summary  Student so'zlash ruxsatini qaytarish (host)
// @Tags     room
// @Security BearerAuth
// @Param    id        path  string  true  "Lesson ID"
// @Param    identity  path  string  true  "Participant identity"
// @Success  204
// @Router   /api/v1/lessons/{id}/participants/{identity}/revoke-speak [post]
func RevokeSpeak(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		if err := h.Room.SetSpeakPermission(c.Request.Context(), mentorID, c.Param("id"), c.Param("identity"), false); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// EndLesson godoc
// @Summary      Darsni yakunlash — xonani yopadi, barchani uzadi
// @Tags         room
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      204
// @Failure      403  {object}  object{code=string,message=string}
// @Router       /api/v1/lessons/{id}/end [post]
func EndLesson(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		lessonID := c.Param("id")
		if err := h.Room.EndLesson(c.Request.Context(), mentorID, lessonID); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

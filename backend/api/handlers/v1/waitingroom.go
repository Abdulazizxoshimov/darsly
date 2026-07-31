package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
)

// ListWaitingRoom godoc
// @Summary      Kutayotgan so'rovlar ro'yxati (mentor)
// @Tags         waitingroom
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      200  {object}  object{data=[]entity.WaitingRoomRequest}
// @Router       /api/v1/lessons/{id}/waitingroom [get]
func ListWaitingRoom(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		lessonID := c.Param("id")
		items, err := h.WaitingRoom.ListPending(c.Request.Context(), mentorID, lessonID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, items)
	}
}

// AdmitWaitingRoom godoc
// @Summary      Kutish so'rovini qabul qilish (mentor) — guestga WS orqali token yuboriladi
// @Tags         waitingroom
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Request ID"
// @Success      200  {object}  object{data=entity.RoomToken}
// @Router       /api/v1/waitingroom/{id}/admit [post]
func AdmitWaitingRoom(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		requestID := c.Param("id")
		rt, err := h.WaitingRoom.Admit(c.Request.Context(), mentorID, requestID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, rt)
	}
}

// AdmitAllWaitingRoom godoc
// @Summary      Barcha kutayotganlarni kiritish (mentor) — har biriga WS orqali token yuboriladi
// @Description  Qisman muvaffaqiyat mumkin: javobda umumiy/kiritilgan/kiritilmagan sonlari.
// @Tags         waitingroom
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      200  {object}  object{data=entity.AdmitAllResp}
// @Router       /api/v1/lessons/{id}/waitingroom/admit-all [post]
func AdmitAllWaitingRoom(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		resp, err := h.WaitingRoom.AdmitAll(c.Request.Context(), mentorID, c.Param("id"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, resp)
	}
}

// RejectWaitingRoom godoc
// @Summary      Kutish so'rovini rad etish (mentor)
// @Tags         waitingroom
// @Security     BearerAuth
// @Param        id  path  string  true  "Request ID"
// @Success      204
// @Router       /api/v1/waitingroom/{id}/reject [post]
func RejectWaitingRoom(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		requestID := c.Param("id")
		if err := h.WaitingRoom.Reject(c.Request.Context(), mentorID, requestID); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// WaitingRoomStatus godoc
// @Summary      Kutish so'rovi holati (ochiq) — guest polling uchun (WS fallback)
// @Tags         waitingroom
// @Produce      json
// @Param        id  path  string  true  "Request ID"
// @Success      200  {object}  object{data=entity.WaitingRoomStatusResp}
// @Router       /api/v1/waitingroom/{id}/status [get]
func WaitingRoomStatus(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Param("id")
		resp, err := h.WaitingRoom.Status(c.Request.Context(), requestID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, resp)
	}
}

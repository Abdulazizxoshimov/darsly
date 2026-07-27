package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
)

// ListNotifications godoc
// @Summary      Bildirishnomalar ro'yxati
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Param        unread query bool false "Faqat o'qilmaganlar"
// @Success      200  {object}  object{data=[]entity.Notification,total=int}
// @Router       /api/v1/notifications [get]
func ListNotifications(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		var filter entity.NotificationFilter
		if err := c.ShouldBindQuery(&filter); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		items, total, err := h.Notification.List(c.Request.Context(), userID, &filter)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.List(c, items, total, filter.Page, filter.GetLimit())
	}
}

// UnreadCount godoc
// @Summary      O'qilmagan bildirishnomalar soni
// @Tags         notifications
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  object{data=object{count=int}}
// @Router       /api/v1/notifications/unread-count [get]
func UnreadCount(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		count, err := h.Notification.UnreadCount(c.Request.Context(), userID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, gin.H{"count": count})
	}
}

// MarkNotificationRead godoc
// @Summary      Bildirishnomani o'qilgan deb belgilash
// @Tags         notifications
// @Security     BearerAuth
// @Param        id  path  string  true  "Notification ID"
// @Success      204
// @Router       /api/v1/notifications/{id}/read [post]
func MarkNotificationRead(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		id := c.Param("id")
		if err := h.Notification.MarkRead(c.Request.Context(), userID, id); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// MarkAllNotificationsRead godoc
// @Summary      Barcha bildirishnomalarni o'qilgan deb belgilash
// @Tags         notifications
// @Security     BearerAuth
// @Success      204
// @Router       /api/v1/notifications/read-all [post]
func MarkAllNotificationsRead(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		if err := h.Notification.MarkAllRead(c.Request.Context(), userID); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

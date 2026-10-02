package v1

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
)

// WSConnect — autentifikatsiyalangan WebSocket (mentor kutish so'rovlarini shu yerda oladi).
// Token: Authorization header yoki ?token=<jwt> (brauzer WS header yubora olmaydi).
// @Summary  WebSocket ulanish (authed)
// @Tags     ws
// @Security BearerAuth
// @Router   /api/v1/ws [get]
func WSConnect(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		// Reconnect: ulanish ro'yxatga olingach (onRegistered) kutayotgan so'rovlar
		// snapshot'ini yetkazamiz. context.Background — WS upgrade (hijack) request
		// kontekstini bekor qilishi mumkin.
		h.Hub.ServeWS(c.Request.Context(), c.Writer, c.Request, userID, func() {
			h.WaitingRoom.DeliverPendingSnapshot(context.Background(), userID)
		})
	}
}

// WSGuestWaitingRoom — guest kutish xonasi WebSocket'i (ochiq).
// Guest ?request_id=<uuid> bilan ulanadi; admit/reject qarori real-time keladi.
// @Summary  Guest kutish xonasi WebSocket
// @Tags     ws
// @Param    request_id query string true "Waiting room request ID"
// @Router   /api/v1/ws/waitingroom [get]
func WSGuestWaitingRoom(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.Query("request_id")
		if requestID == "" {
			hs.BadRequest(c, "request_id is required")
			return
		}
		// So'rov mavjudligini tekshirish (upgrade'dan oldin).
		if _, err := h.WaitingRoom.Status(c.Request.Context(), requestID); err != nil {
			hs.Error(c, err)
			return
		}

		// Guestning WS identifikatori = requestID (unguessable UUID — bearer capability).
		// Ulanish ro'yxatga olingach, agar qaror allaqachon chiqqan bo'lsa darhol yetkazish (race).
		h.Hub.ServeWS(c.Request.Context(), c.Writer, c.Request, requestID, func() {
			h.WaitingRoom.DeliverCurrentStatus(context.Background(), requestID)
		})
	}
}

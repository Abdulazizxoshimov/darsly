package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
)

// TelegramStatus godoc
// @Summary      Telegram bog'lanish holati (mentor)
// @Description  `enabled=false` — serverda Telegram integratsiyasi sozlanmagan, UI yashirilsin.
// @Tags         telegram
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  object{data=entity.TelegramLinkStatus}
// @Router       /api/v1/me/telegram [get]
func TelegramStatus(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		st, err := h.Telegram.Status(c.Request.Context(), userID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, st)
	}
}

// TelegramStartLink godoc
// @Summary      Telegram bog'lash kodini olish (mentor)
// @Description  Bir martalik kod (15 daqiqa). Mentor botga `/start <kod>` yuboradi.
// @Tags         telegram
// @Produce      json
// @Security     BearerAuth
// @Success      201  {object}  object{data=entity.TelegramLink}
// @Router       /api/v1/me/telegram/link [post]
func TelegramStartLink(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		link, err := h.Telegram.StartLink(c.Request.Context(), userID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, link)
	}
}

// TelegramUnlink godoc
// @Summary      Telegram bog'lanishini uzish (mentor)
// @Tags         telegram
// @Security     BearerAuth
// @Success      204
// @Router       /api/v1/me/telegram [delete]
func TelegramUnlink(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		if err := h.Telegram.Unlink(c.Request.Context(), userID); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// GetRecording godoc
// @Summary      Bitta yozuv (mentor)
// @Description  Tiklash holatini poll qilish uchun: `status` = archived | restoring | ready.
// @Tags         recording
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Recording ID"
// @Success      200  {object}  object{data=entity.Recording}
// @Router       /api/v1/recordings/{id} [get]
func GetRecording(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		rec, err := h.Recording.Get(c.Request.Context(), mentorID, c.Param("id"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, rec)
	}
}

// RestoreRecording godoc
// @Summary      Arxivlangan yozuvni Telegramdan qaytarib olish (mentor)
// @Description  DARHOL 202 qaytaradi (`status: restoring`). Yuklab olish 30-60 s davom etadi —
// @Description  klient `GET /api/v1/recordings/{id}` ni `poll_after_s` oralig'ida so'rab turadi.
// @Tags         recording
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Recording ID"
// @Success      202  {object}  object{data=entity.RecordingRestore}
// @Router       /api/v1/recordings/{id}/restore [post]
func RestoreRecording(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		res, err := h.Recording.Restore(c.Request.Context(), mentorID, c.Param("id"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		// 202 Accepted — ish QABUL QILINDI, lekin hali tugamagan. 200 bo'lsa
		// klient "tayyor" deb tushunib, darhol yuklab olishga urinardi.
		hs.Accepted(c, res)
	}
}

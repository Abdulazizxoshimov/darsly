package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
)

// GetLessonArchive godoc
// @Summary      Dars arxivi (host) — video + chat + materiallar bitta so'rovda
// @Description  O'tgan dars sahifasi uchun: yozuv (presigned havola bilan, agar
// @Description  `ready` bo'lsa), chat tarixi (har xabarda `offset_sec` — dars
// @Description  boshidan siljish, pleyerni sakratish uchun) va darsda ulashilgan
// @Description  fayllar. Yozuv yo'q bo'lsa `recording: null`.
// @Tags         archive
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      200  {object}  object{data=entity.LessonArchive}
// @Router       /api/v1/lessons/{id}/archive [get]
func GetLessonArchive(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		a, err := h.Archive.Get(c.Request.Context(), mentorID, c.Param("id"))
		if err != nil {
			hs.Error(c, err)
			return
		}
		// Javobda vaqtinchalik presigned havolalar bor — oraliq keshlarda
		// qolib ketmasin (havola boshqa foydalanuvchiga tushishi mumkin edi).
		c.Header("Cache-Control", "no-store")
		hs.Success(c, a)
	}
}

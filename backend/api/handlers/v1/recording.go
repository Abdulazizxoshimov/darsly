package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
)

// StartRecording godoc
// @Summary      Dars yozib olishni boshlash (mentor) — LiveKit Egress → MinIO
// @Tags         recording
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      201  {object}  object{data=entity.Recording}
// @Router       /api/v1/lessons/{id}/recording/start [post]
func StartRecording(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		lessonID := c.Param("id")
		rec, err := h.Recording.StartRecording(c.Request.Context(), mentorID, lessonID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, rec)
	}
}

// StopRecording godoc
// @Summary      Yozib olishni to'xtatish (mentor)
// @Tags         recording
// @Security     BearerAuth
// @Param        id  path  string  true  "Recording ID"
// @Success      204
// @Router       /api/v1/recordings/{id}/stop [post]
func StopRecording(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		recordingID := c.Param("id")
		if err := h.Recording.StopRecording(c.Request.Context(), mentorID, recordingID); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// ListRecordings godoc
// @Summary      Dars yozuvlari ro'yxati (mentor)
// @Tags         recording
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      200  {object}  object{data=[]entity.Recording}
// @Router       /api/v1/lessons/{id}/recordings [get]
func ListRecordings(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		lessonID := c.Param("id")
		items, err := h.Recording.ListByLesson(c.Request.Context(), mentorID, lessonID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, items)
	}
}

// DownloadRecording godoc
// @Summary      Yozuv uchun vaqtinchalik yuklab olish havolasi (mentor)
// @Tags         recording
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Recording ID"
// @Success      200  {object}  object{data=entity.RecordingDownload}
// @Router       /api/v1/recordings/{id}/download [get]
func DownloadRecording(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		recordingID := c.Param("id")
		dl, err := h.Recording.DownloadURL(c.Request.Context(), mentorID, recordingID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, dl)
	}
}

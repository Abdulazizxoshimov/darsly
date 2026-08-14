package v1

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
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

// LocalStartRecording — telefon lokal yozuvni boshlaganda yozuv qatorini yaratadi
// (client-side «Zoom local recording»). Server egress'ini ishlatmaydi.
// @Router       /api/v1/lessons/{id}/recording/local-start [post]
func LocalStartRecording(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		lessonID := c.Param("id")
		rec, err := h.Recording.LocalStart(c.Request.Context(), mentorID, lessonID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, rec)
	}
}

// LocalUploadURL — telefon faylni to'g'ridan MinIO'ga PUT qilishi uchun havola.
// @Router       /api/v1/recordings/{id}/upload-url [post]
func LocalUploadURL(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		recordingID := c.Param("id")
		url, err := h.Recording.LocalUploadURL(c.Request.Context(), mentorID, recordingID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, gin.H{"url": url})
	}
}

// LocalCompleteRecording — telefon yuklab bo'lgach yozuvni tayyor qiladi.
// @Router       /api/v1/recordings/{id}/complete [post]
func LocalCompleteRecording(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		recordingID := c.Param("id")
		var req struct {
			DurationSec int    `json:"duration_sec"`
			EndedAt     string `json:"ended_at"` // RFC3339, ixtiyoriy (bo'sh → now)
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.Error(c, apperr.BadRequest("noto'g'ri so'rov"))
			return
		}
		var endedAt time.Time
		if req.EndedAt != "" {
			endedAt, _ = time.Parse(time.RFC3339, req.EndedAt)
		}
		if err := h.Recording.LocalComplete(c.Request.Context(), mentorID, recordingID, req.DurationSec, endedAt); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
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

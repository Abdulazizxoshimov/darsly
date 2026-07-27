package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
)

// CreateLesson godoc
// @Summary      Yangi dars yaratish
// @Tags         lessons
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  entity.CreateLessonReq  true  "Dars ma'lumotlari"
// @Success      201  {object}  object{data=entity.Lesson}
// @Router       /api/v1/lessons [post]
func CreateLesson(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		var req entity.CreateLessonReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		lesson, err := h.Lesson.Create(c.Request.Context(), mentorID, &req)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, lesson)
	}
}

// ListLessons godoc
// @Summary      Mentorning darslari ro'yxati
// @Tags         lessons
// @Produce      json
// @Security     BearerAuth
// @Param        page   query  int     false  "Sahifa"
// @Param        limit  query  int     false  "Hajm"
// @Param        status query  string  false  "Status filtri"
// @Success      200  {object}  object{data=[]entity.Lesson,total=int}
// @Router       /api/v1/lessons [get]
func ListLessons(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		var filter entity.LessonFilter
		if err := c.ShouldBindQuery(&filter); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		lessons, total, err := h.Lesson.ListByMentor(c.Request.Context(), mentorID, &filter)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.List(c, lessons, total, filter.Page, filter.GetLimit())
	}
}

// GetLesson godoc
// @Summary      Darsni ID bo'yicha olish
// @Tags         lessons
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      200  {object}  object{data=entity.Lesson}
// @Router       /api/v1/lessons/{id} [get]
func GetLesson(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		id := c.Param("id")
		lesson, err := h.Lesson.GetByID(c.Request.Context(), mentorID, id)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, lesson)
	}
}

// UpdateLesson godoc
// @Summary      Darsni yangilash
// @Tags         lessons
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string                 true  "Lesson ID"
// @Param        body  body  entity.UpdateLessonReq  true  "Yangilash ma'lumotlari"
// @Success      200  {object}  object{data=entity.Lesson}
// @Router       /api/v1/lessons/{id} [patch]
func UpdateLesson(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		id := c.Param("id")
		var req entity.UpdateLessonReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		lesson, err := h.Lesson.Update(c.Request.Context(), mentorID, id, &req)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, lesson)
	}
}

// DeleteLesson godoc
// @Summary      Darsni o'chirish (soft-delete)
// @Tags         lessons
// @Security     BearerAuth
// @Param        id  path  string  true  "Lesson ID"
// @Success      204
// @Router       /api/v1/lessons/{id} [delete]
func DeleteLesson(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		mentorID := c.GetString(middleware.CtxUserID)
		id := c.Param("id")
		if err := h.Lesson.Delete(c.Request.Context(), mentorID, id); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/internal/entity"
)

// PreviewJoinLink godoc
// @Summary      Havola (slug) orqali dars ma'lumotini ko'rish (ochiq)
// @Tags         joinlink
// @Produce      json
// @Param        slug  path  string  true  "Join slug"
// @Success      200  {object}  object{data=entity.LessonPublic}
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /api/v1/joinlink/{slug} [get]
func PreviewJoinLink(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		lesson, err := h.JoinLink.Preview(c.Request.Context(), slug)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, lesson)
	}
}

// JoinLink godoc
// @Summary      Havola orqali darsga kirish (guest/parol tekshiruvi bilan)
// @Tags         joinlink
// @Accept       json
// @Produce      json
// @Param        slug  path  string               true  "Join slug"
// @Param        body  body  entity.JoinLessonReq  false "Guest ismi va/yoki parol"
// @Success      200  {object}  object{data=entity.JoinLessonResp}
// @Failure      401  {object}  object{code=string,message=string}
// @Failure      403  {object}  object{code=string,message=string}
// @Router       /api/v1/joinlink/{slug} [post]
func JoinLink(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		slug := c.Param("slug")
		var req entity.JoinLessonReq
		_ = c.ShouldBindJSON(&req) // body ixtiyoriy (parolsiz dars bo'lishi mumkin)
		// `c.ClientIP()` — lockout'ni klient bo'yicha ajratish uchun (M7). Reverse-proxy
		// ortida to'g'ri ishlashi Gin'ning TrustedProxies sozlamasiga bog'liq.
		resp, err := h.JoinLink.Join(c.Request.Context(), slug, c.ClientIP(), &req)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, resp)
	}
}

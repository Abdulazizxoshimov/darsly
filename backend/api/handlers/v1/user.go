package v1

import (
	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// ListUsers godoc
// @Summary      List users
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        page  query  int     false  "Page number"
// @Param        limit query  int     false  "Page size"
// @Success      200  {object}  object{data=[]entity.UserShort,total=int,page=int,limit=int,total_pages=int}
// @Failure      401  {object}  object{code=string,message=string}
// @Router       /api/v1/users [get]
func ListUsers(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var filter entity.UserFilter
		if err := c.ShouldBindQuery(&filter); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		users, total, err := h.User.List(c.Request.Context(), &filter)
		if err != nil {
			hs.Error(c, err)
			return
		}
		// Yengil ko'rinish — ro'yxatda to'liq User (last_login, timezone, ...) shart emas.
		short := make([]entity.UserShort, len(users))
		for i, u := range users {
			short[i] = u.ToShort()
		}
		hs.List(c, short, total, filter.Page, filter.GetLimit())
	}
}

// GetUser godoc
// @Summary      Get user by ID
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Param        id   path  string  true  "User ID"
// @Success      200  {object}  object{data=entity.User}
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /api/v1/users/{id} [get]
func GetUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		// IDOR himoyasi: faqat admin istalgan profilni ko'ra oladi; mentor/student/o'zga
		// rol faqat o'z profilini (aks holda boshqa foydalanuvchi ma'lumoti sizib chiqadi).
		// Mentor endi global admin EMAS (H-2) — boshqa foydalanuvchi profilini ko'ra olmaydi.
		if c.GetString(middleware.CtxRole) != "admin" && c.GetString(middleware.CtxUserID) != id {
			hs.Error(c, apperr.Forbidden("access denied"))
			return
		}
		user, err := h.User.GetByID(c.Request.Context(), id)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, user)
	}
}

// CreateUser godoc
// @Summary      Create user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  entity.CreateUserReq  true  "User data"
// @Success      201  {object}  object{data=entity.User}
// @Failure      400  {object}  object{code=string,message=string}
// @Router       /api/v1/users [post]
func CreateUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.CreateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		user, err := h.User.Create(c.Request.Context(), &req)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, user)
	}
}

// UpdateUser godoc
// @Summary      Update user
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id    path  string               true  "User ID"
// @Param        body  body  entity.UpdateUserReq  true  "Update data"
// @Success      200  {object}  object{data=entity.User}
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /api/v1/users/{id} [put]
func UpdateUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		var req entity.UpdateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		user, err := h.User.Update(c.Request.Context(), id, &req)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, user)
	}
}

// AdminResetPassword godoc
// @Summary      Reset a user's password (mentor/admin)
// @Description  Mentor boshqa foydalanuvchi parolini joriy-parolsiz tiklaydi. Path'dagi
// @Description  :id maqsad foydalanuvchi. Reset'dan so'ng maqsadning barcha sessiyalari bekor bo'ladi.
// @Tags         users
// @Accept       json
// @Security     BearerAuth
// @Param        id    path  string                         true  "User ID"
// @Param        body  body  entity.AdminResetPasswordReq   true  "New password"
// @Success      204
// @Failure      400  {object}  object{code=string,message=string}
// @Failure      403  {object}  object{code=string,message=string}
// @Router       /api/v1/users/{id}/password [put]
func AdminResetPassword(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		// :id — maqsad foydalanuvchi (avvalgi bug: CtxUserID ishlatilib :id e'tiborsiz qolar edi,
		// natijada mentor faqat o'z parolini o'zgartira olardi). RBAC bu route'ni mentorga cheklaydi.
		id := c.Param("id")
		var req entity.AdminResetPasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		callerID := c.GetString(middleware.CtxUserID)
		callerRole := c.GetString(middleware.CtxRole)
		if err := h.User.ResetPassword(c.Request.Context(), callerID, callerRole, id, req.NewPassword); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// GetCurrentUser godoc
// @Summary      Get current user profile
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  object{data=entity.User}
// @Router       /api/v1/users/me [get]
func GetCurrentUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		user, err := h.User.GetByID(c.Request.Context(), userID)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, user)
	}
}

// UpdateCurrentUser godoc
// @Summary      Update current user profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body  entity.UpdateUserReq  true  "Update data"
// @Success      200  {object}  object{data=entity.User}
// @Router       /api/v1/users/me [put]
func UpdateCurrentUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		var req entity.UpdateUserReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		// Xavfsizlik: foydalanuvchi o'z rolini o'zgartira olmaydi (privilege escalation oldini olish).
		// Rolni faqat mentor `PUT /users/:id` orqali o'zgartira oladi.
		req.Role = nil
		user, err := h.User.Update(c.Request.Context(), userID, &req)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, user)
	}
}

// DeleteCurrentUser godoc
// @Summary      Delete current user's own account (M5 — Play Store majburiyati)
// @Description  Foydalanuvchi o'z akkauntini o'chiradi. Sessiyalar bekor qilinadi.
// @Tags         users
// @Security     BearerAuth
// @Success      204
// @Router       /api/v1/users/me [delete]
func DeleteCurrentUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		// `User.Delete` — soft-delete + barcha sessiyalarni bekor qilish.
		// Foydalanuvchi allaqachon autentifikatsiyadan o'tган (o'z akkaunti),
		// shuning uchun qo'shimcha egalik tekshiruvi shart emas.
		if err := h.User.Delete(c.Request.Context(), userID); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// ChangeCurrentPassword godoc
// @Summary      Change current user password
// @Tags         users
// @Accept       json
// @Security     BearerAuth
// @Param        body  body  entity.ChangePasswordReq  true  "Passwords"
// @Success      204
// @Router       /api/v1/users/me/password [put]
func ChangeCurrentPassword(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetString(middleware.CtxUserID)
		var req entity.ChangePasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		if err := h.User.ChangePassword(c.Request.Context(), userID, &req); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// DeleteUser godoc
// @Summary      Delete user (soft-delete)
// @Tags         users
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID"
// @Success      204
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /api/v1/users/{id} [delete]
func DeleteUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := h.User.Delete(c.Request.Context(), id); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// DeactivateUser godoc
// @Summary      Deactivate user
// @Tags         users
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID"
// @Success      204
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /api/v1/users/{id}/deactivate [post]
func DeactivateUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := h.User.Deactivate(c.Request.Context(), id); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

// ActivateUser godoc
// @Summary      Activate user
// @Tags         users
// @Security     BearerAuth
// @Param        id  path  string  true  "User ID"
// @Success      204
// @Failure      404  {object}  object{code=string,message=string}
// @Router       /api/v1/users/{id}/activate [post]
func ActivateUser(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		if err := h.User.Activate(c.Request.Context(), id); err != nil {
			hs.Error(c, err)
			return
		}
		hs.NoContent(c)
	}
}

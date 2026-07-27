package v1

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/zoom/darsly/api/handlers"
	hs "github.com/zoom/darsly/api/http_status"
	"github.com/zoom/darsly/api/middleware"
	"github.com/zoom/darsly/internal/entity"
)

// Register godoc
// @Summary Register new user
// @Tags auth
// @Accept json
// @Produce json
// @Param body body entity.RegisterReq true "registration data"
// @Success 201 {object} entity.TokenPair
// @Router /api/v1/auth/register [post]
func Register(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !h.AllowOpenRegistration {
			hs.Forbidden(c, "Open registration is disabled. Ask an admin to invite you.")
			return
		}
		var req entity.RegisterReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		ip := c.ClientIP()
		ua := c.GetHeader("User-Agent")
		tokens, err := h.Auth.Register(c.Request.Context(), &req, ip, ua)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Created(c, tokens)
	}
}

// Login godoc
// @Summary Login
// @Tags auth
// @Accept json
// @Produce json
// @Param body body entity.LoginReq true "credentials"
// @Success 200 {object} entity.TokenPair
// @Router /api/v1/auth/login [post]
func Login(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.LoginReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		ip := c.ClientIP()
		ua := c.GetHeader("User-Agent")
		tokens, err := h.Auth.Login(c.Request.Context(), &req, ip, ua)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, tokens)
	}
}

// Refresh godoc
// @Summary Refresh tokens
// @Tags auth
// @Accept json
// @Produce json
// @Param body body entity.RefreshReq true "refresh token"
// @Success 200 {object} entity.TokenPair
// @Router /api/v1/auth/refresh [post]
func Refresh(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.RefreshReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		tokens, err := h.Auth.Refresh(c.Request.Context(), &req)
		if err != nil {
			hs.Error(c, err)
			return
		}
		hs.Success(c, tokens)
	}
}

// Logout godoc
// @Summary Logout
// @Tags auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Router /api/v1/auth/logout [post]
func Logout(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.LogoutReq
		_ = c.ShouldBindJSON(&req)
		if err := h.Auth.Logout(c.Request.Context(), &req); err != nil {
			hs.Error(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// ForgotPassword godoc
// @Summary Forgot password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body entity.ForgotPasswordReq true "email"
// @Router /api/v1/auth/forgot-password [post]
func ForgotPassword(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.ForgotPasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		_ = h.Auth.ForgotPassword(c.Request.Context(), &req)
		c.Status(http.StatusNoContent)
	}
}

// ResetPassword godoc
// @Summary Reset password
// @Tags auth
// @Accept json
// @Produce json
// @Param body body entity.ResetPasswordReq true "new password"
// @Router /api/v1/auth/reset-password [post]
func ResetPassword(h *handlers.Handler) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req entity.ResetPasswordReq
		if err := c.ShouldBindJSON(&req); err != nil {
			hs.BadRequest(c, err.Error())
			return
		}
		if err := h.Auth.ResetPassword(c.Request.Context(), &req); err != nil {
			hs.Error(c, err)
			return
		}
		c.Status(http.StatusNoContent)
	}
}

// Me godoc
// @Summary Get current user
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} entity.User
// @Router /api/v1/auth/me [get]
func Me(h *handlers.Handler) gin.HandlerFunc {
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

package http_status

import (
	"errors"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"

	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// jsonData nil slice'ni bo'sh slice'ga aylantiradi — JSON'da `null` emas `[]` chiqsin.
// Frontend ro'yxatlarni massiv deb kutadi (.length/.map); `null` qaytса mijozда crash bo'ladi.
func jsonData(data any) any {
	if data == nil {
		return data
	}
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Slice && v.IsNil() {
		return reflect.MakeSlice(v.Type(), 0, 0).Interface()
	}
	return data
}

type successResponse struct {
	Data any `json:"data"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type listResponse struct {
	Data       any `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"total_pages"`
}

// Success sends a 200 JSON response with the given data.
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, successResponse{Data: jsonData(data)})
}

// Created sends a 201 JSON response with the given data.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, successResponse{Data: jsonData(data)})
}

// NoContent sends a 204 response with no body.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// List sends a paginated list response.
func List(c *gin.Context, data any, total, page, limit int) {
	totalPages := 0
	if limit > 0 && total > 0 {
		totalPages = (total + limit - 1) / limit
	}
	c.JSON(http.StatusOK, listResponse{
		Data:       jsonData(data),
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	})
}

// Error sends an error JSON response, deriving the HTTP status from the error type.
// Xavfsizlik: klientга FAQAT AppError.Message ko'rsatiladi (wrap qilingan ichki xato —
// LiveKit/SQL/host tafsilotlari — oshkor qilinmaydi). To'liq xato Sentry/log'ga beriladi.
func Error(c *gin.Context, err error) {
	if err == nil {
		InternalError(c)
		return
	}
	// To'liq xatoni Sentry/logging middleware ushlashi uchun kontekstga qo'yamiz.
	_ = c.Error(err)

	status := HTTPStatusFromError(err)
	code := CodeFromError(err)

	msg := "internal server error"
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		msg = ae.Message // faqat xavfsiz, oldindan belgilangan matn (Err ilova qilinmaydi)
	}
	c.JSON(status, errorResponse{Code: code, Message: msg})
}

// AbortError — middleware'lar va ochiq handler'lar uchun: so'rovni to'xtatib
// xatoni AYNI shu {code, message} konvertida qaytaradi (Error() bilan bir xil).
//
// Nega kerak: middleware qatlamida `apperr.AppError` bo'lmaydi (Casbin/rate-limit/JWT
// xatolari), shuning uchun Error()'ni ishlatolmaydi. Ilgari ular `{error, code}`
// qaytarardi — natijada klient (web + mobil) bitta ApiError modeliga parse qila
// olmasdi va eng muhim holatlarda (401 TOKEN_EXPIRED, 403, 429) foydalanuvchiga
// ko'rsatiladigan matn bo'sh chiqardi. Endi barcha xato javoblari bir xil shaklda.
func AbortError(c *gin.Context, status int, code, msg string) {
	c.AbortWithStatusJSON(status, errorResponse{Code: code, Message: msg})
}

// BadRequest sends a 400 response.
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, errorResponse{Code: "BAD_REQUEST", Message: msg})
}

// Unauthorized sends a 401 response.
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, errorResponse{Code: "UNAUTHORIZED", Message: msg})
}

// Forbidden sends a 403 response.
func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, errorResponse{Code: "FORBIDDEN", Message: msg})
}

// NotFound sends a 404 response.
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, errorResponse{Code: "NOT_FOUND", Message: msg})
}

// InternalError sends a 500 response.
func InternalError(c *gin.Context) {
	c.JSON(http.StatusInternalServerError, errorResponse{Code: "INTERNAL_ERROR", Message: "internal server error"})
}

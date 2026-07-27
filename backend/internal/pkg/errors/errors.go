package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a machine-readable error identifier.
type Code string

const (
	CodeNotFound     Code = "NOT_FOUND"
	CodeConflict     Code = "CONFLICT"
	CodeForbidden    Code = "FORBIDDEN"
	CodeUnauthorized Code = "UNAUTHORIZED"
	CodeValidation   Code = "VALIDATION_ERROR"
	CodeBadRequest   Code = "BAD_REQUEST"
	CodeInternal     Code = "INTERNAL_ERROR"
)

// AppError is the standard domain error that carries an HTTP status, a machine
// code, and a human-readable message.
type AppError struct {
	HTTPStatus int
	Code       Code
	Message    string
	Err        error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error { return e.Err }

func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// ─── Constructors ─────────────────────────────────────────────────────────────

func NotFound(entity string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusNotFound,
		Code:       CodeNotFound,
		Message:    entity + " not found",
	}
}

func Conflict(msg string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusConflict,
		Code:       CodeConflict,
		Message:    msg,
	}
}

func Forbidden(msg string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusForbidden,
		Code:       CodeForbidden,
		Message:    msg,
	}
}

func Unauthorized(msg string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusUnauthorized,
		Code:       CodeUnauthorized,
		Message:    msg,
	}
}

func Validation(msg string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusUnprocessableEntity,
		Code:       CodeValidation,
		Message:    msg,
	}
}

func BadRequest(msg string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusBadRequest,
		Code:       CodeBadRequest,
		Message:    msg,
	}
}

func Internal(err error) *AppError {
	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       CodeInternal,
		Message:    "internal server error",
		Err:        err,
	}
}

func Wrap(err error, msg string) *AppError {
	return &AppError{
		HTTPStatus: http.StatusInternalServerError,
		Code:       CodeInternal,
		Message:    msg,
		Err:        err,
	}
}

// ─── Type checks ──────────────────────────────────────────────────────────────

func As(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return nil
}

func IsNotFound(err error) bool {
	ae := As(err)
	return ae != nil && ae.Code == CodeNotFound
}

func IsConflict(err error) bool {
	ae := As(err)
	return ae != nil && ae.Code == CodeConflict
}

func IsValidation(err error) bool {
	ae := As(err)
	return ae != nil && ae.Code == CodeValidation
}

func IsForbidden(err error) bool {
	ae := As(err)
	return ae != nil && ae.Code == CodeForbidden
}

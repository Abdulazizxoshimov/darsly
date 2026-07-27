package http_status

import (
	"errors"
	"net/http"

	apperr "github.com/zoom/darsly/internal/pkg/errors"
)

// HTTPStatusFromError maps a domain AppError to the appropriate HTTP status code.
// Falls back to 500 for unknown errors.
func HTTPStatusFromError(err error) int {
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		return ae.HTTPStatus
	}
	return http.StatusInternalServerError
}

// CodeFromError extracts the machine-readable error code string.
func CodeFromError(err error) string {
	var ae *apperr.AppError
	if errors.As(err, &ae) {
		return string(ae.Code)
	}
	return string(apperr.CodeInternal)
}

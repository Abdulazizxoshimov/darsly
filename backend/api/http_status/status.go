package http_status

import "net/http"

const (
	StatusOK        = http.StatusOK
	StatusCreated   = http.StatusCreated
	StatusNoContent = http.StatusNoContent

	StatusBadRequest          = http.StatusBadRequest
	StatusUnauthorized        = http.StatusUnauthorized
	StatusForbidden           = http.StatusForbidden
	StatusNotFound            = http.StatusNotFound
	StatusConflict            = http.StatusConflict
	StatusUnprocessableEntity = http.StatusUnprocessableEntity

	StatusInternalServerError = http.StatusInternalServerError
)

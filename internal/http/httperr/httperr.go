package httperr

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Message }

func NewUnauthorized(msg string) *APIError {
	return &APIError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: msg}
}

func NewForbidden(msg string) *APIError {
	return &APIError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: msg}
}

func NewBadRequest(code, msg string) *APIError {
	return &APIError{Status: http.StatusBadRequest, Code: code, Message: msg}
}

func NewValidation(msg string) *APIError {
	return &APIError{Status: http.StatusUnprocessableEntity, Code: "VALIDATION_ERROR", Message: msg}
}

func NewNotFound(msg string) *APIError {
	return &APIError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: msg}
}

func NewConflict(code, msg string) *APIError {
	return &APIError{Status: http.StatusConflict, Code: code, Message: msg}
}

func NewInternal() *APIError {
	return &APIError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "Terjadi kesalahan internal, coba lagi nanti"}
}

type errorBody struct {
	Error *APIError `json:"error"`
}

// Write writes a JSON error response. If err is *APIError it uses its status,
// otherwise it returns 500 without leaking internal details.
func Write(c echo.Context, err error) error {
	if apiErr, ok := err.(*APIError); ok {
		return c.JSON(apiErr.Status, errorBody{Error: apiErr})
	}
	internal := NewInternal()
	return c.JSON(internal.Status, errorBody{Error: internal})
}

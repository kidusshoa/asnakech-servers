package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code is a stable machine-readable error identifier for API clients.
type Code string

const (
	CodeInternal      Code = "internal_error"
	CodeValidation    Code = "validation_error"
	CodeUnauthorized  Code = "unauthorized"
	CodeForbidden     Code = "forbidden"
	CodeNotFound      Code = "not_found"
	CodeConflict      Code = "conflict"
	CodeRateLimited   Code = "rate_limited"
)

// Error is the application-level error used across handlers and services.
type Error struct {
	Code       Code
	Message    string
	HTTPStatus int
	Err        error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Err
}

func New(code Code, message string, httpStatus int) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

func Wrap(err error, code Code, message string, httpStatus int) *Error {
	return &Error{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
		Err:        err,
	}
}

func Internal(message string) *Error {
	return New(CodeInternal, message, http.StatusInternalServerError)
}

func Validation(message string) *Error {
	return New(CodeValidation, message, http.StatusBadRequest)
}

func Unauthorized(message string) *Error {
	return New(CodeUnauthorized, message, http.StatusUnauthorized)
}

func Forbidden(message string) *Error {
	return New(CodeForbidden, message, http.StatusForbidden)
}

func NotFound(message string) *Error {
	return New(CodeNotFound, message, http.StatusNotFound)
}

func Conflict(message string) *Error {
	return New(CodeConflict, message, http.StatusConflict)
}

// As extracts an *Error from an error chain.
func As(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

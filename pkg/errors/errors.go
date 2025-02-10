package errors

import (
	"fmt"
)

// ErrorCode represents a unique error code
type ErrorCode int

// AppError represents an application error
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Err     error     `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// Common error codes
const (
	ErrNotFound ErrorCode = iota + 1000
	ErrBadRequest
	ErrUnauthorized
	ErrForbidden
	ErrInternal
)

// Error constructors
func NewNotFound(resource string, err error) *AppError {
	return &AppError{
		Code:    ErrNotFound,
		Message: fmt.Sprintf("%s not found", resource),
		Err:     err,
	}
}

func NewBadRequest(message string, err error) *AppError {
	return &AppError{
		Code:    ErrBadRequest,
		Message: message,
		Err:     err,
	}
}

func NewInternal(err error) *AppError {
	return &AppError{
		Code:    ErrInternal,
		Message: "internal server error",
		Err:     err,
	}
}

// Common errors
func NotFound(resource string, err error) *AppError {
	return NewNotFound(resource, err)
}

func BadRequest(message string, err error) *AppError {
	return NewBadRequest(message, err)
}

func Internal(err error) *AppError {
	return NewInternal(err)
}

func Unauthorized(err error) *AppError {
	return &AppError{
		Code:    ErrUnauthorized,
		Message: "unauthorized",
		Err:     err,
	}
}

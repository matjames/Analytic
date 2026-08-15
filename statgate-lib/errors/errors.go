package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Code string

const (
	CodeNotFound       Code = "NOT_FOUND"
	CodeUnauthorized   Code = "UNAUTHORIZED"
	CodeForbidden      Code = "FORBIDDEN"
	CodeBadRequest     Code = "BAD_REQUEST"
	CodeConflict       Code = "CONFLICT"
	CodeInternal       Code = "INTERNAL_ERROR"
	CodeUnavailable    Code = "SERVICE_UNAVAILABLE"
	CodeValidationFail Code = "VALIDATION_FAILED"
	CodeRateLimited    Code = "RATE_LIMITED"
)

type AppError struct {
	Code       Code                   `json:"code"`
	Message    string                 `json:"message"`
	HTTPStatus int                    `json:"http_status"`
	Details    map[string]interface{} `json:"details,omitempty"`
	Err        error                  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code Code, msg string, status int, details map[string]interface{}) *AppError {
	return &AppError{
		Code:       code,
		Message:    msg,
		HTTPStatus: status,
		Details:    details,
	}
}

func NotFound(msg string) *AppError {
	return New(CodeNotFound, msg, http.StatusNotFound, nil)
}

func Unauthorized(msg string) *AppError {
	return New(CodeUnauthorized, msg, http.StatusUnauthorized, nil)
}

func Forbidden(msg string) *AppError {
	return New(CodeForbidden, msg, http.StatusForbidden, nil)
}

func BadRequest(msg string, details ...map[string]interface{}) *AppError {
	var d map[string]interface{}
	if len(details) > 0 {
		d = details[0]
	}
	return New(CodeBadRequest, msg, http.StatusBadRequest, d)
}

func Internal(err error, msg string) *AppError {
	return &AppError{
		Code:       CodeInternal,
		Message:    msg,
		HTTPStatus: http.StatusInternalServerError,
		Err:        err,
	}
}

func WriteHTTP(w http.ResponseWriter, err *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)
	_ = json.NewEncoder(w).Encode(err)
}

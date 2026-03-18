package errs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
)

type ErrCode uint

const (
	OK               ErrCode = 0
	Cancelled        ErrCode = 1
	Unknown          ErrCode = 2
	InvalidArgument  ErrCode = 3
	NotFound         ErrCode = 5
	AlreadyExists    ErrCode = 6
	PermissionDenied ErrCode = 7
	Internal         ErrCode = 13
	Unauthenticated  ErrCode = 16
)

var httpStatus = map[ErrCode]int{
	OK:               http.StatusOK,
	Cancelled:        http.StatusRequestTimeout,
	Unknown:          http.StatusInternalServerError,
	InvalidArgument:  http.StatusBadRequest,
	NotFound:         http.StatusNotFound,
	AlreadyExists:    http.StatusConflict,
	PermissionDenied: http.StatusForbidden,
	Internal:         http.StatusInternalServerError,
	Unauthenticated:  http.StatusUnauthorized,
}

type Error struct {
	Code     ErrCode `json:"code"`
	Message  string  `json:"message"`
	FuncName string  `json:"-"`
}

func New(code ErrCode, err error) *Error {
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	return &Error{
		Code:     code,
		Message:  err.Error(),
		FuncName: funcName,
	}
}

func Newf(code ErrCode, format string, args ...any) *Error {
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	funcName := "unknown"
	if fn != nil {
		funcName = fn.Name()
	}

	return &Error{
		Code:     code,
		Message:  fmt.Sprintf(format, args...),
		FuncName: funcName,
	}
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return nil, "", err
	}
	return data, "application/json", nil
}

func (e *Error) HTTPStatus() int {
	if s, ok := httpStatus[e.Code]; ok {
		return s
	}
	return http.StatusInternalServerError
}

// FieldError represents a validation error on a specific field.
type FieldError struct {
	Field string `json:"field"`
	Err   string `json:"error"`
}

// FieldErrors is a collection of field validation errors.
type FieldErrors []FieldError

func (fe FieldErrors) Error() string {
	return fmt.Sprintf("validation errors: %v", []FieldError(fe))
}

func (fe FieldErrors) Encode() ([]byte, string, error) {
	data, err := json.Marshal(struct {
		Code   ErrCode      `json:"code"`
		Errors []FieldError `json:"errors"`
	}{
		Code:   InvalidArgument,
		Errors: fe,
	})
	if err != nil {
		return nil, "", err
	}
	return data, "application/json", nil
}

func (fe FieldErrors) HTTPStatus() int {
	return http.StatusUnprocessableEntity
}

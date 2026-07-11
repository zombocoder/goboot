package runtime

import (
	"errors"
	"fmt"
	"net/http"
)

// CodedError is an application error that carries a stable machine-readable
// code (§23.2). The error handler surfaces the code in the Problem body.
type CodedError interface {
	error
	Code() string
}

// HTTPStatusError is an application error that maps itself to an HTTP status
// (§23.2). The error handler uses this status when rendering the Problem.
type HTTPStatusError interface {
	error
	HTTPStatus() int
}

// Error is a convenience implementation of both CodedError and HTTPStatusError,
// sufficient for most application errors without defining a bespoke type.
type Error struct {
	code    string
	status  int
	message string
	cause   error
}

// NewError builds an Error with the given HTTP status, code, and message.
func NewError(status int, code, message string) *Error {
	return &Error{status: status, code: code, message: message}
}

// Errorf builds an Error with a formatted message.
func Errorf(status int, code, format string, args ...any) *Error {
	return &Error{status: status, code: code, message: fmt.Sprintf(format, args...)}
}

// Wrap returns a copy of the error annotating an underlying cause, preserved for
// errors.Unwrap.
func (e *Error) Wrap(cause error) *Error {
	clone := *e
	clone.cause = cause
	return &clone
}

func (e *Error) Error() string {
	if e.cause != nil {
		return e.message + ": " + e.cause.Error()
	}
	return e.message
}

// Code returns the error's machine-readable code, satisfying CodedError.
func (e *Error) Code() string { return e.code }

// HTTPStatus returns the error's HTTP status, satisfying HTTPStatusError.
func (e *Error) HTTPStatus() int { return e.status }

// Unwrap exposes the wrapped cause.
func (e *Error) Unwrap() error { return e.cause }

// StatusOf reports the HTTP status an error should map to: an explicit
// HTTPStatusError status, otherwise 500 (§23.5).
func StatusOf(err error) int {
	var hse HTTPStatusError
	if errors.As(err, &hse) {
		if s := hse.HTTPStatus(); s != 0 {
			return s
		}
	}
	return http.StatusInternalServerError
}

// CodeOf reports an error's application code if it implements CodedError, else
// the empty string.
func CodeOf(err error) string {
	var ce CodedError
	if errors.As(err, &ce) {
		return ce.Code()
	}
	return ""
}

package runtime

import (
	"context"
	"net/http"
	"strings"
)

// Validator validates a bound request value before the controller runs (§20.1).
// The framework core depends only on this interface; adapters plug in concrete
// validators such as go-playground/validator.
type Validator interface {
	Validate(ctx context.Context, value any) error
}

// ValidationError reports one or more field-level validation failures. It
// implements HTTPStatusError (400) and CodedError so the default error handler
// renders it as a validation Problem (§20.4).
type ValidationError struct {
	Fields []FieldError
}

// NewValidationError builds a ValidationError from field failures.
func NewValidationError(fields ...FieldError) *ValidationError {
	return &ValidationError{Fields: fields}
}

func (e *ValidationError) Error() string {
	parts := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		parts[i] = f.Field + ": " + f.Message
	}
	return "validation failed: " + strings.Join(parts, "; ")
}

// Code satisfies CodedError.
func (e *ValidationError) Code() string { return "validation_error" }

// HTTPStatus satisfies HTTPStatusError.
func (e *ValidationError) HTTPStatus() int { return http.StatusBadRequest }

// NoopValidator accepts every value. It is the default until an adapter
// provides real validation, keeping generated handlers functional out of the
// box.
type NoopValidator struct{}

// Validate always returns nil.
func (NoopValidator) Validate(context.Context, any) error { return nil }

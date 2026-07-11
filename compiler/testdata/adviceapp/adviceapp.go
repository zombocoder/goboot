// Package adviceapp exercises @ControllerAdvice with @ExceptionHandler methods
// (§20): a response-form handler, a transform-form handler, and a catch-all.
package adviceapp

import (
	"context"

	"github.com/zombocoder/goboot/runtime"
)

// @Application(name="advice-app")
type Application struct{}

// NotFoundError is a domain error mapped to a 404 response body.
type NotFoundError struct{ Resource string }

func (e *NotFoundError) Error() string { return "not found: " + e.Resource }

// ConflictError is a domain error transformed into a coded runtime error.
type ConflictError struct{ Reason string }

func (e *ConflictError) Error() string { return "conflict: " + e.Reason }

// ErrorBody is the JSON body returned for a handled NotFoundError.
type ErrorBody struct {
	Message  string `json:"message"`
	Resource string `json:"resource"`
}

// Handler is the controller that raises the domain errors.
//
// @RestController
// @RequestMapping(path="/things")
type Handler struct{}

// NewHandler constructs a Handler.
func NewHandler() *Handler { return &Handler{} }

// LookupRequest binds the path id and a kind selector.
type LookupRequest struct {
	ID   string `path:"id"`
	Kind string `query:"kind"`
}

// Lookup raises a different error per kind so tests can drive each handler.
//
// @GetMapping(path="/{id}")
func (h *Handler) Lookup(ctx context.Context, req LookupRequest) (*ErrorBody, error) {
	switch req.Kind {
	case "missing":
		return nil, &NotFoundError{Resource: req.ID}
	case "conflict":
		return nil, &ConflictError{Reason: req.ID}
	default:
		return nil, context.DeadlineExceeded // an unmapped error hits the catch-all
	}
}

// Advice maps domain errors to responses.
//
// @ControllerAdvice
type Advice struct{}

// NewAdvice constructs the Advice.
func NewAdvice() *Advice { return &Advice{} }

// HandleNotFound renders a 404 body (response form).
//
// @ExceptionHandler
// @ResponseStatus(404)
func (a *Advice) HandleNotFound(ctx context.Context, err *NotFoundError) (*ErrorBody, error) {
	return &ErrorBody{Message: err.Error(), Resource: err.Resource}, nil
}

// HandleConflict transforms the error into a coded 409 (transform form); the
// delegate renders it as a Problem.
//
// @ExceptionHandler
func (a *Advice) HandleConflict(ctx context.Context, err *ConflictError) error {
	return runtime.NewError(409, "conflict", err.Error())
}

// HandleAny is the catch-all, transforming any other error into a coded 500.
//
// @ExceptionHandler
func (a *Advice) HandleAny(ctx context.Context, err error) error {
	return err
}

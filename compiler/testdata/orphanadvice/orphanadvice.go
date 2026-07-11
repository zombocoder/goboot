// Package orphanadvice has an @ExceptionHandler on a type that is not a
// @ControllerAdvice, which must raise GOBHTTP006.
package orphanadvice

import "context"

// @Application(name="orphan-advice")
type Application struct{}

// MyError is a domain error.
type MyError struct{}

func (MyError) Error() string { return "boom" }

// Stray is a plain component, not a @ControllerAdvice.
//
// @Component
type Stray struct{}

// NewStray constructs a Stray.
func NewStray() *Stray { return &Stray{} }

// Handle carries @ExceptionHandler but its receiver is not advice.
//
// @ExceptionHandler
func (s *Stray) Handle(ctx context.Context, err MyError) error {
	return err
}

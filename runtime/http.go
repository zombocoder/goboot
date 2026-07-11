package runtime

import (
	"context"
	"net/http"
)

// HTTPHandlerDependencies bundles the collaborators every generated handler
// needs (§22). Generated wiring builds one of these and passes it to each
// route's handler factory.
type HTTPHandlerDependencies struct {
	Binder         Binder
	Validator      Validator
	Authorizer     Authorizer
	ErrorHandler   ErrorHandler
	ResponseWriter ResponseWriter
	Observer       HTTPObserver
}

// DefaultHTTPHandlerDependencies returns dependencies wired with the built-in
// implementations, giving generated handlers a working default configuration.
func DefaultHTTPHandlerDependencies() HTTPHandlerDependencies {
	rw := JSONResponseWriter{}
	return HTTPHandlerDependencies{
		Binder:         DefaultBinder{},
		Validator:      NoopValidator{},
		Authorizer:     PermitAllAuthorizer{},
		ErrorHandler:   DefaultErrorHandler{Writer: rw},
		ResponseWriter: rw,
		Observer:       NoopObserver{},
	}
}

// Recover converts a panic in a handler into a 500 Problem response (§21). It is
// intended to be deferred directly — `defer runtime.Recover(ctx, w, r, deps)` —
// so that recover() runs in the deferred frame.
func Recover(ctx context.Context, w http.ResponseWriter, r *http.Request, eh ErrorHandler) {
	if rec := recover(); rec != nil {
		eh.Handle(ctx, w, r, Errorf(http.StatusInternalServerError, "internal_error", "panic: %v", rec))
	}
}

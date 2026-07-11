package runtime

import "context"

// HTTPRequestOperation identifies the handler an observation covers (§22).
type HTTPRequestOperation struct {
	Method     string
	Pattern    string
	Route      string
	Controller string
	Handler    string
}

// HTTPObserver begins an observation around a handler invocation (§22). The
// returned observation is ended with the final status and error.
type HTTPObserver interface {
	Begin(ctx context.Context, op HTTPRequestOperation) (context.Context, HTTPRequestObservation)
}

// HTTPRequestObservation is the in-flight observation for one request.
type HTTPRequestObservation interface {
	End(status int, err error)
}

// NoopObserver performs no observation and is the default.
type NoopObserver struct{}

// Begin returns the context unchanged and a no-op observation.
func (NoopObserver) Begin(ctx context.Context, _ HTTPRequestOperation) (context.Context, HTTPRequestObservation) {
	return ctx, noopObservation{}
}

type noopObservation struct{}

func (noopObservation) End(int, error) {}

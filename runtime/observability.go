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

// MethodLogger emits structured logging around an intercepted method (§35.3).
// Log is called just before the target runs and returns a function invoked with
// the method's error once it returns, mirroring the Tracer span idiom.
type MethodLogger interface {
	Log(ctx context.Context, method, level string) func(err error)
}

// NoopLogger logs nothing and is the default until a logging adapter is
// configured.
type NoopLogger struct{}

// Log returns a completion function that does nothing.
func (NoopLogger) Log(context.Context, string, string) func(error) {
	return func(error) {}
}

// AuditEvent describes a security-relevant action for the audit trail (§35.4).
// Method is the intercepted method; Action and Resource come from @Audit.
type AuditEvent struct {
	Method   string
	Action   string
	Resource string
}

// AuditSink records audit events, including whether the action failed (§35.4).
type AuditSink interface {
	Record(ctx context.Context, event AuditEvent, err error)
}

// NoopAuditSink records nothing and is the default.
type NoopAuditSink struct{}

func (NoopAuditSink) Record(context.Context, AuditEvent, error) {}

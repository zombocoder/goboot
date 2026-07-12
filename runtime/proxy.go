package runtime

import "context"

// Span is an in-flight trace span for an intercepted method (§35.1). End closes
// the span, recording any error the method returned.
type Span interface {
	End(err error)
}

// Tracer begins a span around an intercepted method (§24.4, §35.1).
type Tracer interface {
	Begin(ctx context.Context, name string) (context.Context, Span)
}

// NoopTracer performs no tracing and is the default until a tracing adapter is
// configured (OpenTelemetry is out of v0.1/v0.2 core scope, §54.2).
type NoopTracer struct{}

// Begin returns the context unchanged and a no-op span.
func (NoopTracer) Begin(ctx context.Context, _ string) (context.Context, Span) {
	return ctx, noopSpan{}
}

type noopSpan struct{}

func (noopSpan) End(error) {}

// MethodMetrics records the outcome of an intercepted method (§35.2). Method
// names are the only labels, so no high-cardinality argument values are
// recorded automatically (§35.2).
type MethodMetrics interface {
	RecordSuccess(method string)
	RecordFailure(method string)
}

// NoopMetrics records nothing and is the default.
type NoopMetrics struct{}

func (NoopMetrics) RecordSuccess(string) {}
func (NoopMetrics) RecordFailure(string) {}

// ProxyDependencies bundles the collaborators generated service proxies need
// (§24.1). Generated wiring builds one — defaulting to the no-op/direct
// implementations — and passes it to each proxy constructor.
type ProxyDependencies struct {
	Transactions TransactionManager
	Tracer       Tracer
	Metrics      MethodMetrics
	Authorizer   Authorizer
	Logger       MethodLogger
	Audit        AuditSink
	Breakers     CircuitBreakerProvider
	RateLimiters RateLimiterProvider
	Bulkheads    BulkheadProvider
	Cache        Cache
}

// DefaultProxyDependencies returns proxy dependencies wired with the built-in
// implementations, so generated proxies run out of the box.
func DefaultProxyDependencies() ProxyDependencies {
	return ProxyDependencies{
		Transactions: DirectTransactionManager{},
		Tracer:       NoopTracer{},
		Metrics:      NoopMetrics{},
		Authorizer:   PermitAllAuthorizer{},
		Logger:       NoopLogger{},
		Audit:        NoopAuditSink{},
		Breakers:     NewCircuitBreakerRegistry(),
		RateLimiters: NewRateLimiterRegistry(),
		Bulkheads:    NewBulkheadRegistry(),
		Cache:        NewMemoryCache(),
	}
}

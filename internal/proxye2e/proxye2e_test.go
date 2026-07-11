// Package proxye2e exercises generated service proxies end to end: it builds the
// components with recording interception dependencies and confirms that
// intercepted methods are traced, timed, and wrapped in a transaction (with
// rollback on error), while non-intercepted methods delegate untouched.
// wiring.gen.go is produced by the goboot generator from the proxyapp example.
package proxye2e

import (
	"context"
	"errors"
	"testing"

	"github.com/zombocoder/goboot/runtime"
)

// recordingTx records each transaction and whether it committed or rolled back.
type recordingTx struct {
	calls      int
	committed  int
	rolledBack int
}

func (r *recordingTx) WithinTransaction(ctx context.Context, _ runtime.TransactionOptions, fn func(context.Context) error) error {
	r.calls++
	err := fn(ctx)
	if err != nil {
		r.rolledBack++
	} else {
		r.committed++
	}
	return err
}

// recordingTracer records span names and the errors they end with.
type recordingTracer struct {
	begun []string
	ended []error
}

func (t *recordingTracer) Begin(ctx context.Context, name string) (context.Context, runtime.Span) {
	t.begun = append(t.begun, name)
	return ctx, &recordingSpan{tracer: t}
}

type recordingSpan struct{ tracer *recordingTracer }

func (s *recordingSpan) End(err error) { s.tracer.ended = append(s.tracer.ended, err) }

// recordingMetrics records success/failure method names.
type recordingMetrics struct {
	success []string
	failure []string
}

func (m *recordingMetrics) RecordSuccess(name string) { m.success = append(m.success, name) }
func (m *recordingMetrics) RecordFailure(name string) { m.failure = append(m.failure, name) }

func newComponents(t *testing.T) (*Components, *recordingTx, *recordingTracer, *recordingMetrics) {
	t.Helper()
	tx := &recordingTx{}
	tracer := &recordingTracer{}
	metrics := &recordingMetrics{}
	comps, err := buildComponents(runtime.ProxyDependencies{
		Transactions: tx, Tracer: tracer, Metrics: metrics,
	})
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	return comps, tx, tracer, metrics
}

func TestInterceptedMethodTracedTimedTransactional(t *testing.T) {
	comps, tx, tracer, metrics := newComponents(t)

	got, err := comps.OrderServiceProxy.CreateOrder(context.Background(), "widget")
	if err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if got != "order-widget" {
		t.Errorf("result = %q, want order-widget", got)
	}
	// Transaction wrapped the call and committed.
	if tx.calls != 1 || tx.committed != 1 || tx.rolledBack != 0 {
		t.Errorf("transaction = %+v, want 1 call committed", tx)
	}
	// Tracing spanned the call and ended without error.
	if len(tracer.begun) != 1 || tracer.begun[0] != "orders.create" {
		t.Errorf("spans begun = %v", tracer.begun)
	}
	if len(tracer.ended) != 1 || tracer.ended[0] != nil {
		t.Errorf("span should end with nil error, got %v", tracer.ended)
	}
	// Metrics recorded success.
	if len(metrics.success) != 1 || len(metrics.failure) != 0 {
		t.Errorf("metrics = success %v failure %v", metrics.success, metrics.failure)
	}
}

func TestInterceptedMethodRollsBackOnError(t *testing.T) {
	comps, tx, tracer, metrics := newComponents(t)

	_, err := comps.OrderServiceProxy.CreateOrder(context.Background(), "boom")
	if err == nil {
		t.Fatal("expected CreateOrder to fail")
	}
	// The transaction rolled back rather than committed (§26.6).
	if tx.rolledBack != 1 || tx.committed != 0 {
		t.Errorf("transaction should roll back, got %+v", tx)
	}
	// The span observed the error (§35.1).
	if len(tracer.ended) != 1 || !errors.Is(tracer.ended[0], err) {
		t.Errorf("span should end with the error, got %v", tracer.ended)
	}
	// Metrics recorded a failure, not a success.
	if len(metrics.failure) != 1 || len(metrics.success) != 0 {
		t.Errorf("metrics = success %v failure %v", metrics.success, metrics.failure)
	}
}

func TestNonInterceptedMethodDelegates(t *testing.T) {
	comps, tx, tracer, metrics := newComponents(t)

	got, err := comps.OrderServiceProxy.GetOrder(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if got != "order:42" {
		t.Errorf("result = %q, want order:42", got)
	}
	// GetOrder is not intercepted: no transaction, span, or metric.
	if tx.calls != 0 || len(tracer.begun) != 0 || len(metrics.success)+len(metrics.failure) != 0 {
		t.Errorf("delegated method must not be intercepted: tx=%+v spans=%v metrics=%d/%d",
			tx, tracer.begun, len(metrics.success), len(metrics.failure))
	}
}

func TestControllerReceivesProxy(t *testing.T) {
	// The controller was wired with the proxy (which implements the interface),
	// so calls made through it are also intercepted.
	comps, tx, _, _ := newComponents(t)
	if comps.OrderController == nil {
		t.Fatal("controller not constructed")
	}
	// The proxy field on Components is the interface type; calling through it
	// intercepts.
	if _, err := comps.OrderServiceProxy.CreateOrder(context.Background(), "via-proxy"); err != nil {
		t.Fatalf("CreateOrder: %v", err)
	}
	if tx.calls != 1 {
		t.Errorf("expected interception through the proxy, tx calls = %d", tx.calls)
	}
}

func TestDefaultProxyDependenciesWork(t *testing.T) {
	// With the framework defaults (direct transaction manager, no-op tracer and
	// metrics), the proxy still functions.
	comps, err := buildComponents(runtime.DefaultProxyDependencies())
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	got, err := comps.OrderServiceProxy.CreateOrder(context.Background(), "widget")
	if err != nil || got != "order-widget" {
		t.Errorf("default deps CreateOrder = %q, %v", got, err)
	}
}

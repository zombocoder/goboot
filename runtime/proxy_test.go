package runtime

import (
	"context"
	"errors"
	"testing"
)

func TestDirectTransactionManagerRunsCallback(t *testing.T) {
	ran := false
	err := DirectTransactionManager{}.WithinTransaction(context.Background(), TransactionOptions{}, func(context.Context) error {
		ran = true
		return nil
	})
	if err != nil || !ran {
		t.Fatalf("callback not run correctly: ran=%v err=%v", ran, err)
	}
}

func TestDirectTransactionManagerPropagatesError(t *testing.T) {
	sentinel := errors.New("rollback")
	err := DirectTransactionManager{}.WithinTransaction(context.Background(), TransactionOptions{}, func(context.Context) error {
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("error should propagate for rollback, got %v", err)
	}
}

func TestNoopTracerAndMetrics(t *testing.T) {
	ctx, span := NoopTracer{}.Begin(context.Background(), "Svc.M")
	if ctx == nil {
		t.Fatal("tracer should return a context")
	}
	span.End(nil) // must not panic
	m := NoopMetrics{}
	m.RecordSuccess("Svc.M")
	m.RecordFailure("Svc.M")
}

func TestDefaultProxyDependencies(t *testing.T) {
	deps := DefaultProxyDependencies()
	if deps.Transactions == nil || deps.Tracer == nil || deps.Metrics == nil {
		t.Fatal("default proxy dependencies must be non-nil")
	}
	// The default transaction manager runs the callback.
	if err := deps.Transactions.WithinTransaction(context.Background(), TransactionOptions{}, func(context.Context) error { return nil }); err != nil {
		t.Errorf("default transaction manager failed: %v", err)
	}
}

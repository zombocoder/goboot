package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func fail(context.Context) error    { return errors.New("boom") }
func succeed(context.Context) error { return nil }

func TestCircuitBreakerLifecycle(t *testing.T) {
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }
	cb := newCircuitBreaker(CircuitBreakerSpec{Name: "c", FailureThreshold: 2, ResetTimeout: time.Minute}, clock)
	ctx := context.Background()

	// Two failures trip the breaker open.
	if err := cb.Execute(ctx, fail); err == nil {
		t.Fatal("call 1 should surface the target error")
	}
	if err := cb.Execute(ctx, fail); err == nil {
		t.Fatal("call 2 should surface the target error")
	}
	// Now open: fails fast without calling the target.
	ran := false
	err := cb.Execute(ctx, func(context.Context) error { ran = true; return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("open breaker should return ErrCircuitOpen, got %v", err)
	}
	if ran {
		t.Fatal("open breaker must not invoke the target")
	}

	// Before the reset timeout, still open.
	now = now.Add(30 * time.Second)
	if err := cb.Execute(ctx, succeed); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("still-open breaker should reject, got %v", err)
	}

	// After the reset timeout, a probe is allowed; success closes the breaker.
	now = now.Add(31 * time.Second)
	if err := cb.Execute(ctx, succeed); err != nil {
		t.Fatalf("half-open probe should run: %v", err)
	}
	// Closed again: a fresh failure count is required to re-trip.
	if err := cb.Execute(ctx, fail); err == nil {
		t.Fatal("expected target error after close")
	}
	if err := cb.Execute(ctx, succeed); err != nil {
		t.Fatalf("single failure should not have re-tripped: %v", err)
	}
}

func TestCircuitBreakerHalfOpenFailureReopens(t *testing.T) {
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }
	cb := newCircuitBreaker(CircuitBreakerSpec{Name: "c", FailureThreshold: 1, ResetTimeout: time.Second}, clock)
	ctx := context.Background()

	if err := cb.Execute(ctx, fail); err == nil {
		t.Fatal("failure should surface")
	}
	now = now.Add(2 * time.Second) // allow a probe
	// The probe fails, which must re-open the breaker.
	if err := cb.Execute(ctx, fail); err == nil {
		t.Fatal("probe failure should surface")
	}
	if err := cb.Execute(ctx, succeed); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("failed probe should re-open the breaker, got %v", err)
	}
}

func TestTokenBucketRefill(t *testing.T) {
	now := time.Unix(0, 0)
	clock := func() time.Time { return now }
	tb := newTokenBucket(RateLimitSpec{Name: "r", Limit: 2, Period: time.Second}, clock)
	ctx := context.Background()

	// Burst defaults to Limit=2: two calls pass, third is throttled.
	if err := tb.Execute(ctx, succeed); err != nil {
		t.Fatalf("call 1: %v", err)
	}
	if err := tb.Execute(ctx, succeed); err != nil {
		t.Fatalf("call 2: %v", err)
	}
	if err := tb.Execute(ctx, succeed); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("call 3 should be rate limited, got %v", err)
	}

	// After half a period, one token has refilled (2 per second → 1 in 500ms).
	now = now.Add(500 * time.Millisecond)
	if err := tb.Execute(ctx, succeed); err != nil {
		t.Fatalf("call after refill should pass: %v", err)
	}
	if err := tb.Execute(ctx, succeed); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("bucket should be empty again, got %v", err)
	}
}

func TestBulkheadWaitTimeout(t *testing.T) {
	bh := NewBulkhead(BulkheadSpec{Name: "b", MaxConcurrent: 1, MaxWait: 20 * time.Millisecond})
	ctx := context.Background()
	held := make(chan struct{})
	release := make(chan struct{})
	go func() {
		_ = bh.Execute(ctx, func(context.Context) error {
			close(held)
			<-release
			return nil
		})
	}()
	<-held
	// The slot is held; a waiter times out after MaxWait and is rejected.
	if err := bh.Execute(ctx, succeed); !errors.Is(err, ErrBulkheadFull) {
		t.Fatalf("waiter should time out with ErrBulkheadFull, got %v", err)
	}
	close(release)
}

func TestRegistriesCacheByName(t *testing.T) {
	cbReg := NewCircuitBreakerRegistry()
	if cbReg.CircuitBreaker(CircuitBreakerSpec{Name: "x"}) != cbReg.CircuitBreaker(CircuitBreakerSpec{Name: "x"}) {
		t.Error("circuit breaker registry should return the same instance per name")
	}
	rlReg := NewRateLimiterRegistry()
	if rlReg.RateLimiter(RateLimitSpec{Name: "x"}) != rlReg.RateLimiter(RateLimitSpec{Name: "x"}) {
		t.Error("rate limiter registry should return the same instance per name")
	}
	bhReg := NewBulkheadRegistry()
	if bhReg.Bulkhead(BulkheadSpec{Name: "x"}) != bhReg.Bulkhead(BulkheadSpec{Name: "x"}) {
		t.Error("bulkhead registry should return the same instance per name")
	}
}

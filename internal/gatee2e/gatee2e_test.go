// Package gatee2e drives generated @CircuitBreaker / @RateLimit / @Bulkhead
// proxies to confirm the resilience gates admit and reject calls as configured.
// wiring.gen.go is produced by the goboot generator from the gateapp example.
package gatee2e

import (
	"context"
	"errors"
	"sync"
	"testing"

	goruntime "github.com/zombocoder/goboot/runtime"
)

func newComps(t *testing.T) *Components {
	t.Helper()
	comps, err := buildComponents(goruntime.DefaultProxyDependencies())
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	return comps
}

func TestCircuitBreakerTripsOpen(t *testing.T) {
	comps := newComps(t)
	comps.Downstream.SetFail(true)
	ctx := context.Background()

	// failureThreshold=2: two failures trip the breaker.
	if err := comps.DownstreamServiceProxy.Call(ctx); err == nil {
		t.Fatal("call 1 should fail")
	}
	if err := comps.DownstreamServiceProxy.Call(ctx); err == nil {
		t.Fatal("call 2 should fail")
	}
	callsBefore := comps.Downstream.Calls()

	// The breaker is now open: the next call fails fast without hitting target.
	err := comps.DownstreamServiceProxy.Call(ctx)
	if !errors.Is(err, goruntime.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if comps.Downstream.Calls() != callsBefore {
		t.Errorf("open circuit must not invoke the target (calls %d -> %d)", callsBefore, comps.Downstream.Calls())
	}
}

func TestRateLimitRejectsOverflow(t *testing.T) {
	comps := newComps(t)
	ctx := context.Background()

	// limit=2 per second, burst defaults to limit: first two pass.
	for i := 0; i < 2; i++ {
		if _, err := comps.DownstreamServiceProxy.Fetch(ctx); err != nil {
			t.Fatalf("fetch %d should pass: %v", i, err)
		}
	}
	// The third within the same window is throttled.
	if _, err := comps.DownstreamServiceProxy.Fetch(ctx); !errors.Is(err, goruntime.ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited on overflow, got %v", err)
	}
}

func TestBulkheadRejectsWhenFull(t *testing.T) {
	comps := newComps(t)
	ctx := context.Background()

	// A no-op call returns immediately, so a fresh slot is always free.
	if err := comps.DownstreamServiceProxy.Bounded(ctx); err != nil {
		t.Fatalf("bounded call should pass when idle: %v", err)
	}

	// maxConcurrent=1: hold the slot from a real bulkhead and confirm a second
	// concurrent Execute is rejected immediately (maxWait defaults to 0).
	bh := goruntime.NewBulkhead(goruntime.BulkheadSpec{Name: "t", MaxConcurrent: 1})
	held := make(chan struct{})
	release := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = bh.Execute(ctx, func(context.Context) error {
			close(held) // signal the slot is now taken
			<-release   // hold it until the test releases
			return nil
		})
	}()
	<-held // the slot is definitely held now
	if err := bh.Execute(ctx, func(context.Context) error { return nil }); !errors.Is(err, goruntime.ErrBulkheadFull) {
		t.Fatalf("expected ErrBulkheadFull when the slot is held, got %v", err)
	}
	close(release)
	wg.Wait()
}

func TestGatesPermitByDefault(t *testing.T) {
	comps := newComps(t)
	ctx := context.Background()
	// A single call through each gate succeeds with the default registries.
	if err := comps.DownstreamServiceProxy.Call(ctx); err != nil {
		t.Errorf("call: %v", err)
	}
	if _, err := comps.DownstreamServiceProxy.Fetch(ctx); err != nil {
		t.Errorf("fetch: %v", err)
	}
	if err := comps.DownstreamServiceProxy.Bounded(ctx); err != nil {
		t.Errorf("bounded: %v", err)
	}
}

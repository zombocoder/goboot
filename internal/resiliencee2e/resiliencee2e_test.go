// Package resiliencee2e drives generated @Retry and @Timeout proxies to confirm
// the interceptors behave: a flaky method is retried until it succeeds, and a
// slow method is bounded by its timeout. wiring.gen.go is produced by the goboot
// generator from the resilienceapp example.
package resiliencee2e

import (
	"context"
	"errors"
	"testing"
	"time"

	goruntime "github.com/zombocoder/goboot/runtime"
)

// newComps builds the components with the default (direct) proxy dependencies.
// comps.Flaky is the concrete target; comps.FlakyServiceProxy is the intercepted
// interface.
func newComps(t *testing.T) *Components {
	t.Helper()
	comps, err := buildComponents(goruntime.DefaultProxyDependencies())
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	return comps
}

func TestRetryRetriesUntilSuccess(t *testing.T) {
	comps := newComps(t)
	// The target fails twice, then succeeds; @Retry(maxAttempts=4) must recover.
	comps.Flaky.SetFailures(2)

	got, err := comps.FlakyServiceProxy.Attempt(context.Background())
	if err != nil {
		t.Fatalf("Attempt should have succeeded after retries: %v", err)
	}
	if comps.Flaky.Calls() != 3 {
		t.Errorf("expected 3 attempts (2 failures + 1 success), got %d", comps.Flaky.Calls())
	}
	if got != 3 {
		t.Errorf("Attempt returned %d, want 3", got)
	}
}

func TestRetryExhausts(t *testing.T) {
	comps := newComps(t)
	comps.Flaky.SetFailures(10) // always fails within the 4 attempts

	_, err := comps.FlakyServiceProxy.Attempt(context.Background())
	if err == nil {
		t.Fatal("Attempt should fail after exhausting retries")
	}
	if comps.Flaky.Calls() != 4 {
		t.Errorf("expected 4 attempts, got %d", comps.Flaky.Calls())
	}
}

func TestTimeoutBoundsSlowCall(t *testing.T) {
	comps := newComps(t)
	start := time.Now()
	err := comps.FlakyServiceProxy.Slow(context.Background())
	elapsed := time.Since(start)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Slow should return a deadline-exceeded error, got %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout did not bound the call; took %v", elapsed)
	}
}

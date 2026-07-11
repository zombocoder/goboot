package compiler

import (
	"testing"
	"time"
)

func TestResilienceGateDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/gateapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	svc := componentByName(res.App, "downstream")
	if svc == nil || !svc.Proxied {
		t.Fatal("downstream service should be proxied")
	}
	byName := map[string]int{}
	for i, m := range svc.Intercepted {
		byName[m.Name] = i
	}

	call := svc.Intercepted[byName["Call"]]
	if call.CircuitBreaker == nil {
		t.Fatal("Call should have a circuit breaker")
	}
	if call.CircuitBreaker.Name != "downstream" || call.CircuitBreaker.FailureThreshold != 2 ||
		call.CircuitBreaker.ResetTimeout != 50*time.Millisecond {
		t.Errorf("circuit breaker spec = %+v", call.CircuitBreaker)
	}

	fetch := svc.Intercepted[byName["Fetch"]]
	if fetch.RateLimit == nil || fetch.RateLimit.Limit != 2 || fetch.RateLimit.Period != time.Second {
		t.Errorf("rate limit spec = %+v", fetch.RateLimit)
	}
	// Name defaults to Type.Method when not given.
	if fetch.RateLimit.Name != "DownstreamService.Fetch" {
		t.Errorf("rate limit default name = %q", fetch.RateLimit.Name)
	}

	bounded := svc.Intercepted[byName["Bounded"]]
	if bounded.Bulkhead == nil || bounded.Bulkhead.MaxConcurrent != 1 {
		t.Errorf("bulkhead spec = %+v", bounded.Bulkhead)
	}
}

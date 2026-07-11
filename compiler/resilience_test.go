package compiler

import (
	"testing"
	"time"
)

func TestResilienceDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/resilienceapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	target := componentByName(res.App, "flaky")
	if target == nil || !target.Proxied {
		t.Fatal("flaky service should be proxied")
	}

	byName := map[string]int{}
	for i, m := range target.Intercepted {
		byName[m.Name] = i
	}

	attempt := target.Intercepted[byName["Attempt"]]
	if attempt.Retry == nil {
		t.Fatal("Attempt should have a retry policy")
	}
	if attempt.Retry.MaxAttempts != 4 || attempt.Retry.Delay != time.Millisecond ||
		attempt.Retry.Multiplier != 2.0 || attempt.Retry.MaxDelay != 10*time.Millisecond {
		t.Errorf("retry policy = %+v", attempt.Retry)
	}

	slow := target.Intercepted[byName["Slow"]]
	if slow.Timeout != 20*time.Millisecond {
		t.Errorf("Slow timeout = %v, want 20ms", slow.Timeout)
	}
}

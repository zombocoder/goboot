package prometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCountsByOutcome(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)

	m.RecordSuccess("Svc.Do")
	m.RecordSuccess("Svc.Do")
	m.RecordFailure("Svc.Do")
	m.RecordSuccess("Svc.Other")

	if got := testutil.ToFloat64(m.calls.WithLabelValues("Svc.Do", "success")); got != 2 {
		t.Errorf("Svc.Do success = %v, want 2", got)
	}
	if got := testutil.ToFloat64(m.calls.WithLabelValues("Svc.Do", "failure")); got != 1 {
		t.Errorf("Svc.Do failure = %v, want 1", got)
	}
	if got := testutil.ToFloat64(m.calls.WithLabelValues("Svc.Other", "success")); got != 1 {
		t.Errorf("Svc.Other success = %v, want 1", got)
	}
}

func TestRegisteredSeriesName(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewMetrics(reg)
	m.RecordSuccess("X.Y")

	// The counter is registered under the expected fully qualified name.
	if n := testutil.CollectAndCount(m.calls); n != 1 {
		t.Errorf("collected series count = %d, want 1", n)
	}
	families, err := reg.Gather()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range families {
		if f.GetName() == "goboot_method_calls_total" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected metric goboot_method_calls_total to be registered")
	}
}

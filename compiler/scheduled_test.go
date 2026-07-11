package compiler

import (
	"testing"
	"time"
)

func TestScheduledDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/schedapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	reporter := componentByName(res.App, "reporter")
	if reporter == nil {
		t.Fatal("reporter not found")
	}
	if len(reporter.Scheduled) != 2 {
		t.Fatalf("scheduled methods = %d, want 2", len(reporter.Scheduled))
	}

	byName := map[string]int{}
	for i, m := range reporter.Scheduled {
		byName[m.MethodName] = i
	}

	poll := reporter.Scheduled[byName["Poll"]]
	if poll.Interval != 2*time.Minute {
		t.Errorf("Poll interval = %v, want 2m (fixedRate=2, MINUTES)", poll.Interval)
	}
	if !poll.TakesContext || !poll.ReturnsError {
		t.Errorf("Poll signature flags = ctx %v err %v", poll.TakesContext, poll.ReturnsError)
	}

	hb := reporter.Scheduled[byName["Heartbeat"]]
	if hb.Interval != 30*time.Second {
		t.Errorf("Heartbeat interval = %v, want 30s", hb.Interval)
	}
	if hb.InitialDelay != 5*time.Second {
		t.Errorf("Heartbeat initialDelay = %v, want 5s", hb.InitialDelay)
	}
	if hb.TakesContext || hb.ReturnsError {
		t.Errorf("Heartbeat signature flags = ctx %v err %v", hb.TakesContext, hb.ReturnsError)
	}
}

func TestInvalidScheduleRejected(t *testing.T) {
	res := analyzeApp(t, "./testdata/badsched")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeInvalidSchedule || d.Code == CodeInvalidScheduled {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a scheduling diagnostic, got %v", res.Diagnostics)
	}
}

func TestTimeUnitDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"":             time.Millisecond,
		"MILLISECONDS": time.Millisecond,
		"SECONDS":      time.Second,
		"MINUTES":      time.Minute,
		"HOURS":        time.Hour,
		"DAYS":         24 * time.Hour,
		"bogus":        time.Millisecond,
	}
	for unit, want := range cases {
		if got := timeUnitDuration(unit); got != want {
			t.Errorf("timeUnitDuration(%q) = %v, want %v", unit, got, want)
		}
	}
}

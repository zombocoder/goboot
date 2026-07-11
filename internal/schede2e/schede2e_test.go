// Package schede2e exercises generated @Scheduled wiring end to end: it builds
// the components, builds the scheduler from the generated buildScheduler, runs
// it, and confirms the component's scheduled method actually fires and stops.
// wiring.gen.go is produced by the goboot generator from the schedtick example.
package schede2e

import (
	"context"
	"testing"
	"time"
)

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

func TestGeneratedScheduledTaskFires(t *testing.T) {
	comps, err := buildComponents()
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	sched := buildScheduler(comps)

	if comps.Ticker.Count() != 0 {
		t.Fatal("ticker should not have fired before the scheduler starts")
	}
	sched.Start(context.Background())
	waitFor(t, func() bool { return comps.Ticker.Count() >= 3 })
	sched.Stop()

	stopped := comps.Ticker.Count()
	time.Sleep(20 * time.Millisecond)
	if comps.Ticker.Count() != stopped {
		t.Errorf("scheduled task kept firing after Stop: %d -> %d", stopped, comps.Ticker.Count())
	}
}

func TestNewApplicationRunsScheduledTask(t *testing.T) {
	app, err := NewApplication()
	if err != nil {
		t.Fatalf("NewApplication: %v", err)
	}
	if app.Scheduler == nil {
		t.Fatal("application should have a scheduler")
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()
	// The scheduler runs the task; give it time, then shut down.
	time.Sleep(40 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}
}

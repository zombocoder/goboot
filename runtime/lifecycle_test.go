package runtime

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"
)

func TestLifecycleStartStopOrder(t *testing.T) {
	var events []string
	lc := NewLifecycle(time.Second)
	for _, name := range []string{"A", "B", "C"} {
		n := name
		lc.Register(n,
			func(context.Context) error { events = append(events, "start "+n); return nil },
			func(context.Context) error { events = append(events, "stop "+n); return nil },
		)
	}
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	want := []string{"start A", "start B", "start C", "stop C", "stop B", "stop A"}
	if len(events) != len(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q (full: %v)", i, events[i], want[i], events)
		}
	}
}

func TestLifecycleStartupRollback(t *testing.T) {
	var events []string
	lc := NewLifecycle(time.Second)
	lc.Register("A",
		func(context.Context) error { events = append(events, "start A"); return nil },
		func(context.Context) error { events = append(events, "stop A"); return nil },
	)
	lc.Register("B",
		func(context.Context) error { events = append(events, "start B"); return nil },
		func(context.Context) error { events = append(events, "stop B"); return nil },
	)
	lc.Register("C",
		func(context.Context) error { events = append(events, "start C"); return errors.New("boom") },
		func(context.Context) error { events = append(events, "stop C"); return nil },
	)
	err := lc.Start(context.Background())
	if err == nil {
		t.Fatal("expected startup error")
	}
	// A and B started; C failed. Rollback stops B then A. C's stop must NOT run
	// (it never successfully initialized).
	want := []string{"start A", "start B", "start C", "stop B", "stop A"}
	if len(events) != len(want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Fatalf("event[%d] = %q, want %q (full: %v)", i, events[i], want[i], events)
		}
	}
}

func TestLifecycleNilHooks(t *testing.T) {
	var stopped []string
	lc := NewLifecycle(time.Second)
	lc.Register("no-hooks", nil, nil)
	lc.Register("stop-only", nil, func(context.Context) error {
		stopped = append(stopped, "stop-only")
		return nil
	})
	if err := lc.Start(context.Background()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := lc.Stop(context.Background()); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if len(stopped) != 1 || stopped[0] != "stop-only" {
		t.Errorf("stop hooks = %v", stopped)
	}
}

func TestLifecycleStopJoinsErrors(t *testing.T) {
	lc := NewLifecycle(time.Second)
	lc.Register("A", nil, func(context.Context) error { return errors.New("a failed") })
	lc.Register("B", nil, func(context.Context) error { return errors.New("b failed") })
	_ = lc.Start(context.Background())
	err := lc.Stop(context.Background())
	if err == nil {
		t.Fatal("expected joined stop errors")
	}
}

func TestDefaultShutdownTimeout(t *testing.T) {
	lc := NewLifecycle(0)
	if lc.ShutdownTimeout() != DefaultShutdownTimeout {
		t.Errorf("timeout = %v, want %v", lc.ShutdownTimeout(), DefaultShutdownTimeout)
	}
}

func TestApplicationRunGracefulShutdown(t *testing.T) {
	rec := &recorder{}
	lc := NewLifecycle(2 * time.Second)
	lc.Register("svc",
		func(context.Context) error { rec.add("start"); return nil },
		func(context.Context) error { rec.add("stop"); return nil },
	)
	server := &http.Server{Addr: "127.0.0.1:0", Handler: http.NewServeMux()}
	app := &Application{Server: server, Lifecycle: lc}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()

	// Give Run a moment to start the lifecycle, then cancel for graceful stop.
	waitFor(t, func() bool { return rec.len() >= 1 })
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}
	events := rec.snapshot()
	if len(events) < 2 || events[0] != "start" || events[len(events)-1] != "stop" {
		t.Errorf("lifecycle events = %v, want start...stop", events)
	}
}

// recorder is a goroutine-safe event log for lifecycle tests.
type recorder struct {
	mu     sync.Mutex
	events []string
}

func (r *recorder) add(e string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, e)
}

func (r *recorder) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events)
}

func (r *recorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

func TestApplicationRunLifecycleStartFailure(t *testing.T) {
	lc := NewLifecycle(time.Second)
	lc.Register("bad", func(context.Context) error { return errors.New("nope") }, nil)
	app := &Application{Lifecycle: lc}
	if err := app.Run(context.Background()); err == nil {
		t.Fatal("expected Run to fail when lifecycle startup fails")
	}
}

// waitFor polls cond until true or the test times out.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

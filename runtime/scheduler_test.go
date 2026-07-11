package runtime

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerRunsTaskRepeatedly(t *testing.T) {
	var count int32
	s := NewScheduler()
	s.Register(ScheduledTask{
		Name:     "tick",
		Interval: 5 * time.Millisecond,
		Run: func(context.Context) error {
			atomic.AddInt32(&count, 1)
			return nil
		},
	})
	s.Start(context.Background())
	waitFor(t, func() bool { return atomic.LoadInt32(&count) >= 3 })
	s.Stop()

	// After Stop, the count must not keep growing.
	stopped := atomic.LoadInt32(&count)
	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(&count) != stopped {
		t.Errorf("task kept running after Stop: %d -> %d", stopped, atomic.LoadInt32(&count))
	}
}

func TestSchedulerStopWaitsForInFlight(t *testing.T) {
	var running, finished int32
	s := NewScheduler()
	s.Register(ScheduledTask{
		Name:     "slow",
		Interval: 5 * time.Millisecond,
		Run: func(ctx context.Context) error {
			atomic.StoreInt32(&running, 1)
			time.Sleep(15 * time.Millisecond)
			atomic.AddInt32(&finished, 1)
			return nil
		},
	})
	s.Start(context.Background())
	waitFor(t, func() bool { return atomic.LoadInt32(&running) == 1 })
	s.Stop() // must block until the in-flight run finishes
	if atomic.LoadInt32(&finished) == 0 {
		t.Error("Stop should wait for an in-flight task to finish")
	}
}

func TestSchedulerInitialDelay(t *testing.T) {
	var ran int32
	s := NewScheduler()
	s.Register(ScheduledTask{
		Name:         "delayed",
		Interval:     5 * time.Millisecond,
		InitialDelay: 50 * time.Millisecond,
		Run:          func(context.Context) error { atomic.AddInt32(&ran, 1); return nil },
	})
	s.Start(context.Background())
	defer s.Stop()
	// Before the initial delay elapses, the task must not have run.
	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(&ran) != 0 {
		t.Error("task ran before its initial delay")
	}
	waitFor(t, func() bool { return atomic.LoadInt32(&ran) >= 1 })
}

func TestSchedulerSkipsNonPositiveInterval(t *testing.T) {
	var ran int32
	s := NewScheduler()
	s.Register(ScheduledTask{Name: "disabled", Interval: 0, Run: func(context.Context) error {
		atomic.AddInt32(&ran, 1)
		return nil
	}})
	s.Start(context.Background())
	time.Sleep(20 * time.Millisecond)
	s.Stop()
	if atomic.LoadInt32(&ran) != 0 {
		t.Error("a task with a non-positive interval must not run")
	}
}

func TestSchedulerErrorHandler(t *testing.T) {
	var mu sync.Mutex
	var errs []string
	s := NewScheduler()
	s.OnError(func(name string, err error) {
		mu.Lock()
		errs = append(errs, name+":"+err.Error())
		mu.Unlock()
	})
	s.Register(ScheduledTask{Name: "boom", Interval: 5 * time.Millisecond,
		Run: func(context.Context) error { return errors.New("failed") }})
	s.Start(context.Background())
	waitFor(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(errs) >= 1
	})
	s.Stop()
	mu.Lock()
	defer mu.Unlock()
	if len(errs) == 0 || errs[0] != "boom:failed" {
		t.Errorf("error handler not invoked correctly: %v", errs)
	}
}

func TestSchedulerStopWithoutStart(t *testing.T) {
	// Stop on a never-started scheduler must not panic or block.
	NewScheduler().Stop()
}

func TestApplicationRunsScheduler(t *testing.T) {
	var count int32
	sched := NewScheduler()
	sched.Register(ScheduledTask{Name: "app", Interval: 5 * time.Millisecond,
		Run: func(context.Context) error { atomic.AddInt32(&count, 1); return nil }})
	app := &Application{Scheduler: sched, Lifecycle: NewLifecycle(time.Second)}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()

	waitFor(t, func() bool { return atomic.LoadInt32(&count) >= 2 })
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancellation")
	}
	// Scheduler was stopped by Shutdown.
	stopped := atomic.LoadInt32(&count)
	time.Sleep(20 * time.Millisecond)
	if atomic.LoadInt32(&count) != stopped {
		t.Error("scheduler kept running after application shutdown")
	}
}

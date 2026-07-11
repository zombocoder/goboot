package runtime

import (
	"context"
	"sync"
	"time"
)

// ScheduledFunc is a scheduled task's callback (§4.2 background workers). The
// generator adapts every supported @Scheduled method signature to this shape.
type ScheduledFunc func(ctx context.Context) error

// ScheduledTask is one periodic task registered with a Scheduler.
type ScheduledTask struct {
	// Name identifies the task for diagnostics and error reporting.
	Name string
	// Interval is the fixed rate between runs. A non-positive interval disables
	// the task.
	Interval time.Duration
	// InitialDelay optionally delays the first run.
	InitialDelay time.Duration
	// Run is the task body.
	Run ScheduledFunc
}

// Scheduler runs registered tasks on background goroutines at a fixed rate. It
// starts with the application and stops gracefully on shutdown, cancelling the
// task context and waiting for in-flight runs to return.
type Scheduler struct {
	tasks   []ScheduledTask
	onError func(name string, err error)

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewScheduler returns an empty scheduler.
func NewScheduler() *Scheduler { return &Scheduler{} }

// Register adds a task.
func (s *Scheduler) Register(task ScheduledTask) {
	s.tasks = append(s.tasks, task)
}

// OnError installs a handler invoked when a task returns an error. Without one,
// task errors are ignored.
func (s *Scheduler) OnError(handler func(name string, err error)) {
	s.onError = handler
}

// Tasks returns the registered tasks.
func (s *Scheduler) Tasks() []ScheduledTask { return s.tasks }

// Start launches a goroutine per task. It derives a cancellable context from
// ctx; Stop cancels it. Start is safe to call once; a task with a non-positive
// interval is skipped.
func (s *Scheduler) Start(ctx context.Context) {
	s.mu.Lock()
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	for _, task := range s.tasks {
		if task.Interval <= 0 || task.Run == nil {
			continue
		}
		s.wg.Add(1)
		go s.runTask(runCtx, task)
	}
}

// runTask executes a single task on its schedule until the context is cancelled.
func (s *Scheduler) runTask(ctx context.Context, task ScheduledTask) {
	defer s.wg.Done()

	if task.InitialDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(task.InitialDelay):
		}
	}

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := task.Run(ctx); err != nil && s.onError != nil {
				s.onError(task.Name, err)
			}
		}
	}
}

// Stop cancels the task context and waits for all task goroutines to finish. It
// is safe to call when Start was never called or when the scheduler has no
// tasks.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
}

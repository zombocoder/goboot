package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// DefaultShutdownTimeout is the global default graceful-shutdown budget (§30.6).
const DefaultShutdownTimeout = 30 * time.Second

// LifecycleHook runs a component's @PostConstruct or @PreDestroy method (§30.1).
// The generator adapts every supported hook signature (§30.2) to this shape.
type LifecycleHook func(ctx context.Context) error

// lifecycleComponent pairs a component's optional start and stop hooks with a
// name for diagnostics.
type lifecycleComponent struct {
	name  string
	start LifecycleHook
	stop  LifecycleHook
}

// Lifecycle runs component startup and shutdown hooks in a well-defined order:
// startup in construction order, shutdown in reverse (§30.3, §30.5). A startup
// failure rolls back the components already started (§30.4).
type Lifecycle struct {
	components      []lifecycleComponent
	started         []lifecycleComponent // initialized components, in order
	shutdownTimeout time.Duration
}

// NewLifecycle creates a Lifecycle with the given graceful-shutdown timeout; a
// non-positive timeout selects DefaultShutdownTimeout.
func NewLifecycle(shutdownTimeout time.Duration) *Lifecycle {
	if shutdownTimeout <= 0 {
		shutdownTimeout = DefaultShutdownTimeout
	}
	return &Lifecycle{shutdownTimeout: shutdownTimeout}
}

// Register adds a component's hooks in construction order. Either hook may be
// nil. Components must be registered in the same order they were constructed so
// that startup and shutdown ordering are correct.
func (l *Lifecycle) Register(name string, start, stop LifecycleHook) {
	l.components = append(l.components, lifecycleComponent{name: name, start: start, stop: stop})
}

// ShutdownTimeout returns the configured graceful-shutdown budget.
func (l *Lifecycle) ShutdownTimeout() time.Duration { return l.shutdownTimeout }

// Start runs every component's start hook in construction order. If one fails,
// it rolls back the components already initialized (in reverse) and returns the
// startup error (§30.4). A component is considered initialized once its start
// hook succeeds (or it has none), so its stop hook will run during rollback or
// shutdown.
func (l *Lifecycle) Start(ctx context.Context) error {
	for _, c := range l.components {
		if c.start != nil {
			if err := c.start(ctx); err != nil {
				rollbackErr := l.stopStarted(context.WithoutCancel(ctx))
				if rollbackErr != nil {
					return fmt.Errorf("start %s: %w (rollback: %v)", c.name, err, rollbackErr)
				}
				return fmt.Errorf("start %s: %w", c.name, err)
			}
		}
		l.started = append(l.started, c)
	}
	return nil
}

// Stop runs the stop hooks of all started components in reverse construction
// order, bounded by the shutdown timeout (§30.5). It attempts every hook and
// returns the joined errors.
func (l *Lifecycle) Stop(ctx context.Context) error {
	stopCtx, cancel := context.WithTimeout(ctx, l.shutdownTimeout)
	defer cancel()
	return l.stopStarted(stopCtx)
}

// stopStarted runs stop hooks in reverse order, clearing the started list.
func (l *Lifecycle) stopStarted(ctx context.Context) error {
	var errs []error
	for i := len(l.started) - 1; i >= 0; i-- {
		c := l.started[i]
		if c.stop == nil {
			continue
		}
		if err := c.stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop %s: %w", c.name, err))
		}
	}
	l.started = nil
	return errors.Join(errs...)
}

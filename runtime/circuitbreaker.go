package runtime

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrCircuitOpen is returned by a circuit breaker that is rejecting calls
// because it has tripped open (§36.3).
var ErrCircuitOpen = errors.New("goboot: circuit breaker open")

// CircuitBreakerSpec configures a named circuit breaker from @CircuitBreaker
// arguments (§36.3). Zero fields fall back to the defaults in NewCircuitBreaker.
type CircuitBreakerSpec struct {
	Name string
	// FailureThreshold is the number of consecutive failures that trips the
	// breaker open (default 5).
	FailureThreshold int
	// ResetTimeout is how long the breaker stays open before allowing a probe
	// (default 30s).
	ResetTimeout time.Duration
	// HalfOpenMax is the number of probe calls allowed while half-open; a
	// success closes the breaker, a failure re-opens it (default 1).
	HalfOpenMax int
}

// CircuitBreaker guards a call, failing fast with ErrCircuitOpen while open
// (§36.3).
type CircuitBreaker interface {
	// Execute runs fn when the circuit permits it and records the outcome,
	// returning ErrCircuitOpen (without calling fn) when the circuit is open.
	Execute(ctx context.Context, fn func(context.Context) error) error
}

// CircuitBreakerProvider resolves a circuit breaker for a spec, caching by name
// so repeated calls share state (§36.3). Adapters may supply their own.
type CircuitBreakerProvider interface {
	CircuitBreaker(spec CircuitBreakerSpec) CircuitBreaker
}

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

// circuitBreaker is the built-in in-memory breaker. now is injectable for tests.
type circuitBreaker struct {
	spec CircuitBreakerSpec
	now  func() time.Time

	mu           sync.Mutex
	state        circuitState
	failures     int
	openedAt     time.Time
	halfOpenReqs int
}

// NewCircuitBreaker builds an in-memory circuit breaker, applying defaults for
// any zero spec fields.
func NewCircuitBreaker(spec CircuitBreakerSpec) CircuitBreaker {
	return newCircuitBreaker(spec, time.Now)
}

func newCircuitBreaker(spec CircuitBreakerSpec, now func() time.Time) *circuitBreaker {
	if spec.FailureThreshold <= 0 {
		spec.FailureThreshold = 5
	}
	if spec.ResetTimeout <= 0 {
		spec.ResetTimeout = 30 * time.Second
	}
	if spec.HalfOpenMax <= 0 {
		spec.HalfOpenMax = 1
	}
	return &circuitBreaker{spec: spec, now: now, state: circuitClosed}
}

func (b *circuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if err := b.allow(); err != nil {
		return err
	}
	err := fn(ctx)
	b.record(err)
	return err
}

// allow decides whether a call may proceed, advancing the state machine.
func (b *circuitBreaker) allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case circuitOpen:
		if b.now().Sub(b.openedAt) < b.spec.ResetTimeout {
			return ErrCircuitOpen
		}
		// Reset window elapsed: begin probing.
		b.state = circuitHalfOpen
		b.halfOpenReqs = 1
		return nil
	case circuitHalfOpen:
		if b.halfOpenReqs >= b.spec.HalfOpenMax {
			return ErrCircuitOpen
		}
		b.halfOpenReqs++
		return nil
	default:
		return nil
	}
}

// record folds a call outcome back into the breaker state.
func (b *circuitBreaker) record(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		switch b.state {
		case circuitHalfOpen:
			b.trip()
		default:
			b.failures++
			if b.failures >= b.spec.FailureThreshold {
				b.trip()
			}
		}
		return
	}
	// Success.
	switch b.state {
	case circuitHalfOpen:
		b.state = circuitClosed
		b.failures = 0
		b.halfOpenReqs = 0
	default:
		b.failures = 0
	}
}

func (b *circuitBreaker) trip() {
	b.state = circuitOpen
	b.openedAt = b.now()
	b.failures = 0
	b.halfOpenReqs = 0
}

// circuitBreakerRegistry is the default provider: it caches one breaker per
// name so all methods naming the same breaker share its state.
type circuitBreakerRegistry struct {
	mu       sync.Mutex
	breakers map[string]CircuitBreaker
}

// NewCircuitBreakerRegistry returns the default CircuitBreakerProvider backed by
// in-memory breakers.
func NewCircuitBreakerRegistry() CircuitBreakerProvider {
	return &circuitBreakerRegistry{breakers: map[string]CircuitBreaker{}}
}

func (r *circuitBreakerRegistry) CircuitBreaker(spec CircuitBreakerSpec) CircuitBreaker {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.breakers[spec.Name]; ok {
		return b
	}
	b := NewCircuitBreaker(spec)
	r.breakers[spec.Name] = b
	return b
}

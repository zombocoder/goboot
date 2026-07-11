// Package resilienceapp exercises @Retry and @Timeout interceptors (§36).
package resilienceapp

import "context"

// @Application(name="resilience-app")
type Application struct{}

// Flaky is the interface the service is exposed as.
type Flaky interface {
	Attempt(ctx context.Context) (int, error)
	Slow(ctx context.Context) error
}

// FlakyService fails a configurable number of times before succeeding.
//
// @Service(name="flaky", implements="Flaky")
type FlakyService struct {
	failuresLeft int
	calls        int
}

// NewFlakyService constructs the service. failuresLeft controls how many times
// Attempt fails before succeeding; the integration test sets it directly.
func NewFlakyService() *FlakyService { return &FlakyService{} }

// SetFailures configures the number of initial failures (test helper).
func (s *FlakyService) SetFailures(n int) { s.failuresLeft = n }

// Calls returns how many times Attempt has been invoked (test helper).
func (s *FlakyService) Calls() int { return s.calls }

// Attempt retries up to 4 times with backoff.
//
// @Retry(maxAttempts=4, delay="1ms", multiplier=2.0, maxDelay="10ms")
func (s *FlakyService) Attempt(ctx context.Context) (int, error) {
	s.calls++
	if s.failuresLeft > 0 {
		s.failuresLeft--
		return 0, errTransient
	}
	return s.calls, nil
}

// Slow is bounded by a timeout.
//
// @Timeout("20ms")
func (s *FlakyService) Slow(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	}
}

var errTransient = transientError("transient failure")

type transientError string

func (e transientError) Error() string { return string(e) }

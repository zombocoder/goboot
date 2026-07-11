// Package gateapp exercises the resilience-gate interceptors: circuit breaker,
// rate limit, and bulkhead (§36.3–§36.5).
package gateapp

import "context"

// @Application(name="gate-app")
type Application struct{}

// Downstream is the interface the service is exposed as.
type Downstream interface {
	Call(ctx context.Context) error
	Fetch(ctx context.Context) (string, error)
	Bounded(ctx context.Context) error
}

// DownstreamService gates its methods with resilience interceptors.
//
// @Service(name="downstream", implements="Downstream")
type DownstreamService struct {
	calls int
	fail  bool
}

func NewDownstreamService() *DownstreamService { return &DownstreamService{} }

// Calls reports how many times the target Call ran (test helper).
func (s *DownstreamService) Calls() int { return s.calls }

// SetFail toggles whether Call returns an error (test helper).
func (s *DownstreamService) SetFail(v bool) { s.fail = v }

// Call trips a circuit breaker after two consecutive failures.
//
// @CircuitBreaker(name="downstream", failureThreshold=2, resetTimeout="50ms")
func (s *DownstreamService) Call(ctx context.Context) error {
	s.calls++
	if s.fail {
		return context.Canceled
	}
	return nil
}

// Fetch is rate limited to two calls per second.
//
// @RateLimit(limit=2, period="1s")
func (s *DownstreamService) Fetch(ctx context.Context) (string, error) {
	return "ok", nil
}

// Bounded runs at most one at a time, rejecting overflow immediately.
//
// @Bulkhead(maxConcurrent=1)
func (s *DownstreamService) Bounded(ctx context.Context) error {
	return nil
}

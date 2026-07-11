package runtime

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrRateLimited is returned when a rate limiter has no permit available for a
// call (§36.4).
var ErrRateLimited = errors.New("goboot: rate limit exceeded")

// RateLimitSpec configures a named rate limiter from @RateLimit arguments
// (§36.4). Zero fields fall back to the defaults in NewRateLimiter.
type RateLimitSpec struct {
	Name string
	// Limit is the number of permits refilled every Period (default 100).
	Limit int
	// Period is the refill window (default 1s).
	Period time.Duration
	// Burst is the maximum number of permits that can accumulate; defaults to
	// Limit when zero.
	Burst int
}

// RateLimiter throttles calls, rejecting with ErrRateLimited when the rate is
// exceeded (§36.4).
type RateLimiter interface {
	// Execute runs fn when a permit is available, otherwise returns
	// ErrRateLimited without calling fn.
	Execute(ctx context.Context, fn func(context.Context) error) error
}

// RateLimiterProvider resolves a rate limiter for a spec, caching by name so
// repeated calls share the same token bucket (§36.4).
type RateLimiterProvider interface {
	RateLimiter(spec RateLimitSpec) RateLimiter
}

// tokenBucket is the built-in rate limiter. now is injectable for tests.
type tokenBucket struct {
	limit  float64
	period time.Duration
	burst  float64
	now    func() time.Time

	mu       sync.Mutex
	tokens   float64
	lastFill time.Time
}

// NewRateLimiter builds an in-memory token-bucket rate limiter, applying
// defaults for any zero spec fields.
func NewRateLimiter(spec RateLimitSpec) RateLimiter {
	return newTokenBucket(spec, time.Now)
}

func newTokenBucket(spec RateLimitSpec, now func() time.Time) *tokenBucket {
	if spec.Limit <= 0 {
		spec.Limit = 100
	}
	if spec.Period <= 0 {
		spec.Period = time.Second
	}
	if spec.Burst <= 0 {
		spec.Burst = spec.Limit
	}
	return &tokenBucket{
		limit:    float64(spec.Limit),
		period:   spec.Period,
		burst:    float64(spec.Burst),
		now:      now,
		tokens:   float64(spec.Burst),
		lastFill: now(),
	}
}

func (t *tokenBucket) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !t.take() {
		return ErrRateLimited
	}
	return fn(ctx)
}

// take refills the bucket for elapsed time and consumes one token if available.
func (t *tokenBucket) take() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.now()
	elapsed := now.Sub(t.lastFill)
	t.lastFill = now
	if elapsed > 0 {
		perSecond := t.limit / t.period.Seconds()
		t.tokens += elapsed.Seconds() * perSecond
		if t.tokens > t.burst {
			t.tokens = t.burst
		}
	}
	if t.tokens >= 1 {
		t.tokens--
		return true
	}
	return false
}

// rateLimiterRegistry is the default provider: one bucket per name.
type rateLimiterRegistry struct {
	mu       sync.Mutex
	limiters map[string]RateLimiter
}

// NewRateLimiterRegistry returns the default RateLimiterProvider backed by
// in-memory token buckets.
func NewRateLimiterRegistry() RateLimiterProvider {
	return &rateLimiterRegistry{limiters: map[string]RateLimiter{}}
}

func (r *rateLimiterRegistry) RateLimiter(spec RateLimitSpec) RateLimiter {
	r.mu.Lock()
	defer r.mu.Unlock()
	if l, ok := r.limiters[spec.Name]; ok {
		return l
	}
	l := NewRateLimiter(spec)
	r.limiters[spec.Name] = l
	return l
}

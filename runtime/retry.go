package runtime

import (
	"context"
	"time"
)

// RetryPolicy configures @Retry interception (§36.1).
type RetryPolicy struct {
	// MaxAttempts is the total number of attempts, including the first. Values
	// below 1 are treated as 1.
	MaxAttempts int
	// Delay is the wait before the first retry.
	Delay time.Duration
	// Multiplier scales the delay after each attempt (exponential backoff). A
	// value of 1 or less keeps the delay constant.
	Multiplier float64
	// MaxDelay caps the backoff delay; zero means no cap.
	MaxDelay time.Duration
}

// Retry runs fn, retrying on error up to the policy's attempts with exponential
// backoff. It honors context cancellation: if the context is done while waiting
// to retry, it returns the context error (§36.1).
func Retry(ctx context.Context, policy RetryPolicy, fn func(ctx context.Context) error) error {
	attempts := policy.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}
	delay := policy.Delay

	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if err = fn(ctx); err == nil {
			return nil
		}
		if attempt == attempts-1 {
			break // last attempt; do not wait
		}
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		} else if ctx.Err() != nil {
			return ctx.Err()
		}
		if policy.Multiplier > 1 {
			delay = time.Duration(float64(delay) * policy.Multiplier)
			if policy.MaxDelay > 0 && delay > policy.MaxDelay {
				delay = policy.MaxDelay
			}
		}
	}
	return err
}

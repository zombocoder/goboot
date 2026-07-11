package runtime

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetrySucceedsFirstTry(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), RetryPolicy{MaxAttempts: 3}, func(context.Context) error {
		calls++
		return nil
	})
	if err != nil || calls != 1 {
		t.Errorf("calls=%d err=%v, want 1 call, nil", calls, err)
	}
}

func TestRetryRetriesThenSucceeds(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), RetryPolicy{MaxAttempts: 3, Delay: time.Millisecond}, func(context.Context) error {
		calls++
		if calls < 3 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil || calls != 3 {
		t.Errorf("calls=%d err=%v, want 3 calls, nil", calls, err)
	}
}

func TestRetryExhaustsAttempts(t *testing.T) {
	calls := 0
	sentinel := errors.New("always")
	err := Retry(context.Background(), RetryPolicy{MaxAttempts: 4, Delay: time.Millisecond}, func(context.Context) error {
		calls++
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Errorf("err = %v, want sentinel", err)
	}
	if calls != 4 {
		t.Errorf("calls = %d, want 4", calls)
	}
}

func TestRetryHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	err := Retry(ctx, RetryPolicy{MaxAttempts: 10, Delay: 50 * time.Millisecond}, func(context.Context) error {
		calls++
		cancel() // cancel during the first attempt
		return errors.New("fail")
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (should stop after cancellation)", calls)
	}
}

func TestRetryBackoffMultiplier(t *testing.T) {
	// With a multiplier the delay grows; assert it retries the expected number
	// of times without asserting exact timing.
	calls := 0
	_ = Retry(context.Background(), RetryPolicy{MaxAttempts: 3, Delay: time.Millisecond, Multiplier: 2, MaxDelay: 5 * time.Millisecond},
		func(context.Context) error { calls++; return errors.New("x") })
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryZeroAttemptsRunsOnce(t *testing.T) {
	calls := 0
	_ = Retry(context.Background(), RetryPolicy{MaxAttempts: 0}, func(context.Context) error {
		calls++
		return errors.New("x")
	})
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

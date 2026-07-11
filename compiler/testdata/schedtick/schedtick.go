// Package schedtick is a fast, observable @Scheduled fixture for the runtime
// integration test.
package schedtick

import (
	"context"
	"sync/atomic"
)

// @Application(name="sched-tick")
type Application struct{}

// Ticker counts how many times its scheduled task has fired.
//
// @Service(name="ticker")
type Ticker struct {
	count int64
}

func NewTicker() *Ticker { return &Ticker{} }

// Count returns the number of ticks so far.
func (t *Ticker) Count() int64 { return atomic.LoadInt64(&t.count) }

// Tick fires every 5 milliseconds.
//
// @Scheduled(fixedRate="5ms")
func (t *Ticker) Tick(ctx context.Context) error {
	atomic.AddInt64(&t.count, 1)
	return nil
}

package runtime

import (
	"context"
	"errors"
	"sync"
	"time"
)

// ErrBulkheadFull is returned when a bulkhead has no free concurrency slot for a
// call (§36.5).
var ErrBulkheadFull = errors.New("goboot: bulkhead full")

// BulkheadSpec configures a named bulkhead from @Bulkhead arguments (§36.5).
// Zero fields fall back to the defaults in NewBulkhead.
type BulkheadSpec struct {
	Name string
	// MaxConcurrent is the number of calls allowed to run at once (default 10).
	MaxConcurrent int
	// MaxWait is how long a call waits for a free slot before being rejected;
	// zero rejects immediately (default 0).
	MaxWait time.Duration
}

// Bulkhead isolates a call behind a concurrency limit, rejecting with
// ErrBulkheadFull when the limit (and any wait) is exhausted (§36.5).
type Bulkhead interface {
	// Execute runs fn once a slot is acquired, otherwise returns ErrBulkheadFull.
	Execute(ctx context.Context, fn func(context.Context) error) error
}

// BulkheadProvider resolves a bulkhead for a spec, caching by name so repeated
// calls share the same slot pool (§36.5).
type BulkheadProvider interface {
	Bulkhead(spec BulkheadSpec) Bulkhead
}

// semaphoreBulkhead is the built-in bulkhead backed by a buffered channel.
type semaphoreBulkhead struct {
	slots   chan struct{}
	maxWait time.Duration
}

// NewBulkhead builds an in-memory semaphore bulkhead, applying defaults for any
// zero spec fields.
func NewBulkhead(spec BulkheadSpec) Bulkhead {
	if spec.MaxConcurrent <= 0 {
		spec.MaxConcurrent = 10
	}
	return &semaphoreBulkhead{
		slots:   make(chan struct{}, spec.MaxConcurrent),
		maxWait: spec.MaxWait,
	}
}

func (b *semaphoreBulkhead) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !b.acquire(ctx) {
		return ErrBulkheadFull
	}
	defer func() { <-b.slots }()
	return fn(ctx)
}

// acquire tries to take a slot, waiting up to MaxWait (and honoring context
// cancellation). It reports whether a slot was acquired.
func (b *semaphoreBulkhead) acquire(ctx context.Context) bool {
	select {
	case b.slots <- struct{}{}:
		return true
	default:
	}
	if b.maxWait <= 0 {
		return false
	}
	timer := time.NewTimer(b.maxWait)
	defer timer.Stop()
	select {
	case b.slots <- struct{}{}:
		return true
	case <-timer.C:
		return false
	case <-ctx.Done():
		return false
	}
}

// bulkheadRegistry is the default provider: one semaphore per name.
type bulkheadRegistry struct {
	mu        sync.Mutex
	bulkheads map[string]Bulkhead
}

// NewBulkheadRegistry returns the default BulkheadProvider backed by in-memory
// semaphores.
func NewBulkheadRegistry() BulkheadProvider {
	return &bulkheadRegistry{bulkheads: map[string]Bulkhead{}}
}

func (r *bulkheadRegistry) Bulkhead(spec BulkheadSpec) Bulkhead {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.bulkheads[spec.Name]; ok {
		return b
	}
	b := NewBulkhead(spec)
	r.bulkheads[spec.Name] = b
	return b
}

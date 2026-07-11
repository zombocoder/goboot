package runtime

import (
	"context"
	"time"
)

// IsolationLevel selects a transaction's isolation (§26.3).
type IsolationLevel int

const (
	// IsolationDefault uses the database's default isolation.
	IsolationDefault IsolationLevel = iota
	IsolationReadCommitted
	IsolationRepeatableRead
	IsolationSerializable
)

// Propagation selects how a @Transactional method joins an existing transaction
// (§26.4). The MVP implements only PropagationRequired.
type Propagation int

const (
	// PropagationRequired joins the current transaction or starts a new one.
	PropagationRequired Propagation = iota
	PropagationRequiresNew
	PropagationSupports
	PropagationNotSupported
)

// TransactionOptions configures a transactional method (§26.3).
type TransactionOptions struct {
	ReadOnly    bool
	Isolation   IsolationLevel
	Propagation Propagation
	Timeout     time.Duration
}

// TransactionManager runs a callback within a transaction (§26.2). The callback
// receives a context carrying the active transaction; repositories retrieve it
// from the context rather than any global (§26.5). A non-nil error rolls the
// transaction back; nil commits (§26.6).
type TransactionManager interface {
	WithinTransaction(ctx context.Context, opts TransactionOptions, fn func(ctx context.Context) error) error
}

// DirectTransactionManager runs the callback directly with no real transaction.
// It is the default so generated proxies work before a database adapter is
// configured; the rollback-on-error contract is preserved (the error simply
// propagates).
type DirectTransactionManager struct{}

// WithinTransaction runs fn with the given context.
func (DirectTransactionManager) WithinTransaction(ctx context.Context, _ TransactionOptions, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

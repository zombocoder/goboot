package db

import "context"

// txKeyType is the unexported context key under which the active transaction is
// stored, avoiding collisions with other packages' context values.
type txKeyType struct{}

var txKey txKeyType

// WithTx returns a context carrying tx as the active transaction. A
// TransactionManager calls this before invoking the transactional callback so
// that repositories run against the transaction (§26.5).
func WithTx(ctx context.Context, tx DBTX) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// ActiveTx returns the active transaction carried by the context, if any.
func ActiveTx(ctx context.Context) (DBTX, bool) {
	tx, ok := ctx.Value(txKey).(DBTX)
	return tx, ok
}

// NewProvider returns a DBProvider backed by pool that transparently prefers the
// active transaction from the context when one is present (§26.5). Adapters wrap
// their connection pool as a DBTX and pass it here.
func NewProvider(pool DBTX) DBProvider {
	return contextProvider{pool: pool}
}

// contextProvider selects the active transaction or the pool per call.
type contextProvider struct {
	pool DBTX
}

// DB returns the active transaction if the context carries one, else the pool.
func (p contextProvider) DB(ctx context.Context) DBTX {
	if tx, ok := ActiveTx(ctx); ok {
		return tx
	}
	return p.pool
}

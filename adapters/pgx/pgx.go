// Package pgx adapts jackc/pgx v5 to goboot's driver-neutral db abstraction
// (§6.6, §27) so generated repositories run natively on PostgreSQL over a
// pgxpool connection pool. It lives in its own module to keep the pgx dependency
// out of the goboot core; nothing here changes the runtime or any generated
// repository. Pair it with the default `postgres` dialect ($1, $2, …).
//
// Wire it into the generated dependencies:
//
//	pool, _ := pgxpool.New(ctx, dsn)
//	proxyDeps := runtime.DefaultProxyDependencies()
//	proxyDeps.Transactions = adapterpgx.NewTransactionManager(pool)
//	dbProvider := adapterpgx.NewProvider(pool)
package pgx

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	goruntime "github.com/zombocoder/goboot/runtime"
	"github.com/zombocoder/goboot/runtime/db"
)

// querier is the subset of *pgxpool.Pool and pgx.Tx the adapter uses, so a pool
// and a transaction are wrapped uniformly.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// dbtx wraps a pgx querier as a db.DBTX.
type dbtx struct{ q querier }

// QueryContext implements db.DBTX.
func (t dbtx) QueryContext(ctx context.Context, query string, args ...any) (db.Rows, error) {
	rows, err := t.q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return pgxRows{rows}, nil
}

// QueryRowContext implements db.DBTX, translating no-rows to db.ErrNoRows.
func (t dbtx) QueryRowContext(ctx context.Context, query string, args ...any) db.Row {
	return pgxRow{t.q.QueryRow(ctx, query, args...)}
}

// ExecContext implements db.DBTX.
func (t dbtx) ExecContext(ctx context.Context, query string, args ...any) (db.Result, error) {
	tag, err := t.q.Exec(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return result{tag}, nil
}

// pgxRows adapts pgx.Rows to db.Rows. pgx.Rows.Close returns nothing; any
// iteration error surfaces through Err.
type pgxRows struct{ rows pgx.Rows }

func (r pgxRows) Next() bool             { return r.rows.Next() }
func (r pgxRows) Scan(dest ...any) error { return r.rows.Scan(dest...) }
func (r pgxRows) Err() error             { return r.rows.Err() }
func (r pgxRows) Close() error           { r.rows.Close(); return nil }

// pgxRow adapts pgx.Row to db.Row, translating pgx.ErrNoRows into db.ErrNoRows
// so generated repositories stay driver-neutral (§27.7).
type pgxRow struct{ row pgx.Row }

func (r pgxRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	if errors.Is(err, pgx.ErrNoRows) {
		return db.ErrNoRows
	}
	return err
}

// result adapts a pgconn.CommandTag to db.Result.
type result struct{ tag pgconn.CommandTag }

func (r result) RowsAffected() (int64, error) { return r.tag.RowsAffected(), nil }

// NewProvider returns a db.DBProvider backed by a pgxpool.Pool that
// transparently uses the active transaction from the context when present
// (§26.5).
func NewProvider(pool *pgxpool.Pool) db.DBProvider {
	return db.NewProvider(dbtx{q: pool})
}

// TransactionManager implements goboot's runtime.TransactionManager over pgx,
// beginning a transaction, publishing it on the context so repositories join it,
// and committing or rolling back on the callback's result (§26.2, §26.6).
type TransactionManager struct{ pool *pgxpool.Pool }

// NewTransactionManager builds a TransactionManager over the pool.
func NewTransactionManager(pool *pgxpool.Pool) *TransactionManager {
	return &TransactionManager{pool: pool}
}

// WithinTransaction runs fn inside a pgx transaction.
func (m *TransactionManager) WithinTransaction(ctx context.Context, opts goruntime.TransactionOptions, fn func(ctx context.Context) error) error {
	tx, err := m.pool.BeginTx(ctx, pgxTxOptions(opts))
	if err != nil {
		return err
	}
	txCtx := db.WithTx(ctx, dbtx{q: tx})
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// pgxTxOptions maps goboot transaction options to pgx's.
func pgxTxOptions(o goruntime.TransactionOptions) pgx.TxOptions {
	opts := pgx.TxOptions{IsoLevel: mapIsolation(o.Isolation)}
	if o.ReadOnly {
		opts.AccessMode = pgx.ReadOnly
	}
	return opts
}

// Compile-time assertions that the adapter satisfies goboot's contracts.
var (
	_ db.DBTX                      = dbtx{}
	_ db.Rows                      = pgxRows{}
	_ db.Row                       = pgxRow{}
	_ db.Result                    = result{}
	_ goruntime.TransactionManager = (*TransactionManager)(nil)
)

// mapIsolation maps goboot isolation levels to pgx's; an empty level uses the
// server default.
func mapIsolation(level goruntime.IsolationLevel) pgx.TxIsoLevel {
	switch level {
	case goruntime.IsolationReadCommitted:
		return pgx.ReadCommitted
	case goruntime.IsolationRepeatableRead:
		return pgx.RepeatableRead
	case goruntime.IsolationSerializable:
		return pgx.Serializable
	default:
		return ""
	}
}

// Package databasesql adapts Go's standard database/sql to goboot's
// driver-neutral db abstraction (§6.6, §27). Because it targets the standard
// library rather than a specific driver, it works with any registered
// database/sql driver — PostgreSQL (pgx stdlib, pq), MySQL, SQLite, and others —
// while the SQL dialect is chosen independently at generation time. A pgx-native
// adapter or a plugin-provided driver can be added later without changing the
// runtime or any generated repository.
package databasesql

import (
	"context"
	"database/sql"
	"errors"

	goruntime "github.com/zombocoder/goboot/runtime"
	"github.com/zombocoder/goboot/runtime/db"
)

// querier is the subset of *sql.DB and *sql.Tx the adapter uses, so both a pool
// and a transaction are wrapped uniformly.
type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// dbtx wraps a database/sql querier as a db.DBTX.
type dbtx struct{ q querier }

// QueryContext implements db.DBTX. *sql.Rows already satisfies db.Rows.
func (t dbtx) QueryContext(ctx context.Context, query string, args ...any) (db.Rows, error) {
	rows, err := t.q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// QueryRowContext implements db.DBTX, translating no-rows to db.ErrNoRows.
func (t dbtx) QueryRowContext(ctx context.Context, query string, args ...any) db.Row {
	return row{t.q.QueryRowContext(ctx, query, args...)}
}

// ExecContext implements db.DBTX. sql.Result already satisfies db.Result.
func (t dbtx) ExecContext(ctx context.Context, query string, args ...any) (db.Result, error) {
	return t.q.ExecContext(ctx, query, args...)
}

// row translates database/sql's sql.ErrNoRows into db.ErrNoRows so generated
// repositories stay driver-neutral (§27.7).
type row struct{ r *sql.Row }

func (r row) Scan(dest ...any) error {
	err := r.r.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return db.ErrNoRows
	}
	return err
}

// NewProvider returns a db.DBProvider backed by a *sql.DB pool that transparently
// uses the active transaction from the context when one is present (§26.5).
func NewProvider(pool *sql.DB) db.DBProvider {
	return db.NewProvider(dbtx{q: pool})
}

// TransactionManager implements goboot's runtime.TransactionManager over
// database/sql, beginning a transaction, publishing it on the context so
// repositories join it, and committing or rolling back on the callback's result
// (§26.2, §26.6).
type TransactionManager struct{ pool *sql.DB }

// NewTransactionManager builds a TransactionManager over the pool.
func NewTransactionManager(pool *sql.DB) *TransactionManager {
	return &TransactionManager{pool: pool}
}

// WithinTransaction runs fn inside a database/sql transaction.
func (m *TransactionManager) WithinTransaction(ctx context.Context, opts goruntime.TransactionOptions, fn func(ctx context.Context) error) error {
	tx, err := m.pool.BeginTx(ctx, sqlTxOptions(opts))
	if err != nil {
		return err
	}
	txCtx := db.WithTx(ctx, dbtx{q: tx})
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// sqlTxOptions maps goboot transaction options to database/sql's.
func sqlTxOptions(o goruntime.TransactionOptions) *sql.TxOptions {
	return &sql.TxOptions{
		ReadOnly:  o.ReadOnly,
		Isolation: mapIsolation(o.Isolation),
	}
}

// mapIsolation maps goboot isolation levels to database/sql's.
func mapIsolation(level goruntime.IsolationLevel) sql.IsolationLevel {
	switch level {
	case goruntime.IsolationReadCommitted:
		return sql.LevelReadCommitted
	case goruntime.IsolationRepeatableRead:
		return sql.LevelRepeatableRead
	case goruntime.IsolationSerializable:
		return sql.LevelSerializable
	default:
		return sql.LevelDefault
	}
}

// Package db defines the driver-neutral database abstractions that generated
// repositories depend on (§27). Nothing here imports a concrete driver: pgx,
// database/sql, and future plugin-provided drivers all satisfy these interfaces
// through adapters (§6.6). The generated repository code calls only DBProvider
// and DBTX, so swapping drivers never changes generated output — only the
// adapter and the SQL dialect differ.
package db

import (
	"context"
	"errors"
)

// ErrNoRows is returned by a single-row query that matched nothing (§27.7).
// Adapters translate their driver's no-rows error into this sentinel so
// generated code is driver-neutral.
var ErrNoRows = errors.New("goboot/db: no rows in result set")

// Rows is an iterable result set. Its shape matches database/sql's *Rows so the
// standard adapter can return it directly.
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Close() error
	Err() error
}

// Row is a single-row result whose Scan reports ErrNoRows when empty.
type Row interface {
	Scan(dest ...any) error
}

// Result is the outcome of an Exec.
type Result interface {
	RowsAffected() (int64, error)
}

// DBTX is the minimal query surface a repository needs. Both a connection pool
// and an active transaction implement it, so repositories are agnostic to which
// they run against (§26.5).
type DBTX interface {
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	ExecContext(ctx context.Context, query string, args ...any) (Result, error)
}

// DBProvider returns the DBTX a repository should use for the current call: the
// active transaction when one is present in the context, otherwise the pool
// (§26.5). The runtime never stores transactions in globals.
type DBProvider interface {
	DB(ctx context.Context) DBTX
}

// RowMapper maps a single Row into a value of type T (§27.8). Generated
// repositories may accept a mapper instead of scanning by field order.
type RowMapper[T any] interface {
	MapRow(row Row) (T, error)
}

package pgx

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	goruntime "github.com/zombocoder/goboot/runtime"
	"github.com/zombocoder/goboot/runtime/db"
)

// TestPostgresIntegration exercises the adapter against a real database. Set
// PGX_TEST_DSN (e.g. postgres://user:pass@localhost:5432/db) to run it; it is
// skipped otherwise so the default test run stays hermetic.
func TestPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("PGX_TEST_DSN")
	if dsn == "" {
		t.Skip("set PGX_TEST_DSN to run the pgx integration test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer pool.Close()

	provider := NewProvider(pool)
	txm := NewTransactionManager(pool)

	// A temporary table for the test connection's session.
	if _, err := pool.Exec(ctx, `CREATE TEMP TABLE goboot_pgx_test (id text primary key, n int)`); err != nil {
		t.Fatalf("create temp table: %v", err)
	}

	// Exec + rows-affected through the provider's DBTX.
	res, err := provider.DB(ctx).ExecContext(ctx, `INSERT INTO goboot_pgx_test (id, n) VALUES ($1, $2)`, "a", 1)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if n, _ := res.RowsAffected(); n != 1 {
		t.Errorf("rows affected = %d, want 1", n)
	}

	// Single-row scan.
	var n int
	if err := provider.DB(ctx).QueryRowContext(ctx, `SELECT n FROM goboot_pgx_test WHERE id = $1`, "a").Scan(&n); err != nil {
		t.Fatalf("query row: %v", err)
	}
	if n != 1 {
		t.Errorf("n = %d, want 1", n)
	}

	// No-rows translates to db.ErrNoRows.
	err = provider.DB(ctx).QueryRowContext(ctx, `SELECT n FROM goboot_pgx_test WHERE id = $1`, "missing").Scan(&n)
	if !errors.Is(err, db.ErrNoRows) {
		t.Errorf("missing row error = %v, want db.ErrNoRows", err)
	}

	// Multi-row query iteration.
	rows, err := provider.DB(ctx).QueryContext(ctx, `SELECT id FROM goboot_pgx_test ORDER BY id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	count := 0
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatalf("scan: %v", err)
		}
		count++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	if count != 1 {
		t.Errorf("row count = %d, want 1", count)
	}

	// A transaction that returns an error rolls back; the provider inside the
	// callback uses the active transaction from the context.
	wantErr := errors.New("rollback please")
	err = txm.WithinTransaction(ctx, goruntime.TransactionOptions{}, func(txCtx context.Context) error {
		if _, err := provider.DB(txCtx).ExecContext(txCtx, `INSERT INTO goboot_pgx_test (id, n) VALUES ($1, $2)`, "b", 2); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("transaction error = %v, want %v", err, wantErr)
	}
	if err := provider.DB(ctx).QueryRowContext(ctx, `SELECT n FROM goboot_pgx_test WHERE id = $1`, "b").Scan(&n); !errors.Is(err, db.ErrNoRows) {
		t.Errorf("rolled-back row should be absent, got err %v", err)
	}

	// A transaction that succeeds commits.
	err = txm.WithinTransaction(ctx, goruntime.TransactionOptions{}, func(txCtx context.Context) error {
		_, err := provider.DB(txCtx).ExecContext(txCtx, `INSERT INTO goboot_pgx_test (id, n) VALUES ($1, $2)`, "c", 3)
		return err
	})
	if err != nil {
		t.Fatalf("commit transaction: %v", err)
	}
	if err := provider.DB(ctx).QueryRowContext(ctx, `SELECT n FROM goboot_pgx_test WHERE id = $1`, "c").Scan(&n); err != nil || n != 3 {
		t.Errorf("committed row missing: n=%d err=%v", n, err)
	}
}

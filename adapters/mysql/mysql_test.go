package mysql

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	goruntime "github.com/zombocoder/goboot/runtime"
	"github.com/zombocoder/goboot/runtime/db"
)

func TestOpen(t *testing.T) {
	pool, err := Open("user:pass@tcp(localhost:3306)/app?charset=utf8mb4")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pool.Close()
	if pool == nil {
		t.Fatal("expected a non-nil pool")
	}
	if _, err := Open("not-a-valid-dsn"); err == nil {
		t.Error("expected an error for a malformed DSN")
	}
}

func TestProviderQuery(t *testing.T) {
	pool, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	mock.ExpectQuery("SELECT name FROM users WHERE id = ?").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("alice"))

	ctx := context.Background()
	var name string
	err = NewProvider(pool).DB(ctx).
		QueryRowContext(ctx, "SELECT name FROM users WHERE id = ?", 1).
		Scan(&name)
	if err != nil || name != "alice" {
		t.Fatalf("query = %q, %v", name, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestProviderErrNoRows(t *testing.T) {
	pool, mock, _ := sqlmock.New()
	defer pool.Close()
	mock.ExpectQuery("SELECT x").WillReturnRows(sqlmock.NewRows([]string{"x"})) // empty

	ctx := context.Background()
	var x int
	err := NewProvider(pool).DB(ctx).QueryRowContext(ctx, "SELECT x").Scan(&x)
	if !errors.Is(err, db.ErrNoRows) {
		t.Errorf("empty single-row query = %v, want db.ErrNoRows", err)
	}
}

func TestWithinTransactionCommit(t *testing.T) {
	pool, mock, _ := sqlmock.New()
	defer pool.Close()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO t").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	provider := NewProvider(pool)
	tm := NewTransactionManager(pool)
	err := tm.WithinTransaction(context.Background(), goruntime.TransactionOptions{}, func(ctx context.Context) error {
		_, e := provider.DB(ctx).ExecContext(ctx, "INSERT INTO t VALUES (1)")
		return e
	})
	if err != nil {
		t.Fatalf("WithinTransaction: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestWithinTransactionRollback(t *testing.T) {
	pool, mock, _ := sqlmock.New()
	defer pool.Close()
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO t").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectRollback()

	provider := NewProvider(pool)
	tm := NewTransactionManager(pool)
	boom := errors.New("boom")
	err := tm.WithinTransaction(context.Background(), goruntime.TransactionOptions{}, func(ctx context.Context) error {
		_, _ = provider.DB(ctx).ExecContext(ctx, "INSERT INTO t VALUES (1)")
		return boom
	})
	if !errors.Is(err, boom) {
		t.Errorf("WithinTransaction err = %v, want boom", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

// TestIntegration runs a repository-style round trip against a real MySQL. It is
// skipped unless GOBOOT_MYSQL_DSN is set, e.g.
// GOBOOT_MYSQL_DSN='root:root@tcp(localhost:3306)/test' go test ./adapters/mysql/.
func TestIntegration(t *testing.T) {
	dsn := os.Getenv("GOBOOT_MYSQL_DSN")
	if dsn == "" {
		t.Skip("set GOBOOT_MYSQL_DSN to run the MySQL integration test")
	}
	pool, err := Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	ctx := context.Background()
	if err := pool.PingContext(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	provider := NewProvider(pool)
	tm := NewTransactionManager(pool)
	exec := func(q string, args ...any) {
		if _, e := provider.DB(ctx).ExecContext(ctx, q, args...); e != nil {
			t.Fatalf("exec %q: %v", q, e)
		}
	}
	exec("DROP TABLE IF EXISTS goboot_it")
	exec("CREATE TABLE goboot_it (id INT PRIMARY KEY, name VARCHAR(64))")
	defer exec("DROP TABLE IF EXISTS goboot_it")

	// A write inside a @Transactional-style unit.
	if err := tm.WithinTransaction(ctx, goruntime.TransactionOptions{}, func(ctx context.Context) error {
		_, e := provider.DB(ctx).ExecContext(ctx, "INSERT INTO goboot_it (id, name) VALUES (?, ?)", 1, "alice")
		return e
	}); err != nil {
		t.Fatalf("tx: %v", err)
	}

	var name string
	if err := provider.DB(ctx).QueryRowContext(ctx, "SELECT name FROM goboot_it WHERE id = ?", 1).Scan(&name); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "alice" {
		t.Errorf("name = %q, want alice", name)
	}
	// A miss maps to db.ErrNoRows.
	if err := provider.DB(ctx).QueryRowContext(ctx, "SELECT name FROM goboot_it WHERE id = ?", 999).Scan(&name); !errors.Is(err, db.ErrNoRows) {
		t.Errorf("missing row = %v, want db.ErrNoRows", err)
	}
}

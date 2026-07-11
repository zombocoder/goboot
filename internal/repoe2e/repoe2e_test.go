// Package repoe2e exercises a generated repository end to end. It drives the
// generated methods against a fake db.DBTX, asserting the compiled SQL, bound
// arguments, row scanning, no-rows handling, and — crucially — that the
// repository uses the active transaction from the context when one is present
// (the M6/@Transactional to M7/@Query integration). wiring.gen.go is produced by
// the goboot generator from the repoapp example.
package repoe2e

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zombocoder/goboot/runtime/db"
)

// fakeDBTX records the query and args it received and returns canned results.
type fakeDBTX struct {
	name      string
	lastQuery string
	lastArgs  []any
	row       *fakeRow
	rows      *fakeRows
	result    *fakeResult
	queryErr  error
}

func (f *fakeDBTX) QueryRowContext(_ context.Context, query string, args ...any) db.Row {
	f.lastQuery, f.lastArgs = query, args
	return f.row
}

func (f *fakeDBTX) QueryContext(_ context.Context, query string, args ...any) (db.Rows, error) {
	f.lastQuery, f.lastArgs = query, args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.rows, nil
}

func (f *fakeDBTX) ExecContext(_ context.Context, query string, args ...any) (db.Result, error) {
	f.lastQuery, f.lastArgs = query, args
	if f.queryErr != nil {
		return nil, f.queryErr
	}
	return f.result, nil
}

type fakeRow struct {
	values []any
	err    error
}

func (r *fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	return assign(dest, r.values)
}

type fakeRows struct {
	data [][]any
	i    int
	err  error
}

func (r *fakeRows) Next() bool { r.i++; return r.i < len(r.data) }
func (r *fakeRows) Scan(dest ...any) error {
	return assign(dest, r.data[r.i])
}
func (r *fakeRows) Close() error { return nil }
func (r *fakeRows) Err() error   { return r.err }

type fakeResult struct{ affected int64 }

func (r fakeResult) RowsAffected() (int64, error) { return r.affected, nil }

// assign copies canned values into scan destinations.
func assign(dest []any, values []any) error {
	if len(dest) != len(values) {
		return errors.New("scan destination count mismatch")
	}
	for i, d := range dest {
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(values[i]))
	}
	return nil
}

func newRepo(pool *fakeDBTX) (*Components, error) {
	// db.NewProvider prefers the active transaction from the context, else pool.
	return buildComponents(db.NewProvider(pool))
}

func newFakeRows() *fakeRows { return &fakeRows{i: -1} }

func TestFindByID(t *testing.T) {
	pool := &fakeDBTX{name: "pool", row: &fakeRow{values: []any{"u1", "Ada", "ada@example.com"}}}
	comps, err := newRepo(pool)
	if err != nil {
		t.Fatal(err)
	}
	user, err := comps.UserRepository.FindByID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if user.ID != "u1" || user.Name != "Ada" || user.Email != "ada@example.com" {
		t.Errorf("scanned entity = %+v", user)
	}
	if pool.lastQuery != "SELECT id, name, email FROM users WHERE id = $1" {
		t.Errorf("query = %q", pool.lastQuery)
	}
	if len(pool.lastArgs) != 1 || pool.lastArgs[0] != "u1" {
		t.Errorf("args = %v", pool.lastArgs)
	}
}

func TestFindByIDNoRows(t *testing.T) {
	pool := &fakeDBTX{row: &fakeRow{err: db.ErrNoRows}}
	comps, _ := newRepo(pool)
	user, err := comps.UserRepository.FindByID(context.Background(), "missing")
	if !errors.Is(err, db.ErrNoRows) {
		t.Errorf("expected db.ErrNoRows, got %v", err)
	}
	if user != nil {
		t.Errorf("expected nil user on no rows, got %+v", user)
	}
}

func TestFindAll(t *testing.T) {
	rows := newFakeRows()
	rows.data = [][]any{
		{"u1", "Ada", "ada@example.com"},
		{"u2", "Alan", "alan@example.com"},
	}
	pool := &fakeDBTX{rows: rows}
	comps, _ := newRepo(pool)
	users, err := comps.UserRepository.FindAll(context.Background())
	if err != nil {
		t.Fatalf("FindAll: %v", err)
	}
	if len(users) != 2 || users[0].ID != "u1" || users[1].Name != "Alan" {
		t.Errorf("users = %+v", users)
	}
}

func TestCount(t *testing.T) {
	pool := &fakeDBTX{row: &fakeRow{values: []any{int64(7)}}}
	comps, _ := newRepo(pool)
	n, err := comps.UserRepository.Count(context.Background())
	if err != nil || n != 7 {
		t.Errorf("Count = %d, %v", n, err)
	}
}

func TestCreate(t *testing.T) {
	pool := &fakeDBTX{result: &fakeResult{}}
	comps, _ := newRepo(pool)
	err := comps.UserRepository.Create(context.Background(), "u1", "Ada", "ada@example.com")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if pool.lastQuery != "INSERT INTO users (id, name, email) VALUES ($1, $2, $3)" {
		t.Errorf("query = %q", pool.lastQuery)
	}
	if len(pool.lastArgs) != 3 || pool.lastArgs[2] != "ada@example.com" {
		t.Errorf("args = %v", pool.lastArgs)
	}
}

func TestDeleteRowsAffected(t *testing.T) {
	pool := &fakeDBTX{result: &fakeResult{affected: 3}}
	comps, _ := newRepo(pool)
	n, err := comps.UserRepository.Delete(context.Background(), "u1")
	if err != nil || n != 3 {
		t.Errorf("Delete = %d, %v", n, err)
	}
}

// TestRepositoryUsesActiveTransaction proves the §26.5 contract: when a
// transaction is present on the context, the repository runs against it, not the
// pool. This is the join point between @Transactional (M6) and @Query (M7).
func TestRepositoryUsesActiveTransaction(t *testing.T) {
	pool := &fakeDBTX{name: "pool", row: &fakeRow{values: []any{"u1", "Ada", "ada@example.com"}}}
	tx := &fakeDBTX{name: "tx", row: &fakeRow{values: []any{"u1", "Ada", "ada@example.com"}}}
	comps, _ := newRepo(pool)

	ctx := db.WithTx(context.Background(), tx)
	if _, err := comps.UserRepository.FindByID(ctx, "u1"); err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if tx.lastQuery == "" {
		t.Error("repository should have used the active transaction")
	}
	if pool.lastQuery != "" {
		t.Error("repository should NOT have used the pool while a transaction was active")
	}
}

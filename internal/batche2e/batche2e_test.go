// Package batche2e exercises generated @Batch and @Call repository methods
// against a fake db.DBTX: @Batch runs one statement per slice element (binding
// element fields and shared scalars), and @Call runs as an exec or a query per
// its return shape. wiring.gen.go is produced by the goboot generator from the
// batchapp example.
package batche2e

import (
	"context"
	"reflect"
	"testing"

	"github.com/zombocoder/goboot/compiler/testdata/batchapp"
	"github.com/zombocoder/goboot/runtime/db"
)

// recordingDBTX captures every exec/query call so batch iteration is observable.
type recordingDBTX struct {
	execs   []call
	queries []call
	rows    *fakeRows
}

type call struct {
	query string
	args  []any
}

func (f *recordingDBTX) ExecContext(_ context.Context, query string, args ...any) (db.Result, error) {
	f.execs = append(f.execs, call{query, args})
	return fakeResult{1}, nil
}

func (f *recordingDBTX) QueryContext(_ context.Context, query string, args ...any) (db.Rows, error) {
	f.queries = append(f.queries, call{query, args})
	return f.rows, nil
}

func (f *recordingDBTX) QueryRowContext(_ context.Context, query string, args ...any) db.Row {
	f.queries = append(f.queries, call{query, args})
	return nil
}

type fakeResult struct{ n int64 }

func (r fakeResult) RowsAffected() (int64, error) { return r.n, nil }

type fakeRows struct {
	data [][]any
	i    int
}

func (r *fakeRows) Next() bool { r.i++; return r.i <= len(r.data) }
func (r *fakeRows) Scan(dest ...any) error {
	for i, d := range dest {
		reflect.ValueOf(d).Elem().Set(reflect.ValueOf(r.data[r.i-1][i]))
	}
	return nil
}
func (r *fakeRows) Close() error { return nil }
func (r *fakeRows) Err() error   { return nil }

func repo(t *testing.T, tx db.DBTX) batchapp.UserRepository {
	t.Helper()
	comps, err := buildComponents(db.NewProvider(tx))
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	return comps.UserRepository
}

func TestBatchRunsPerElementWithRowsAffected(t *testing.T) {
	tx := &recordingDBTX{}
	r := repo(t, tx)

	n, err := r.InsertAll(context.Background(), []batchapp.User{
		{ID: "1", Name: "a"},
		{ID: "2", Name: "b"},
		{ID: "3", Name: "c"},
	})
	if err != nil {
		t.Fatalf("InsertAll: %v", err)
	}
	if n != 3 {
		t.Errorf("rows affected = %d, want 3 (one per element)", n)
	}
	if len(tx.execs) != 3 {
		t.Fatalf("expected 3 execs, got %d", len(tx.execs))
	}
	// Each exec binds that element's fields.
	if !reflect.DeepEqual(tx.execs[1].args, []any{"2", "b"}) {
		t.Errorf("second exec args = %v, want [2 b]", tx.execs[1].args)
	}
}

func TestBatchEmptySliceRunsNothing(t *testing.T) {
	tx := &recordingDBTX{}
	r := repo(t, tx)

	n, err := r.InsertAll(context.Background(), nil)
	if err != nil || n != 0 {
		t.Fatalf("empty InsertAll = %d, %v; want 0, nil", n, err)
	}
	if len(tx.execs) != 0 {
		t.Errorf("empty batch should run no execs, got %d", len(tx.execs))
	}
}

func TestBatchMixesScalarAndSlice(t *testing.T) {
	tx := &recordingDBTX{}
	r := repo(t, tx)

	if err := r.TouchAll(context.Background(), "acme", []string{"x", "y"}); err != nil {
		t.Fatalf("TouchAll: %v", err)
	}
	if len(tx.execs) != 2 {
		t.Fatalf("expected 2 execs, got %d", len(tx.execs))
	}
	// The shared scalar (org) is the first arg in every row; the element id second.
	if !reflect.DeepEqual(tx.execs[0].args, []any{"acme", "x"}) {
		t.Errorf("first exec args = %v, want [acme x]", tx.execs[0].args)
	}
	if !reflect.DeepEqual(tx.execs[1].args, []any{"acme", "y"}) {
		t.Errorf("second exec args = %v, want [acme y]", tx.execs[1].args)
	}
}

func TestCallExecForm(t *testing.T) {
	tx := &recordingDBTX{}
	r := repo(t, tx)

	if err := r.Reindex(context.Background()); err != nil {
		t.Fatalf("Reindex: %v", err)
	}
	if len(tx.execs) != 1 || tx.execs[0].query != "CALL reindex_users()" {
		t.Errorf("Reindex should exec the CALL once, got %+v", tx.execs)
	}
}

func TestCallQueryForm(t *testing.T) {
	tx := &recordingDBTX{rows: &fakeRows{data: [][]any{{"1", "ann"}, {"2", "bob"}}}}
	r := repo(t, tx)

	users, err := r.TopByScore(context.Background(), 5)
	if err != nil {
		t.Fatalf("TopByScore: %v", err)
	}
	if len(users) != 2 || users[0].Name != "ann" || users[1].ID != "2" {
		t.Errorf("TopByScore result = %+v", users)
	}
	if len(tx.queries) != 1 || !reflect.DeepEqual(tx.queries[0].args, []any{int64(5)}) {
		t.Errorf("TopByScore should query once with the limit arg, got %+v", tx.queries)
	}
}

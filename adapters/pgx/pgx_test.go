package pgx

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	goruntime "github.com/zombocoder/goboot/runtime"
	"github.com/zombocoder/goboot/runtime/db"
)

// fakeRow is a minimal pgx.Row for testing error translation.
type fakeRow struct{ err error }

func (f fakeRow) Scan(...any) error { return f.err }

func TestRowTranslatesNoRows(t *testing.T) {
	if err := (pgxRow{fakeRow{pgx.ErrNoRows}}).Scan(); !errors.Is(err, db.ErrNoRows) {
		t.Errorf("pgx.ErrNoRows should translate to db.ErrNoRows, got %v", err)
	}
	sentinel := errors.New("boom")
	if err := (pgxRow{fakeRow{sentinel}}).Scan(); !errors.Is(err, sentinel) {
		t.Errorf("other errors should pass through, got %v", err)
	}
	if err := (pgxRow{fakeRow{nil}}).Scan(); err != nil {
		t.Errorf("nil should pass through, got %v", err)
	}
}

func TestResultRowsAffected(t *testing.T) {
	n, err := result{pgconn.NewCommandTag("UPDATE 3")}.RowsAffected()
	if err != nil || n != 3 {
		t.Errorf("RowsAffected = %d, %v; want 3, nil", n, err)
	}
}

func TestTxOptionsMapping(t *testing.T) {
	ro := pgxTxOptions(goruntime.TransactionOptions{ReadOnly: true, Isolation: goruntime.IsolationSerializable})
	if ro.AccessMode != pgx.ReadOnly {
		t.Errorf("read-only not mapped: %+v", ro)
	}
	if ro.IsoLevel != pgx.Serializable {
		t.Errorf("isolation = %v, want serializable", ro.IsoLevel)
	}
	// A zero-value options set uses the server defaults (empty iso, read-write).
	def := pgxTxOptions(goruntime.TransactionOptions{})
	if def.AccessMode != "" || def.IsoLevel != "" {
		t.Errorf("default options should be empty, got %+v", def)
	}
}

func TestMapIsolation(t *testing.T) {
	cases := map[goruntime.IsolationLevel]pgx.TxIsoLevel{
		goruntime.IsolationDefault:        "",
		goruntime.IsolationReadCommitted:  pgx.ReadCommitted,
		goruntime.IsolationRepeatableRead: pgx.RepeatableRead,
		goruntime.IsolationSerializable:   pgx.Serializable,
	}
	for level, want := range cases {
		if got := mapIsolation(level); got != want {
			t.Errorf("mapIsolation(%v) = %v, want %v", level, got, want)
		}
	}
}

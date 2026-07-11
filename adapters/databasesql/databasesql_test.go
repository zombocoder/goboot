package databasesql

import (
	"database/sql"
	"testing"

	goruntime "github.com/zombocoder/goboot/runtime"
)

// The adapter must satisfy the runtime TransactionManager contract (§26.2).
var _ goruntime.TransactionManager = (*TransactionManager)(nil)

func TestMapIsolation(t *testing.T) {
	cases := map[goruntime.IsolationLevel]sql.IsolationLevel{
		goruntime.IsolationDefault:        sql.LevelDefault,
		goruntime.IsolationReadCommitted:  sql.LevelReadCommitted,
		goruntime.IsolationRepeatableRead: sql.LevelRepeatableRead,
		goruntime.IsolationSerializable:   sql.LevelSerializable,
	}
	for in, want := range cases {
		if got := mapIsolation(in); got != want {
			t.Errorf("mapIsolation(%v) = %v, want %v", in, got, want)
		}
	}
}

func TestSQLTxOptions(t *testing.T) {
	opts := sqlTxOptions(goruntime.TransactionOptions{ReadOnly: true, Isolation: goruntime.IsolationSerializable})
	if !opts.ReadOnly || opts.Isolation != sql.LevelSerializable {
		t.Errorf("sqlTxOptions = %+v", opts)
	}
}

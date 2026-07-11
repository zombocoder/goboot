package db

import (
	"context"
	"testing"
)

// fakeDBTX is a stand-in DBTX used to distinguish the pool from a transaction.
type fakeDBTX struct{ name string }

func (fakeDBTX) QueryContext(context.Context, string, ...any) (Rows, error) { return nil, nil }
func (fakeDBTX) QueryRowContext(context.Context, string, ...any) Row        { return nil }
func (fakeDBTX) ExecContext(context.Context, string, ...any) (Result, error) {
	return nil, nil
}

func TestProviderUsesPoolByDefault(t *testing.T) {
	pool := fakeDBTX{name: "pool"}
	p := NewProvider(pool)
	got := p.DB(context.Background())
	if got.(fakeDBTX).name != "pool" {
		t.Errorf("expected pool, got %v", got)
	}
}

func TestProviderPrefersActiveTx(t *testing.T) {
	pool := fakeDBTX{name: "pool"}
	tx := fakeDBTX{name: "tx"}
	p := NewProvider(pool)
	ctx := WithTx(context.Background(), tx)
	got := p.DB(ctx)
	if got.(fakeDBTX).name != "tx" {
		t.Errorf("expected active transaction, got %v", got)
	}
}

func TestActiveTx(t *testing.T) {
	if _, ok := ActiveTx(context.Background()); ok {
		t.Error("no transaction should be active on a bare context")
	}
	ctx := WithTx(context.Background(), fakeDBTX{name: "tx"})
	if tx, ok := ActiveTx(ctx); !ok || tx.(fakeDBTX).name != "tx" {
		t.Errorf("ActiveTx = %v, %v", tx, ok)
	}
}

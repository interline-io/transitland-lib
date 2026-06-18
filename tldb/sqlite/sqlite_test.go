//go:build cgo
// +build cgo

package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/interline-io/transitland-lib/tldb/tldbtest"
)

func TestSQLiteAdapter(t *testing.T) {
	adapter := &SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	tldbtest.AdapterTest(context.TODO(), t, adapter)
}

func TestSQLiteAdapter_NestedTx(t *testing.T) {
	adapter := &SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	if err := adapter.Open(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.Create(); err != nil {
		t.Fatal(err)
	}
	// Outer Tx should commit; inner Tx should reuse the same transaction
	outerCalled := false
	innerCalled := false
	err := adapter.Tx(func(outer Adapter) error {
		outerCalled = true
		return outer.Tx(func(inner Adapter) error {
			innerCalled = true
			return nil
		})
	})
	if err != nil {
		t.Fatal(err)
	}
	if !outerCalled || !innerCalled {
		t.Fatal("expected both outer and inner callbacks to be called")
	}

	// Inner error should propagate without double-rollback
	err = adapter.Tx(func(outer Adapter) error {
		return outer.Tx(func(inner Adapter) error {
			return errors.New("inner error")
		})
	})
	if err == nil || err.Error() != "inner error" {
		t.Fatalf("expected inner error, got: %v", err)
	}
}

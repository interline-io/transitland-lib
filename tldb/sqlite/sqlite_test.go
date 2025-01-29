//go:build cgo
// +build cgo

package sqlite

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/tldb/tldbtest"
)

func TestSQLiteAdapter(t *testing.T) {
	adapter := &SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	tldbtest.AdapterTest(context.TODO(), t, adapter)
}

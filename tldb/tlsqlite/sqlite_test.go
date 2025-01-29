//go:build cgo
// +build cgo

package tldb

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/tldb/tldbtest"
)

func TestSQLiteAdapter(t *testing.T) {
	adapter := &SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	tldbtest.TestAdapter(context.TODO(), t, adapter)
}

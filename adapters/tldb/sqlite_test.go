// +build cgo

package tldb

import (
	"testing"
)

func init() {
	testAdapters["SQLiteAdapter-Memory"] = func() Adapter { return &SQLiteAdapter{DBURL: "sqlite3://:memory:"} }
	testAdapters["SQLiteAdapter-Disk"] = func() Adapter { return &SQLiteAdapter{DBURL: "sqlite3://test.db"} }
}

func TestSQLiteAdapter(t *testing.T) {
	adapter := &SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	testAdapter(t, adapter)
}

// +build cgo

package gtdb

import (
	"testing"
)

func init() {
	testAdapters["SpatiaLiteAdapter-Memory"] = func() Adapter { return &SpatiaLiteAdapter{DBURL: "sqlite3://:memory:"} }
	testAdapters["SpatiaLiteAdapter-Disk"] = func() Adapter { return &SpatiaLiteAdapter{DBURL: "sqlite3://test.db"} }
}

func TestSpatiaLiteAdapter(t *testing.T) {
	adapter := &SpatiaLiteAdapter{DBURL: "sqlite3://:memory:"}
	testAdapter(t, adapter)
}

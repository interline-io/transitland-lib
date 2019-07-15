package gtdb

import (
	"testing"
)

func TestSpatiaLiteAdapter(t *testing.T) {
	dburl := "sqlite3://:memory:"
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	adapter := SpatiaLiteAdapter{DBURL: dburl}
	testAdapter(t, &adapter)
}

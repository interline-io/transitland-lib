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
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	if err := adapter.Create(); err != nil {
		t.Error(err)
	}
	testAdapter(t, &adapter)
}

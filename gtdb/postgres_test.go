package gtdb

import (
	"os"
	"testing"
)

func TestPostgresAdapter(t *testing.T) {
	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	adapter := PostgresAdapter{DBURL: dburl}
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	if err := adapter.Create(); err != nil {
		t.Error(err)
	}
	testAdapter(t, &adapter)
}

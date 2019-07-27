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
	testAdapter(t, &adapter)
}

package gtdb

import (
	"os"
	"testing"
)

func init() {
	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
	if dburl != "" {
		testAdapters["PostgresAdapter"] = func() Adapter { return &PostgresAdapter{DBURL: dburl} }
	}
}

func TestPostgresAdapter(t *testing.T) {
	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
	if dburl == "" {
		t.Skip("GOTRANSIT_TEST_POSTGRES_URL is not set")
		return
	}
	adapter := &PostgresAdapter{DBURL: dburl}
	testAdapter(t, adapter)
}

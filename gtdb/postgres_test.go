package gtdb

import (
	"os"
	"testing"
)

func init() {
	dburl := os.Getenv("TRANSITLAND_TEST_POSTGRES_URL")
	if dburl != "" {
		testAdapters["PostgresAdapter"] = func() Adapter { return &PostgresAdapter{DBURL: dburl} }
	}
}

func TestPostgresAdapter(t *testing.T) {
	dburl := os.Getenv("TRANSITLAND_TEST_POSTGRES_URL")
	if dburl == "" {
		t.Skip("TRANSITLAND_TEST_POSTGRES_URL is not set")
		return
	}
	adapter := &PostgresAdapter{DBURL: dburl}
	testAdapter(t, adapter)
}

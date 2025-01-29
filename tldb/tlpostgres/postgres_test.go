package tlpostgres

import (
	"context"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/tldb/tldbtest"
)

func TestPostgresAdapter(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL is not set")
		return
	}
	adapter := &PostgresAdapter{DBURL: dburl}
	tldbtest.AdapterTest(context.TODO(), t, adapter)
}

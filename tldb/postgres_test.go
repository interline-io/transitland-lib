package tldb

import (
	"context"
	"os"
	"testing"
)

func TestPostgresAdapter(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL is not set")
		return
	}
	adapter := &PostgresAdapter{DBURL: dburl}
	testAdapter(context.TODO(), t, adapter)
}

package tldb_test

import (
	"context"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertBorrowedAdapterSurvivesReaderClose reproduces the borrowed-adapter
// footgun: a Tx callback hands its adapter to a Reader that closes it, then
// keeps using the adapter. Close must leave a handle it did not open alone.
func assertBorrowedAdapterSurvivesReaderClose(t *testing.T, adapter tldb.Adapter) {
	ctx := context.Background()
	err := adapter.Tx(func(atx tldb.Adapter) error {
		r := &tldb.Reader{Adapter: atx}
		require.NoError(t, r.Open())
		require.NoError(t, r.Close())
		var n int
		return atx.Get(ctx, &n, "select 1")
	})
	assert.NoError(t, err)
}

func TestReaderClose_BorrowedPostgresAdapter(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL is not set")
	}
	adapter := &postgres.PostgresAdapter{DBURL: dburl}
	require.NoError(t, adapter.Open())
	defer adapter.Close()
	assertBorrowedAdapterSurvivesReaderClose(t, adapter)
}

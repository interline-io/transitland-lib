//go:build cgo

package tldb_test

import (
	"testing"

	"github.com/interline-io/transitland-lib/tldb/sqlite"
	"github.com/stretchr/testify/require"
)

func TestReaderClose_BorrowedSQLiteAdapter(t *testing.T) {
	adapter := &sqlite.SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	require.NoError(t, adapter.Open())
	defer adapter.Close()
	assertBorrowedAdapterSurvivesReaderClose(t, adapter)
}

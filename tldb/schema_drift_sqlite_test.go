//go:build cgo

package tldb_test

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/sqlite"
	"github.com/stretchr/testify/require"
)

// TestSchemaDrift_SQLite checks that every column each GTFS entity writes exists
// in the embedded SQLite schema (schema/sqlite/sqlite.sql), using an in-memory
// database. No external database is required.
func TestSchemaDrift_SQLite(t *testing.T) {
	assertEntityColumns(t, context.Background(), newSQLiteSchemaDB(t), sqliteColumns)
}

// TestSchemaDrift_MaterializedSQLite checks that every column the materialization
// projects exists in the embedded SQLite materialized active tables.
func TestSchemaDrift_MaterializedSQLite(t *testing.T) {
	assertMaterializedColumns(t, context.Background(), newSQLiteSchemaDB(t), sqliteColumns, false)
}

// newSQLiteSchemaDB opens an in-memory SQLite database and applies the embedded schema.
func newSQLiteSchemaDB(t *testing.T) tldb.Ext {
	adapter := &sqlite.SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	require.NoError(t, adapter.Open())
	t.Cleanup(func() { adapter.Close() })
	require.NoError(t, adapter.Create())
	return adapter.DBX()
}

// sqliteColumns returns the set of column names for a table via the
// pragma_table_info table-valued function with a bound argument.
func sqliteColumns(ctx context.Context, db tldb.Ext, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, "SELECT name FROM pragma_table_info(?)", table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cols := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		cols[name] = true
	}
	return cols, rows.Err()
}

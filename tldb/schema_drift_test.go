//go:build cgo

package tldb_test

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchemaDrift_SQLite asserts that every database column the writer expects
// for each GTFS entity actually exists in the embedded SQLite schema
// (schema/sqlite/sqlite.sql).
//
// The "expected" set is tldb.MapperCache.GetHeader(ent) — the exact column
// header the insert path builds from the struct's `db` tags. That means
// `db:"-"` fields and CSV-only fields are already excluded, and the embedded
// tt.BaseEntity bookkeeping columns (id, feed_version_id, ...) are included. If
// a struct gains a field but the schema doesn't, the writer's INSERT would fail
// at runtime; this moves that failure to a fast, DB-less CI check.
//
// This is deliberately a forward (subset) check: a table may legitimately carry
// extra bookkeeping columns the struct doesn't model, so we only assert that
// struct columns are a subset of table columns, not equality. Entities without
// a TableName (e.g. Shape, whose points are aggregated into gtfs_shapes via a
// separate DB representation) are skipped.
func TestSchemaDrift_SQLite(t *testing.T) {
	ctx := context.Background()
	adapter := &sqlite.SQLiteAdapter{DBURL: "sqlite3://:memory:"}
	require.NoError(t, adapter.Open())
	defer adapter.Close()
	require.NoError(t, adapter.Create())
	db := adapter.DBX()

	for _, ent := range gtfs.AllEntities() {
		tn, ok := ent.(tldb.HasTableName)
		if !ok {
			continue
		}
		table := tn.TableName()
		t.Run(table, func(t *testing.T) {
			want, err := tldb.MapperCache.GetHeader(ent)
			require.NoError(t, err)
			got, err := sqliteColumns(ctx, db, table)
			require.NoError(t, err)
			require.NotEmptyf(t, got, "table %q has no columns; is it missing from schema/sqlite/sqlite.sql?", table)
			for _, col := range want {
				assert.Containsf(t, got, col,
					"%T: db column %q is missing from sqlite table %q (add it to schema/sqlite/sqlite.sql)", ent, col, table)
			}
		})
	}
}

// sqliteColumns returns the set of column names for a table. It uses the
// pragma_table_info table-valued function with a bound argument, so the table
// name is passed as a parameter rather than interpolated.
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

package tldb_test

import (
	"context"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/feedstate"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// These tests guard against schema drift: every database column the writer
// expects for a GTFS entity, and every column the materialization projects, must
// exist in the live schema. The SQLite variants (cgo-gated, in-memory, in
// schema_drift_sqlite_test.go) run everywhere; the Postgres variants below run
// against the migrated test database when TL_TEST_DATABASE_URL is set (CI, or
// locally via testdata/test_setup.sh).

// columnLister returns the set of column names for a table from a live schema,
// abstracting over the SQLite (pragma) and Postgres (information_schema) sources.
type columnLister func(ctx context.Context, db tldb.Ext, table string) (map[string]bool, error)

// assertEntityColumns checks that every db column each GTFS entity writes exists
// in its table. The expected set is tldb.MapperCache.GetHeader(ent) — the exact
// header the insert path builds from the struct's `db` tags, so `db:"-"` and
// CSV-only fields are excluded and the embedded BaseEntity columns are included.
// It is a forward subset check; a table may carry extra bookkeeping columns the
// struct doesn't model. Entities without a TableName (e.g. Shape, aggregated into
// gtfs_shapes via a separate representation) are skipped.
func assertEntityColumns(t *testing.T, ctx context.Context, db tldb.Ext, cols columnLister) {
	for _, ent := range gtfs.AllEntities() {
		tn, ok := ent.(tldb.HasTableName)
		if !ok {
			continue
		}
		table := tn.TableName()
		t.Run(table, func(t *testing.T) {
			want, err := tldb.MapperCache.GetHeader(ent)
			require.NoError(t, err)
			got, err := cols(ctx, db, table)
			require.NoError(t, err)
			require.NotEmptyf(t, got, "table %q is missing from the schema", table)
			for _, col := range want {
				assert.Containsf(t, got, col, "%T: db column %q is missing from table %q", ent, col, table)
			}
		})
	}
}

// assertMaterializedColumns checks that every column the materialization projects
// exists in the materialized active table. The source of truth is
// feedstate.MaterializedTableFields — the same destination columns
// MaterializeFeedVersion inserts.
func assertMaterializedColumns(t *testing.T, ctx context.Context, db tldb.Ext, cols columnLister, spatial bool) {
	for table, fields := range feedstate.MaterializedTableFields(spatial) {
		t.Run(table, func(t *testing.T) {
			got, err := cols(ctx, db, table)
			require.NoError(t, err)
			require.NotEmptyf(t, got, "materialized table %q is missing from the schema", table)
			for col := range fields {
				assert.Containsf(t, got, col, "materialization projects column %q but it is missing from %q", col, table)
			}
		})
	}
}

// TestSchemaDrift_Postgres runs both drift checks against the migrated Postgres
// test database. It is skipped unless TL_TEST_DATABASE_URL is set. The Postgres
// path materializes with the simplified geometry, so spatial=true is used; the
// destination column set is identical regardless.
func TestSchemaDrift_Postgres(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	adapter := &postgres.PostgresAdapter{DBURL: dburl}
	require.NoError(t, adapter.Open())
	defer adapter.Close()
	db := adapter.DBX()
	t.Run("entities", func(t *testing.T) { assertEntityColumns(t, ctx, db, postgresColumns) })
	t.Run("materialized", func(t *testing.T) { assertMaterializedColumns(t, ctx, db, postgresColumns, true) })
}

// postgresColumns returns the set of column names for a table from
// information_schema, with the table name passed as a bound parameter.
func postgresColumns(ctx context.Context, db tldb.Ext, table string) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx,
		"SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1",
		table)
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

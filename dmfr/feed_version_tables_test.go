package dmfr

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

// TestImportedTablesCoverEntities checks that ImportedTables includes every GTFS
// entity's table; unimport and delete only clear listed tables.
func TestImportedTablesCoverEntities(t *testing.T) {
	imported := map[string]bool{}
	for _, table := range GetFeedVersionTables().ImportedTables() {
		imported[table] = true
	}
	for _, ent := range gtfs.AllEntities() {
		table := tldb.GetTableName(ent)
		if table == "" {
			// gtfs.Shape has no table; shapes are stored as service.ShapeLine rows.
			continue
		}
		assert.True(t, imported[table], "table %q is missing from ImportedTables", table)
	}
}

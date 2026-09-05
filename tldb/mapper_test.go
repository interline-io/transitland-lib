package tldb_test

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

// The csv and db mappers share the same tag cache, and Pathway.ReverseSignpostedAs
// carries a csv alias that is exactly the column name the db mapper derives. The
// insert header must still name the database column, not the GTFS one.
func TestMapperCacheHeaderIgnoresCsvAlias(t *testing.T) {
	header, err := tldb.MapperCache.GetHeader(&gtfs.Pathway{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, header, "reverse_signposted_as")
	assert.NotContains(t, header, "reversed_signposted_as")
}

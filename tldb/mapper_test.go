package tldb_test

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

// tlcsv and tldb keep separate tag caches that share one parser, and an alias is
// read from the options of the mapper's own tag. A csv alias must therefore stay
// out of the db field map: Pathway's alias is the column name the db mapper
// derives for that same field, so leaking it here would displace the real entry
// and drop the column from every insert.
func TestMapperCacheIgnoresCsvAlias(t *testing.T) {
	fmap := tldb.MapperCache.GetStructTagMap(&gtfs.Pathway{})
	for name, fi := range fmap {
		assert.False(t, fi.IsAlias(), "db field map should hold no aliases, got %q aliasing %q", name, fi.AliasOf)
	}
	header, err := tldb.MapperCache.GetHeader(&gtfs.Pathway{})
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, header, "reverse_signposted_as", "the db column name must still be written")
}

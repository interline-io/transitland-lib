package dmfr

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestFeedVersionFilenames(t *testing.T) {
	// A feed version, its import record and its stats are entities, each with a
	// file named for its table.
	ents := []interface {
		tt.Entity
		TableName() string
	}{
		&FeedVersion{},
		&FeedVersionImport{},
		&FeedVersionServiceWindow{},
		&FeedVersionServiceLevel{},
		&FeedVersionFileInfo{},
	}
	for _, ent := range ents {
		assert.Equal(t, ent.TableName()+".txt", ent.Filename())
	}
}

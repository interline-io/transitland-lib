package testutil

import (
	"testing"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/direct"
	"github.com/interline-io/transitland-lib/tt"
)

func TestDirectCopy(t *testing.T) {
	fe, reader := NewMinimalTestFeed()
	writer := direct.NewWriter()
	if err := DirectCopy(reader, writer); err != nil {
		t.Error(err)
	}
	TestReader(t, *fe, func() adapters.Reader {
		r, _ := writer.NewReader()
		return r
	})
}

func TestAllEntities(t *testing.T) {
	fe, reader := NewMinimalTestFeed()
	fetotal := 0
	for _, v := range fe.Counts {
		fetotal += v
	}
	count := 0
	AllEntities(reader, func(tt.Entity) { count++ })
	if count != fetotal {
		t.Errorf("got %d expect %d", count, fetotal)
	}
}

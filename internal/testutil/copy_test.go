package testutil

import (
	"testing"

	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/internal/mock"
)

func TestDirectCopy(t *testing.T) {
	fe, reader := NewMinimalTestFeed()
	writer := mock.NewWriter()
	if err := DirectCopy(reader, writer); err != nil {
		t.Error(err)
	}
	TestReader(t, *fe, func() tl.Reader {
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
	AllEntities(reader, func(tl.Entity) { count++ })
	if count != fetotal {
		t.Errorf("got %d expect %d", count, fetotal)
	}
}

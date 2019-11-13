package gtdb

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/testutil"
)

// Writer interface tests.
func TestWriter(t *testing.T) {
	for k, adapter := range testAdapters {
		fe, reader := testutil.NewMinimalTestFeed()
		t.Run(k, func(t *testing.T) {
			testutil.TestWriter(t, *fe, func() gotransit.Reader { return reader }, func() gotransit.Writer { return &Writer{Adapter: adapter()} })
		})
	}
}

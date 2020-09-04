package gtdb

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
)

// Writer interface tests.
func TestWriter(t *testing.T) {
	for k, adapter := range testAdapters {
		fe, reader := testutil.NewMinimalTestFeed()
		t.Run(k, func(t *testing.T) {
			testutil.TestWriter(t, *fe, func() tl.Reader { return reader }, func() tl.Writer { return &Writer{Adapter: adapter()} })
		})
	}
}

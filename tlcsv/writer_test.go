package tlcsv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
)

// Round trip Writer test.
func TestWriter(t *testing.T) {
	fe, reader := testutil.NewMinimalTestFeed()
	tmpdir, err := ioutil.TempDir("", "gtfs")
	if err != nil {
		t.Error(err)
	}
	writer, err := NewWriter(tmpdir)
	if err != nil {
		t.Error(err)
	}
	testutil.TestWriter(t, *fe, func() tl.Reader { return reader }, func() tl.Writer { return writer })
	// Clean up and double check
	if err := os.RemoveAll(tmpdir); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(tmpdir); !os.IsNotExist(err) {
		t.Error("did not remove temporary directory!", tmpdir)
	}
}

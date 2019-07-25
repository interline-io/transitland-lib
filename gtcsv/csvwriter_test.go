package gtcsv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

// Round trip test.
func TestWriter_NewReader(t *testing.T) {
	fe, reader := testutil.NewMinimalExpect()
	reader.Open()
	defer reader.Close()
	tmpdir, err := ioutil.TempDir("", "gtfs")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tmpdir)
	writer, err := NewWriter(tmpdir)
	if err != nil {
		t.Error(err)
		return
	}
	writer.Open()
	defer writer.Close()
	if err := testutil.DirectCopy(reader, writer); err != nil {
		t.Error(err)
	}
	r2, _ := writer.NewReader()
	testutil.CheckExpectEntities(t, *fe, r2)
}

package gtcsv

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/mock"
)

// Round trip test.
func TestWriter_NewReader(t *testing.T) {
	fe := mock.NewExampleExpect()
	fe.Reader.Open()
	defer fe.Reader.Close()

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
	mock.DirectCopy(fe.Reader, writer)
	r2, _ := writer.NewReader()
	mock.TestExpect(t, *fe, r2)
}

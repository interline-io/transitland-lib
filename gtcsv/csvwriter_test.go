package gtcsv

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

// Writer interface tests.
func TestWriter(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "gtfs")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tmpdir)
	writer, _ := NewWriter(tmpdir)
	writer.Open()
	writer.Create()
	writer.Delete()
	defer writer.Close()
	testutil.WriterTester(writer, t)
}

// Round trip test.
func TestWriter_NewReader(t *testing.T) {
	reader, err := NewReader("../testdata/example")
	if err != nil {
		t.Error(err)
		return
	}
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
	writer.Create()
	writer.Delete()
	defer writer.Close()
	testutil.WriterTesterRoundTrip(reader, writer, t)
}

// Writer specific tests.

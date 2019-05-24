package gtdb

import (
	"os"
	"testing"

	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/internal/testutil"
)

func filldb(writer *Writer) {
	r1, _ := gtcsv.NewReader("../testdata/example")
	r1.Open()
	defer r1.Close()

	cp := copier.NewCopier(r1, writer)
	cp.NormalizeServiceIDs = true
	cp.Copy()

}

// Reader interface tests.
func TestReader(t *testing.T) {
	t.Run("SpatiaLite", func(t *testing.T) {
		// dburl := os.Getenv("GOTRANSIT_TEST_SQLITE_URL")
		dburl := "sqlite3://:memory:"
		if len(dburl) == 0 {
			t.Skip()
			return
		}
		writer, _ := NewWriter(dburl)
		writer.Open()
		writer.Create()
		writer.Delete()
		defer writer.Close()
		filldb(writer)
		reader, _ := writer.NewReader()
		testutil.ReaderTester(reader, t)
	})
	t.Run("PostGIS", func(t *testing.T) {
		dburl := os.Getenv("GOTRANSIT_TEST_DB_URL")
		if len(dburl) == 0 {
			t.Skip()
			return
		}
		writer, _ := NewWriter(dburl)
		writer.Open()
		writer.Create()
		writer.Delete()
		defer writer.Close()
		filldb(writer)
		reader, _ := writer.NewReader()
		testutil.ReaderTester(reader, t)
	})
}

// Reader specific tests.

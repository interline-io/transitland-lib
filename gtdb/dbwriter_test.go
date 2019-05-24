package gtdb

import (
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

// Writer interface tests.
func TestWriter(t *testing.T) {
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
		testutil.WriterTester(writer, t)
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
		testutil.WriterTester(writer, t)
	})
}

// Writer Round Trip tests are handled by the copy operation in dbreader_test.go.

// Writer specific tests.

package gtdb

import (
	"os"
	"testing"

	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/internal/testutil"
)

// Writer interface tests.
func TestWriter_Postgres(t *testing.T) {
	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	adapter := SQLXAdapter{DBURL: dburl}
	writer := Writer{Adapter: &adapter}
	if err := writer.Open(); err != nil {
		t.Error(err)
	}
	if err := writer.Create(); err != nil {
		t.Error(err)
	}
	defer writer.Close()
	r1, _ := gtcsv.NewReader("../testdata/example")
	if _, err := writer.CreateFeedVersion(r1); err != nil {
		t.Error(err)
	}
	testutil.WriterTester(&writer, t)
}

func TestWriter_SpatiaLite(t *testing.T) {
	dburl := "sqlite3://:memory:"
	adapter := SpatiaLiteAdapter{DBURL: dburl}
	writer := Writer{Adapter: &adapter}
	if err := writer.Open(); err != nil {
		t.Error(err)
	}
	if err := writer.Create(); err != nil {
		t.Error(err)
	}
	defer writer.Close()
	r1, _ := gtcsv.NewReader("../testdata/example")
	if _, err := writer.CreateFeedVersion(r1); err != nil {
		t.Error(err)
	}
	testutil.WriterTester(&writer, t)
}

// Writer Round Trip tests are handled by the copy operation in dbreader_test.go.

// Writer specific tests.

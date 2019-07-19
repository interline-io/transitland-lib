package gtdb

import (
	"os"
	"testing"

	"github.com/interline-io/gotransit/gtcsv"
)

func writerTest(t *testing.T, adapter Adapter) {
	writer := Writer{Adapter: adapter}
	if err := writer.Open(); err != nil {
		t.Error(err)
	}
	if err := writer.Create(); err != nil {
		t.Error(err)
	}
	defer writer.Close()
	r1, err := gtcsv.NewReader("../testdata/example")
	if err != nil {
		t.Error(err)
	}
	if _, err := writer.CreateFeedVersion(r1); err != nil {
		t.Error(err)
	}
	// testutil.WriterTester(&writer, t)
}

// Writer interface tests.
func TestWriter_Postgres(t *testing.T) {
	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	adapter := PostgresAdapter{DBURL: dburl}
	writerTest(t, &adapter)
}

func TestWriter_SpatiaLite(t *testing.T) {
	dburl := "sqlite3://:memory:"
	adapter := SpatiaLiteAdapter{DBURL: dburl}
	writerTest(t, &adapter)
}

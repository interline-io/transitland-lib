package gtdb

import (
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
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
	fe, reader := testutil.NewMinimalExpect()
	if err := reader.Open(); err != nil {
		t.Error(err)
	}
	defer reader.Close()
	if _, err := writer.CreateFeedVersion(reader); err != nil {
		t.Error(err)
	}
	if err := testutil.DirectCopy(reader, &writer); err != nil {
		t.Error(err)
	}
	reader2, err := writer.NewReader()
	if err != nil {
		t.Error(err)
	}
	testutil.TestExpect(t, *fe, reader2)
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

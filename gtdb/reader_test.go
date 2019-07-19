package gtdb

import (
	"os"
	"testing"
)

func testReader(t *testing.T, adapter Adapter) {
	writer := Writer{Adapter: adapter}
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	if err := adapter.Create(); err != nil {
		t.Error(err)
	}
	adapter.Create()
	defer writer.Close()
	filldb(&writer)
	reader, err := writer.NewReader()
	_ = reader
	if err != nil {
		t.Error(err)
	}
	// testutil.ReaderTester(reader, t)
}

// Reader interface tests.

func TestReader_Postgres(t *testing.T) {
	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	adapter := PostgresAdapter{DBURL: dburl}
	testReader(t, &adapter)
}

func TestReader_SpatiaLite(t *testing.T) {
	dburl := "sqlite3://:memory:"
	adapter := SpatiaLiteAdapter{DBURL: dburl}
	testReader(t, &adapter)
}

package gtdb

import (
	"os"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

// Reader interface tests.

func TestReader_Postgres(t *testing.T) {
	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	adapter := SQLXAdapter{DBURL: dburl}
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	if err := adapter.Create(); err != nil {
		t.Error(err)
	}
	adapter.Create()
	writer := Writer{Adapter: &adapter}
	defer writer.Close()
	filldb(&writer)
	reader, err := writer.NewReader()
	if err != nil {
		t.Error(err)
	}
	testutil.ReaderTester(reader, t)
}

func TestReader_SpatiaLite(t *testing.T) {
	dburl := "sqlite3://:memory:"
	adapter := SpatiaLiteAdapter{DBURL: dburl}
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	if err := adapter.Create(); err != nil {
		t.Error(err)
	}
	adapter.Create()
	writer := Writer{Adapter: &adapter}
	defer writer.Close()
	filldb(&writer)
	reader, err := writer.NewReader()
	if err != nil {
		t.Error(err)
	}
	testutil.ReaderTester(reader, t)
}

package gtdb

import (
	"fmt"
	"testing"

	"github.com/interline-io/gotransit"
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

func TestReader_Sqlx(t *testing.T) {
	dburl := "postgres://localhost/tl?binary_parameters=no&sslmode=disable"
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	writer, _ := NewWriter(dburl)
	adapter := SQLXAdapter{DBURL: dburl}
	writer.Adapter = &adapter
	writer.Open()
	fv := gotransit.FeedVersion{}
	eid, err := adapter.Insert("feed_versions", &fv)
	if err != nil {
		panic(err)
	}
	writer.FeedVersionID = eid
	defer writer.Close()
	filldb(writer)
	reader, _ := writer.NewReader()
	testutil.ReaderTester(reader, t)
	for stop := range reader.Stops() {
		fmt.Println(stop.ID, stop.Coordinates())
	}
}

func TestReader_SpatiaLite(t *testing.T) {
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
	for stop := range reader.Stops() {
		fmt.Println(stop.ID, stop.Coordinates())
	}
}

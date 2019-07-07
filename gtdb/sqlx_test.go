package gtdb

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/testutil"
)

func TestSQLXAdapter(t *testing.T) {
	dburl := "postgres://localhost/tl?binary_parameters=yes&sslmode=disable"
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	writer, _ := NewWriter(dburl)
	adapter := SQLXAdapter{DBURL: dburl}
	writer.Adapter = &adapter
	writer.Open()
	defer writer.Close()

	// writer.Create()
	fv := gotransit.FeedVersion{}
	eid, err := Insert(adapter.db, "feed_versions", &fv)
	writer.FeedVersionID = eid
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	testutil.WriterTester(writer, t)
}

package gtdb

import (
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

// Reader interface tests.

// func TestReader_Postgres(t *testing.T) {
// 	dburl := os.Getenv("GOTRANSIT_TEST_POSTGRES_URL")
// 	if len(dburl) == 0 {
// 		t.Skip()
// 		return
// 	}
// 	writer, _ := NewWriter(dburl)
// 	adapter := SQLXAdapter{DBURL: dburl}
// 	writer.Adapter = &adapter
// 	writer.Open()
// 	fv := gotransit.FeedVersion{}
// 	fv.EarliestCalendarDate = time.Now()
// 	fv.LatestCalendarDate = time.Now()
// 	fv.FetchedAt = time.Now()

// 	eid, err := adapter.Insert(&fv)
// 	if err != nil {
// 		panic(err)
// 	}
// 	writer.FeedVersionID = eid
// 	defer writer.Close()
// 	filldb(writer)
// 	reader, _ := writer.NewReader()
// 	testutil.ReaderTester(reader, t)
// 	for stop := range reader.Stops() {
// 		fmt.Println(stop.ID, stop.Coordinates())
// 	}

// 	s := gotransit.Stop{}
// 	q, v, _ := adapter.Sqrl().Select("*").From("gtfs_stops").ToSql()
// 	if err := adapter.DBX().Get(&s, q, v...); err != nil {
// 		t.Error(err)
// 	}
// 	fmt.Printf("s: %#v\n", s)

// 	// if err := adapter.db.Get(&s, "SELECT * FROM gtfs_stops LIMIT 1"); err != nil {
// 	// 	t.Error(err)
// 	// }
// 	// a, b, err := adapter.Sqrl().Select("*").From("gtfs_stops").ToSql()
// 	// fmt.Println("a:", a)
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// adapter.db.Get(&s, a, b...)
// 	// fmt.Println("s:", s.ID, s)
// }

func TestReader_SpatiaLite(t *testing.T) {
	dburl := "sqlite3://:memory:"
	if len(dburl) == 0 {
		t.Skip()
		return
	}
	adapter := SpatiaLiteAdapter{DBURL: dburl}
	if err := adapter.Open(); err != nil {
		t.Error(err)
	}
	if err := adapter.Create(); err != nil {
		t.Error(err)
	}
	adapter.Create()

	writer := Writer{Adapter: &adapter}
	// if err := writer.Open(); err != nil {
	// 	t.Error(err)
	// }
	// if err := writer.Create(); err != nil {
	// 	t.Error(err)
	// }
	defer writer.Close()
	filldb(&writer)
	reader, _ := writer.NewReader()
	testutil.ReaderTester(reader, t)
	// for stop := range reader.Stops() {
	// 	fmt.Println(stop.ID, stop.Coordinates())
	// }
}

package importer

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

func TestImportFeedVersion(t *testing.T) {
	ctx := context.TODO()
	setup := func(atx tldb.Adapter, filename string) int {
		// Create FV
		fv := dmfr.FeedVersion{}
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fv.File = filename
		return testdb.ShouldInsert(t, atx, &fv)
	}
	t.Run("Success", func(t *testing.T) {
		testdb.TempSqlite(func(atx tldb.Adapter) error {
			fvid := setup(atx, testutil.ExampleZip.URL)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, &atx2, Options{Activate: true, FeedVersionID: fvid, Storage: "/"})
			if err != nil {
				t.Fatal(err)
			}
			// Check results
			fvi := dmfr.FeedVersionImport{}
			testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
			if fvi.Success != true {
				t.Errorf("expected success = true")
			}
			if fvi.ExceptionLog != "" {
				t.Errorf("expected empty, got %s", fvi.ExceptionLog)
			}
			if fvi.InProgress != false {
				t.Errorf("expected in_progress = false")
			}
			count := 0
			expstops := testutil.ExampleZip.Counts["stops.txt"]
			testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
			if count != expstops {
				t.Errorf("got %d stops, expect %d stops", count, expstops)
			}
			expfvistops := fvi.EntityCount["stops.txt"]
			if count != expfvistops {
				t.Errorf("got %d stops, expect %d stops", count, expfvistops)
			}
			return nil
		})
	})
	t.Run("Failed", func(t *testing.T) {
		fvid := 0
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			fvid = setup(atx, testpath.RelPath("testdata/gtfs-examples/does-not-exist"))
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, &atx2, Options{FeedVersionID: fvid, Storage: "/"})
			if err == nil {
				t.Errorf("expected an error, got none")
			}
			fvi := dmfr.FeedVersionImport{}
			testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
			if fvi.Success != false {
				t.Errorf("expected success = false")
			}
			if fvi.ExceptionLog == "" {
				t.Error("got no exception log error, expected something", fvi.ExceptionLog)
			}
			if fvi.InProgress != false {
				t.Errorf("expected in_progress = false")
			}
			return nil
		})
		if err != nil {
			t.Error(err)
		}
	})
}

func Test_iImportFeedVersionTx(t *testing.T) {
	ctx := context.TODO()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		// Create FV
		fv := dmfr.FeedVersion{File: testutil.ExampleZip.URL}
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fvid := testdb.ShouldInsert(t, atx, &fv)
		fv.ID = fvid // TODO: ?? Should be set by canSetID
		// Import
		fviresult, err := importFeedVersionTx(ctx, atx, fv, Options{Storage: "/"})
		if err != nil {
			t.Error(err)
		}
		// Check
		count := 0
		expstops := testutil.ExampleZip.Counts["stops.txt"]
		testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
		if count != expstops {
			t.Errorf("expect %d stops, got %d", count, expstops)
		}
		expstopcount := fviresult.EntityCount["stops.txt"]
		if count != expstopcount {
			t.Errorf("expect %d stops in fvi result, got %d", count, expstops)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

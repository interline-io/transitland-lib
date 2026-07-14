package importer

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testreader"
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
			fvid := setup(atx, testreader.ExampleZip.URL)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(&atx2), Options{Activate: true, FeedVersionID: fvid, Storage: "/"})
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
			// An import with no ImportSource set defaults to automatic.
			if fvi.ImportSource != dmfr.ImportSourceAutomatic {
				t.Errorf("expected import_source = %q, got %q", dmfr.ImportSourceAutomatic, fvi.ImportSource)
			}
			count := 0
			expstops := testreader.ExampleZip.Counts["stops.txt"]
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
	t.Run("ManualSource", func(t *testing.T) {
		testdb.TempSqlite(func(atx tldb.Adapter) error {
			fvid := setup(atx, testreader.ExampleZip.URL)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(&atx2), Options{FeedVersionID: fvid, Storage: "/", ImportSource: dmfr.ImportSourceManual})
			if err != nil {
				t.Fatal(err)
			}
			fvi := dmfr.FeedVersionImport{}
			testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
			if fvi.ImportSource != dmfr.ImportSourceManual {
				t.Errorf("expected import_source = %q, got %q", dmfr.ImportSourceManual, fvi.ImportSource)
			}
			return nil
		})
	})
	t.Run("Failed", func(t *testing.T) {
		fvid := 0
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			fvid = setup(atx, testpath.RelPath("testdata/gtfs-examples/does-not-exist"))
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(&atx2), Options{FeedVersionID: fvid, Storage: "/"})
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

// A failed import leaves behind the rows the copier already wrote: ImportFeedVersion neither
// rolls them back nor removes them, and the failed import record is what keeps them out of
// entity queries. TestImportFeedVersion/Failed cannot cover this: it fails on a missing file,
// before the copier writes anything.
func TestImportFeedVersion_FailedImportLeavesRows(t *testing.T) {
	ctx := context.TODO()
	// Not TempSqlite: that runs the callback inside a transaction, which is the thing this
	// import no longer has.
	atx := testdb.TempSqliteAdapter()
	fv := testdb.CreateTestFeedVersion(atx, testreader.ExampleZip.URL)

	// A negative threshold can never be met, and it is checked after the copier runs, so the
	// import fails with every entity already written.
	if _, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(atx), Options{
		FeedVersionID:  fv.ID,
		Storage:        "/",
		ErrorThreshold: map[string]float64{"*": -1},
	}); err == nil {
		t.Fatal("expected the import to fail on the error threshold")
	}

	// Entity queries gate on success and in_progress, so check them as stored.
	fvi := dmfr.FeedVersionImport{}
	testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fv.ID)
	if fvi.Success || fvi.InProgress {
		t.Errorf("stored import record is success=%v in_progress=%v, want false/false", fvi.Success, fvi.InProgress)
	}

	count := 0
	testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fv.ID)
	if count == 0 {
		t.Error("expected the failed import to leave its stops behind")
	}
}

func Test_importFeedVersion(t *testing.T) {
	ctx := context.TODO()
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		// Create FV
		fv := dmfr.FeedVersion{File: testreader.ExampleZip.URL}
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fvid := testdb.ShouldInsert(t, atx, &fv)
		fv.ID = fvid // TODO: ?? Should be set by canSetID
		// Import
		fviresult, err := importFeedVersion(ctx, feedmanager.NewDBFeedManager(atx), fv, Options{Storage: "/"})
		if err != nil {
			t.Error(err)
		}
		// Check
		count := 0
		expstops := testreader.ExampleZip.Counts["stops.txt"]
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

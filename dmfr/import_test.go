package dmfr

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestFindImportableFeeds(t *testing.T) {
	err := testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		f := caltrain(atx, "test")
		allfvids := []int{}
		for i := 0; i < 10; i++ {
			fv1 := testdb.ShouldInsert(t, atx, &tl.FeedVersion{FeedID: f.ID})
			allfvids = append(allfvids, fv1)
		}
		expfvids := allfvids[:5]
		for _, fvid := range allfvids[5:] {
			testdb.ShouldInsert(t, atx, &FeedVersionImport{FeedVersionID: fvid})
		}
		foundfvids, err := FindImportableFeeds(atx)
		if err != nil {
			t.Error(err)
		}
		if !testutil.CompareSliceInt(foundfvids, expfvids) {
			t.Errorf("%v != %v", foundfvids, expfvids)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestMainImportFeedVersion(t *testing.T) {
	setup := func(atx tldb.Adapter, filename string) int {
		// Create FV
		fv := tl.FeedVersion{}
		fv.File = filename
		return testdb.ShouldInsert(t, atx, &fv)
	}
	t.Run("Success", func(t *testing.T) {
		testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
			fvid := setup(atx, testutil.ExampleDir.URL)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := MainImportFeedVersion(&atx2, ImportOptions{FeedVersionID: fvid})
			if err != nil {
				t.Fatal(err)
			}
			// Check results
			fvi := FeedVersionImport{}
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
			expstops := testutil.ExampleDir.Counts["stops.txt"]
			testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
			if count != expstops {
				t.Errorf("expect %d stops, got %d", count, expstops)
			}
			expfvistops := fvi.EntityCount["stops.txt"]
			if count != expfvistops {
				t.Errorf("expect %d stops in fvi result, got %d", count, expfvistops)
			}
			return nil
		})
	})
	t.Run("Failed", func(t *testing.T) {
		fvid := 0
		err := testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
			fvid = setup(atx, "../testdata/does-not-exist")
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			_, err := MainImportFeedVersion(&atx2, ImportOptions{FeedVersionID: fvid})
			if err == nil {
				t.Errorf("expected an error, got none")
			}
			fvi := FeedVersionImport{}
			testdb.ShouldGet(t, atx, &fvi, "SELECT * FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
			if fvi.Success != false {
				t.Errorf("expected success = false")
			}
			explog := "file does not exist"
			if fvi.ExceptionLog != explog {
				t.Errorf("got %s expected %s", fvi.ExceptionLog, explog)
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

func TestImportFeedVersion(t *testing.T) {
	err := testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		// Create FV
		fv := tl.FeedVersion{File: testutil.ExampleZip.URL}
		fvid := testdb.ShouldInsert(t, atx, &fv)
		fv.ID = fvid // TODO: ?? Should be set by canSetID
		// Import
		fviresult, err := ImportFeedVersion(atx, fv, ImportOptions{})
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

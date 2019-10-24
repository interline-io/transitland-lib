package dmfr

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/testdb"
	"github.com/interline-io/gotransit/internal/testutil"
)

func TestFindImportableFeeds(t *testing.T) {
	err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		f := caltrain(atx, "test")
		allfvids := []int{}
		for i := 0; i < 10; i++ {
			fv1 := testdb.ShouldInsert(t, atx, &gotransit.FeedVersion{FeedID: f})
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
	setup := func(atx gtdb.Adapter, filename string) int {
		// Create FV
		fv := gotransit.FeedVersion{}
		fv.File = filename
		return testdb.ShouldInsert(t, atx, &fv)
	}
	t.Run("Success", func(t *testing.T) {
		testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
			fvid := setup(atx, testutil.ExampleDir.URL)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			err := MainImportFeedVersion(&atx2, fvid)
			if err != nil {
				t.Error(err)
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
			return nil
		})
	})
	t.Run("Failed", func(t *testing.T) {
		fvid := 0
		err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
			fvid = setup(atx, "../testdata/does-not-exist")
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			err := MainImportFeedVersion(&atx2, fvid)
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
	err := testdb.WithAdapterRollback(func(atx gtdb.Adapter) error {
		// Create FV
		fv := gotransit.FeedVersion{File: testutil.ExampleDir.URL}
		fvid := testdb.ShouldInsert(t, atx, &fv)
		// Import
		err := ImportFeedVersion(atx, fvid)
		if err != nil {
			t.Error(err)
		}
		// Check
		count := 0
		expstops := testutil.ExampleDir.Counts["stops.txt"]
		testdb.ShouldGet(t, atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
		if count != expstops {
			t.Errorf("expect %d stops, got %d", count, expstops)
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

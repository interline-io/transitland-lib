package unimporter

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tldb"
)

func setupImport(t *testing.T, atx tldb.Adapter) int {
	// Create FV
	fv := tl.FeedVersion{File: testutil.ExampleZip.URL}
	fv.EarliestCalendarDate = tt.NewDate(time.Now())
	fv.LatestCalendarDate = tt.NewDate(time.Now())
	fvid := testdb.ShouldInsert(t, atx, &fv)
	fv.ID = fvid
	// Import
	_, err := importer.ImportFeedVersion(atx, fv, importer.Options{FeedVersionID: fvid})
	if err != nil {
		t.Error(err)
	}
	return fv.ID
}

func TestUnimportSchedule(t *testing.T) {
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		// Note - it's difficult to test feed_version_gtfs_imports.schedule_removed
		// This test uses ImportFeedVersion because MainImportFeedVersion, which creates feed_version_gtfs_import records,
		// requires multiple transaction commits to run.
		fvid := setupImport(t, atx)
		if err := UnimportSchedule(atx, fvid); err != nil {
			t.Fatal(err)
		}
		tcs := []struct {
			table  string
			expect int
		}{
			{"gtfs_stops", 9},
			{"gtfs_trips", 0},
			{"gtfs_stop_times", 0},
		}
		for _, tc := range tcs {
			t.Run(tc.table, func(t *testing.T) {
				count := 0
				if err := atx.Sqrl().Select("count(*)").From(tc.table).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != tc.expect {
					t.Errorf("got %d trips, expected %d", count, tc.expect)
				}
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUnimportFeedVersion(t *testing.T) {
	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		fvid := setupImport(t, atx)
		// TODO: test ExtraTables option
		if err := UnimportFeedVersion(atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		tcs := []struct {
			table  string
			expect int
		}{
			{"gtfs_stops", 0},
			{"gtfs_trips", 0},
			{"gtfs_stop_times", 0},
		}
		for _, tc := range tcs {
			t.Run(tc.table, func(t *testing.T) {
				count := 0
				if err := atx.Sqrl().Select("count(*)").From(tc.table).Scan(&count); err != nil {
					t.Fatal(err)
				}
				if count != tc.expect {
					t.Errorf("got %d trips, expected %d", count, tc.expect)
				}
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

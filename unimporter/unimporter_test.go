package unimporter

import (
	"fmt"
	"os"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		fmt.Println("TL_TEST_DATABASE_URL is not set, skipping")
		return
	}
	os.Exit(m.Run())
}

func setupImport(t *testing.T, atx tldb.Adapter) int {
	// Create FV
	feed := tl.Feed{}
	feed.FeedID = fmt.Sprintf("feed-%d", time.Now().UnixNano())
	feedid := testdb.ShouldInsert(t, atx, &feed)
	fv := tl.FeedVersion{File: testutil.ExampleZip.URL}
	fv.FeedID = feedid
	fv.EarliestCalendarDate = tt.NewDate(time.Now())
	fv.LatestCalendarDate = tt.NewDate(time.Now())
	fvid := testdb.ShouldInsert(t, atx, &fv)
	fv.ID = fvid
	// Generate stats
	tlreader, err := tlcsv.NewReader(testutil.ExampleZip.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := stats.CreateFeedStats(atx, tlreader, fvid); err != nil {
		t.Fatal(err)
	}
	// Import
	if _, err := importer.ImportFeedVersion(atx, importer.Options{FeedVersionID: fvid, Storage: "/"}); err != nil {
		t.Fatal(err)
	}
	return fv.ID
}

func TestUnimportSchedule(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		// Note - it's difficult to test feed_version_gtfs_imports.schedule_removed
		fvid := setupImport(t, atx)
		if err := UnimportSchedule(atx, fvid); err != nil {
			t.Fatal(err)
		}
		tcs := []struct {
			table  string
			expect int
		}{
			{
				table:  "gtfs_stops",
				expect: 9,
			},
			{
				table:  "gtfs_trips",
				expect: 0,
			},
			{
				table:  "gtfs_stop_times",
				expect: 0,
			},
			{
				table:  "feed_version_stop_onestop_ids",
				expect: 9,
			},
			{
				table:  "tl_feed_version_geometries",
				expect: 1,
			},
			{
				table:  "feed_version_gtfs_imports",
				expect: 1,
			},
		}
		for _, tc := range tcs {
			t.Run(tc.table, func(t *testing.T) {
				count := 0
				if err := atx.Sqrl().Select("count(*)").From(tc.table).Where(sq.Eq{"feed_version_id": fvid}).Scan(&count); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.expect, count, tc.table)
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestUnimportFeedVersion(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(t, atx)
		// TODO: test ExtraTables option
		if err := UnimportFeedVersion(atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		tcs := []struct {
			table  string
			expect int
		}{
			{
				table:  "gtfs_stops",
				expect: 0,
			},
			{
				table:  "gtfs_trips",
				expect: 0,
			},
			{
				table:  "gtfs_stop_times",
				expect: 0,
			},
			{
				table:  "feed_version_stop_onestop_ids",
				expect: 9,
			},
			{
				table:  "tl_feed_version_geometries",
				expect: 0,
			},
			{
				table:  "feed_version_gtfs_imports",
				expect: 0,
			},
		}
		for _, tc := range tcs {
			t.Run(tc.table, func(t *testing.T) {
				count := 0
				if err := atx.Sqrl().Select("count(*)").From(tc.table).Where(sq.Eq{"feed_version_id": fvid}).Scan(&count); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.expect, count, tc.table)
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

func TestDeleteFeedVersion(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(t, atx)
		if err := DeleteFeedVersion(atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		tcs := []struct {
			table  string
			expect int
		}{
			{
				table:  "gtfs_stops",
				expect: 0,
			},
			{
				table:  "gtfs_trips",
				expect: 0,
			},
			{
				table:  "gtfs_stop_times",
				expect: 0,
			},
			{
				table:  "feed_version_stop_onestop_ids",
				expect: 0,
			},
			{
				table:  "feed_version_gtfs_imports",
				expect: 0,
			},
			{
				table:  "tl_feed_version_geometries",
				expect: 0,
			},
			{
				table:  "feed_version_gtfs_imports",
				expect: 0,
			},
		}
		for _, tc := range tcs {
			t.Run(tc.table, func(t *testing.T) {
				count := 0
				if err := atx.Sqrl().Select("count(*)").From(tc.table).Where(sq.Eq{"feed_version_id": fvid}).Scan(&count); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, tc.expect, count, tc.table)
				fvCount := 0
				if err := atx.Sqrl().Select("count(*)").From("feed_versions").Where(sq.Eq{"id": fvid}).Scan(&fvCount); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, 0, fvCount, "feed versions")
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

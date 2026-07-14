package importer

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		log.Infof("TL_TEST_DATABASE_URL is not set, skipping")
		return
	}
	os.Exit(m.Run())
}

func setupImport(ctx context.Context, t *testing.T, atx tldb.Adapter) int {
	// Create FV
	feed := dmfr.Feed{}
	feed.FeedID = fmt.Sprintf("feed-%d", time.Now().UnixNano())
	feedid := testdb.ShouldInsert(t, atx, &feed)
	fv := dmfr.FeedVersion{File: testreader.ExampleZip.URL}
	fv.FeedID = feedid
	fv.EarliestCalendarDate = tt.NewDate(time.Now())
	fv.LatestCalendarDate = tt.NewDate(time.Now())
	fvid := testdb.ShouldInsert(t, atx, &fv)
	fv.ID = fvid
	// Generate stats
	tlreader, err := tlcsv.NewReader(testreader.ExampleZip.URL)
	if err != nil {
		t.Fatal(err)
	}
	if err := stats.CreateFeedStats(ctx, atx, tlreader, fvid, stats.WriteOptions{}); err != nil {
		t.Fatal(err)
	}
	// Import
	if _, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(atx), Options{FeedVersionID: fvid, Storage: "/"}); err != nil {
		t.Fatal(err)
	}
	return fv.ID
}

// Unimporting one feed version must not touch another's rows. This is a real hazard rather
// than a truism: gtfs_stop_times is hash partitioned on feed_version_id, and the batched
// delete matches rows by ctid, which is only unique within a single partition. Without the
// feed_version_id predicate on the outer delete the planner cannot prune, so it probes every
// partition and removes any row that happens to share a ctid.
func TestUnimportFeedVersion_LeavesOtherFeedVersionsAlone(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		keepFvid := setupImport(ctx, t, atx)
		dropFvid := setupImport(ctx, t, atx)

		// Counted across every feed version, not just the two here. A ctid collision lands on
		// whichever rows happen to occupy the same offset in another partition, so scoping the
		// assertion to one feed version makes it depend on how full the partitions already are.
		total := func(table string) int {
			count := 0
			switch table {
			case "gtfs_stop_times":
				testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_stop_times")
			case "gtfs_trips":
				testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_trips")
			case "gtfs_stops":
				testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_stops")
			}
			return count
		}
		forFv := func(table string, fvid int) int {
			count := 0
			switch table {
			case "gtfs_stop_times":
				testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_stop_times WHERE feed_version_id = ?", fvid)
			case "gtfs_trips":
				testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_trips WHERE feed_version_id = ?", fvid)
			case "gtfs_stops":
				testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
			}
			return count
		}

		tables := []string{"gtfs_stop_times", "gtfs_trips", "gtfs_stops"}
		totalBefore := map[string]int{}
		dropBefore := map[string]int{}
		for _, table := range tables {
			totalBefore[table] = total(table)
			dropBefore[table] = forFv(table, dropFvid)
			assert.Greater(t, dropBefore[table], 0, "%s: nothing to unimport", table)
			assert.Greater(t, forFv(table, keepFvid), 0, "%s: nothing to lose", table)
		}

		if err := UnimportFeedVersion(ctx, atx, dropFvid, nil); err != nil {
			t.Fatal(err)
		}

		for _, table := range tables {
			assert.Equal(t, 0, forFv(table, dropFvid), "%s: unimported feed version should have no rows", table)
			assert.Equal(t, totalBefore[table]-dropBefore[table], total(table),
				"%s: unimport removed rows belonging to other feed versions", table)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestUnimportSchedule(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		// Note - it's difficult to test feed_version_gtfs_imports.schedule_removed
		fvid := setupImport(ctx, t, atx)
		if err := UnimportSchedule(ctx, atx, fvid); err != nil {
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
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)
		// TODO: test ExtraTables option
		if err := UnimportFeedVersion(ctx, atx, fvid, nil); err != nil {
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
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)
		if err := DeleteFeedVersion(ctx, atx, fvid, nil); err != nil {
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
				if err := atx.Sqrl().Select("count(*)").From("feed_versions").Where(sq.Eq{"id": fvid}).Where(sq.NotEq{"deleted_at": nil}).Scan(&fvCount); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, 1, fvCount, "feed versions")
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

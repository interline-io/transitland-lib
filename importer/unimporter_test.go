package importer

import (
	"context"
	"errors"
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

// setupFetch creates a feed version and the stats a fetch writes for it -- file infos, service
// levels, onestop ids -- without importing it.
func setupFetch(ctx context.Context, t *testing.T, atx tldb.Adapter) int {
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
	return fv.ID
}

func setupImport(ctx context.Context, t *testing.T, atx tldb.Adapter) int {
	fvid := setupFetch(ctx, t, atx)
	if _, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(atx), Options{FeedVersionID: fvid, Storage: "/"}); err != nil {
		t.Fatal(err)
	}
	return fvid
}

// Most feed versions are fetched and never imported, and delete is what reclaims them: their
// fetch-time rows are the bulk of the tables this is meant to keep from growing without bound.
// Nothing else removes those rows, so this path is delete doing its own job rather than tidying
// up after an unimport.
func TestDeleteFeedVersion_NeverImported(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupFetch(ctx, t, atx)

		// The fetch really did write the rows we are about to claim were removed.
		before := 0
		testdb.MustGet(atx, &before, "SELECT count(*) FROM feed_version_stop_onestop_ids WHERE feed_version_id = ?", fvid)
		if before == 0 {
			t.Fatal("expected the fetch to have written stop onestop ids")
		}

		if err := DeleteFeedVersion(ctx, atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{
			"feed_version_file_infos",
			"feed_version_service_levels",
			"feed_version_stop_onestop_ids",
			"feed_version_route_onestop_ids",
			"feed_version_agency_onestop_ids",
			"tl_feed_version_geohashes",
		} {
			count := 0
			if err := atx.Sqrl().Select("count(*)").From(table).Where(sq.Eq{"feed_version_id": fvid}).Scan(&count); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, count, table)
		}
		deleted := 0
		testdb.MustGet(atx, &deleted, "SELECT count(*) FROM feed_versions WHERE id = ? AND deleted_at IS NOT NULL", fvid)
		assert.Equal(t, 1, deleted, "feed version should be soft deleted")
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

// Delete refuses any feed version that still has an import record, whatever state it is in: the
// record may belong to an import in flight, whose rows the copier is still writing.
func TestDeleteFeedVersion_RequiresUnimport(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	tcs := []struct {
		name       string
		success    bool
		inProgress bool
	}{
		{"imported", true, false},
		{"import in flight", false, true},
		{"import failed", false, false},
		{"unimport interrupted", true, true},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
				fvid := setupImport(ctx, t, atx)
				if _, err := atx.Sqrl().
					Update("feed_version_gtfs_imports").
					Set("success", tc.success).
					Set("in_progress", tc.inProgress).
					Where(sq.Eq{"feed_version_id": fvid}).
					ExecContext(ctx); err != nil {
					t.Fatal(err)
				}
				if err := DeleteFeedVersion(ctx, atx, fvid, nil); !errors.Is(err, ErrFeedVersionImported) {
					t.Fatalf("expected ErrFeedVersionImported, got %v", err)
				}
				count := 0
				testdb.MustGet(atx, &count, "SELECT count(*) FROM feed_versions WHERE id = ? AND deleted_at IS NOT NULL", fvid)
				if count != 0 {
					t.Error("refused delete should not have soft deleted the feed version")
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

// Entity rows can outlive their import record -- a crash, or something deleting the record
// directly. Delete is terminal, so it must not leave them behind: the missing record is only a
// proxy for "not imported", and here the proxy is wrong.
func TestDeleteFeedVersion_OrphanedRows(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)
		// Lose the import record, keeping the entity rows.
		if _, err := atx.Sqrl().Delete("feed_version_gtfs_imports").
			Where(sq.Eq{"feed_version_id": fvid}).ExecContext(ctx); err != nil {
			t.Fatal(err)
		}
		stops := 0
		testdb.MustGet(atx, &stops, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fvid)
		if stops == 0 {
			t.Fatal("expected orphaned stops to exist")
		}

		if err := DeleteFeedVersion(ctx, atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
		for _, table := range []string{"gtfs_stops", "gtfs_trips", "gtfs_stop_times", "feed_version_stop_onestop_ids"} {
			count := 0
			if err := atx.Sqrl().Select("count(*)").From(table).Where(sq.Eq{"feed_version_id": fvid}).Scan(&count); err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 0, count, table)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteFeedVersion(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)
		// Unimport is now a precondition, not something delete does for you.
		if err := UnimportFeedVersion(ctx, atx, fvid, nil); err != nil {
			t.Fatal(err)
		}
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

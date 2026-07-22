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
	"github.com/interline-io/transitland-lib/validator"
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

// A stored validation report and its child/grandchild rows have NOT DEFERRABLE
// foreign keys onto tl_validation_reports and no feed_version_id column, so they used
// to block feed version delete. DeleteFeedVersion must clear the whole subtree.
func TestDeleteFeedVersion_ValidationReport(t *testing.T) {
	ctx := context.TODO()
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		fvid := setupImport(ctx, t, atx)

		// Report -> error group -> exemplar, plus the two RT stat children.
		report := validator.NewResult(time.Now(), time.Now())
		report.FeedVersionID = fvid
		reportID := testdb.ShouldInsert(t, atx, report)
		group := &validator.ValidationReportErrorGroup{
			ValidationReportID: reportID,
			Filename:           "stops.txt",
			Field:              "stop_id",
			ErrorType:          "test",
			ErrorCode:          "test",
			Count:              1,
		}
		groupID := testdb.ShouldInsert(t, atx, group)
		testdb.ShouldInsert(t, atx, &validator.ValidationReportErrorExemplar{
			ValidationReportErrorGroupID: groupID,
			Line:                         1,
			EntityID:                     "s-1",
			Value:                        "x",
			Message:                      "test",
		})
		testdb.ShouldInsert(t, atx, &validator.ValidationReportTripUpdateStat{ValidationReportID: reportID, AgencyID: "a", RouteID: "r"})
		testdb.ShouldInsert(t, atx, &validator.ValidationReportVehiclePositionStat{ValidationReportID: reportID, AgencyID: "a", RouteID: "r"})

		if err := DeleteFeedVersion(ctx, atx, fvid, nil); err != nil {
			t.Fatal(err)
		}

		for _, tc := range []struct {
			table string
			col   string
			id    int
		}{
			{"tl_validation_reports", "feed_version_id", fvid},
			{"tl_validation_report_error_groups", "validation_report_id", reportID},
			{"tl_validation_report_error_exemplars", "validation_report_error_group_id", groupID},
			{"tl_validation_trip_update_stats", "validation_report_id", reportID},
			{"tl_validation_vehicle_position_stats", "validation_report_id", reportID},
		} {
			t.Run(tc.table, func(t *testing.T) {
				count := 0
				if err := atx.Sqrl().Select("count(*)").From(tc.table).Where(sq.Eq{tc.col: tc.id}).Scan(&count); err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, 0, count, tc.table)
			})
		}
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

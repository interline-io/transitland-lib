package importer

import (
	"context"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/feedstate"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
	sq "github.com/irees/squirrel"
)

// UnimportSchedule removes schedule data for a feed version and updates the import record.
// stops, routes, agencies, pathways, levels are not affected.
// Note: calendars and calendar_dates MAY be deleted in future versions.
func UnimportSchedule(ctx context.Context, atx tldb.Adapter, id int) error {
	fvt := dmfr.GetFeedVersionTables()
	tables := fvt.ScheduleTables()
	for _, table := range tables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, false); err != nil {
			return err
		}
	}
	where := sq.Eq{"feed_version_id": id}
	if _, err := atx.Sqrl().Update("feed_version_gtfs_imports").Set("schedule_removed", true).Where(where).ExecContext(ctx); err != nil {
		return err
	}
	return nil
}

// UnimportFeedVersion unimports a feed version and removes the feed_version_gtfs_import record.
func UnimportFeedVersion(ctx context.Context, atx tldb.Adapter, id int, extraTables []string) error {
	fvt := dmfr.GetFeedVersionTables()

	// Allow extension tables to not exist
	var optTables []string
	optTables = append(optTables, extraTables...)
	optTables = append(optTables, fvt.GtfsExtTables...)
	for _, table := range optTables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, true); err != nil {
			return err
		}
	}

	// Required tables
	tables := []string{}
	tables = append(tables, fvt.ImportedTables()...)
	for _, table := range tables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, false); err != nil {
			return err
		}
	}

	// Remove fvgi
	where := sq.Eq{"feed_version_id": id}
	if _, err := atx.Sqrl().Delete("feed_version_gtfs_imports").Where(where).ExecContext(ctx); err != nil {
		return err
	}
	// Deactivate the feed version (handles both feed_states and materialized tables)
	manager := feedstate.NewManager(atx)
	if err := manager.DeactivateFeedVersion(ctx, id); err != nil {
		return err
	}

	return nil
}

// RemoveOnestopIds deletes the fetch-derived onestop_id rows (agency/route/stop)
// for a feed version. These only power AllowPrevious lookups; the feed version and
// its other data are unaffected. It deletes unconditionally: the caller selects which
// versions to remove (e.g. per the feed's onestop_id_retention_period) and must
// exclude the active and materialized versions.
func RemoveOnestopIds(ctx context.Context, atx tldb.Adapter, id int) error {
	// Mirrors the tables of stats.StatOnestopIDs.
	tables := []string{
		"feed_version_agency_onestop_ids",
		"feed_version_route_onestop_ids",
		"feed_version_stop_onestop_ids",
	}
	for _, table := range tables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, false); err != nil {
			return err
		}
	}
	return nil
}

func DeleteFeedVersion(ctx context.Context, atx tldb.Adapter, id int, extraTables []string) error {
	// Unimport feed version first
	if err := UnimportFeedVersion(ctx, atx, id, extraTables); err != nil {
		return err
	}

	// Required tables
	fvt := dmfr.GetFeedVersionTables()
	tables := []string{}
	tables = append(tables, extraTables...)
	tables = append(tables, fvt.AllTables()...)
	for _, table := range tables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, false); err != nil {
			return err
		}
	}

	// Soft delete feed version
	_, err := atx.Sqrl().
		Update("feed_versions").
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		Set("deleted_at", tt.NewTime(time.Now().UTC())).
		Exec()
	if err != nil {
		return err
	}
	return nil

}

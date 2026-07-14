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

// setImportInProgress flags the import record so entity queries stop returning this feed
// version's rows. Must be committed before anything is deleted.
func setImportInProgress(ctx context.Context, atx tldb.Adapter, id int) error {
	_, err := atx.Sqrl().
		Update("feed_version_gtfs_imports").
		Set("in_progress", true).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"feed_version_id": id}).
		ExecContext(ctx)
	return err
}

// UnimportSchedule removes schedule data for a feed version and updates the import record.
// stops, routes, agencies, pathways, levels are not affected.
// Note: calendars and calendar_dates MAY be deleted in future versions.
func UnimportSchedule(ctx context.Context, atx tldb.Adapter, id int) error {
	if err := setImportInProgress(ctx, atx, id); err != nil {
		return err
	}
	fvt := dmfr.GetFeedVersionTables()
	tables := fvt.ScheduleTables()
	for _, table := range tables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, false); err != nil {
			return err
		}
	}
	// Clearing in_progress last makes the feed version visible again, now without schedule.
	where := sq.Eq{"feed_version_id": id}
	if _, err := atx.Sqrl().
		Update("feed_version_gtfs_imports").
		Set("schedule_removed", true).
		Set("in_progress", false).
		Where(where).
		ExecContext(ctx); err != nil {
		return err
	}
	return nil
}

// UnimportFeedVersion unimports a feed version and removes the feed_version_gtfs_import record.
//
// Not transactional: the import record is flagged in_progress first, which hides the feed
// version, so the deletes need not be atomic; one transaction spanning every entity table would
// pin the xmin horizon and stall autovacuum database-wide. The deletes are idempotent, so a
// failure part way through leaves hidden rows for a later run to remove.
func UnimportFeedVersion(ctx context.Context, atx tldb.Adapter, id int, extraTables []string) error {
	if err := setImportInProgress(ctx, atx, id); err != nil {
		return err
	}
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

	// Deactivation (feed_states, then the materialized tables) and dropping the import record
	// commit together: the record is the only marker that this feed version still needs an
	// unimport, so it must not disappear unless the deactivation happened too.
	return atx.Tx(func(atx tldb.Adapter) error {
		if err := feedstate.NewManager(atx).DeactivateFeedVersion(ctx, id); err != nil {
			return err
		}
		_, err := atx.Sqrl().
			Delete("feed_version_gtfs_imports").
			Where(sq.Eq{"feed_version_id": id}).
			ExecContext(ctx)
		return err
	})
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

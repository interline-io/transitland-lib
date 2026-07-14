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

// setImportInProgress flags the import record so that entity queries stop returning this
// feed version's rows. Callers must commit this before deleting anything: it is what makes
// the partial state left behind by a non-transactional unimport unreachable, and what
// makes a crashed unimport safe to simply run again. The flag is cleared by whatever
// finishes the job -- a completed unimport deletes the record outright.
func setImportInProgress(ctx context.Context, atx tldb.Adapter, id int) error {
	_, err := atx.Sqrl().
		Update("feed_version_gtfs_imports").
		Set("in_progress", true).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"feed_version_id": id}).
		ExecContext(ctx)
	return err
}

// deleteFeedVersionEntities removes every row the import wrote for a feed version. The
// import record itself is left alone: an unimport deletes it afterwards, while a failed
// import keeps it to record the failure.
func deleteFeedVersionEntities(ctx context.Context, atx tldb.Adapter, id int, extraTables []string) error {
	fvt := dmfr.GetFeedVersionTables()
	// Extension tables are allowed not to exist.
	var optTables []string
	optTables = append(optTables, extraTables...)
	optTables = append(optTables, fvt.GtfsExtTables...)
	for _, table := range optTables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, true); err != nil {
			return err
		}
	}
	for _, table := range fvt.ImportedTables() {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, false); err != nil {
			return err
		}
	}
	return nil
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
func UnimportFeedVersion(ctx context.Context, atx tldb.Adapter, id int, extraTables []string) error {
	// Hide the feed version before touching its data. Everything below is idempotent, so a
	// failure part way through leaves rows that are invisible and that a later run removes.
	if err := setImportInProgress(ctx, atx, id); err != nil {
		return err
	}
	if err := deleteFeedVersionEntities(ctx, atx, id, extraTables); err != nil {
		return err
	}

	// Deactivating and dropping the import record commit together. Deactivating is a
	// multi-statement swap -- clear feed_states, then the materialized tables -- that must not
	// be observed half done. The import record goes last within the transaction, because it is
	// the only thing that marks this feed version as still needing an unimport: were it dropped
	// on its own and the process then died, nothing would select this feed version again to
	// finish deactivating it.
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

	// UnimportFeedVersion has already removed the extension and imported tables; deleting
	// AllTables() here would run those ~40 statements a second time for nothing. Only the
	// fetch-time stats and the system tables are left (the import record among them is
	// already gone, so its delete is a no-op).
	fvt := dmfr.GetFeedVersionTables()
	var tables []string
	tables = append(tables, fvt.FetchStatDerivedTables...)
	tables = append(tables, fvt.SystemTables...)
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

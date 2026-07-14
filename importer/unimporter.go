package importer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
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

// deleteTables removes a feed version's rows from each table. ifExists tolerates a missing table,
// which extension tables may be.
func deleteTables(ctx context.Context, atx tldb.Adapter, tables []string, id int, ifExists bool) error {
	for _, table := range tables {
		if err := stats.FeedVersionTableDelete(ctx, atx, table, id, ifExists); err != nil {
			return err
		}
	}
	return nil
}

// UnimportOptions sets options for unimporting a feed version.
type UnimportOptions struct {
	// ExtraTables are deleted alongside the feed version's own tables.
	ExtraTables []string
	// Force unimports a feed version whose import is in progress. An import that died mid-run is
	// indistinguishable from one still running, and this is how it gets cleaned up.
	Force bool
}

// ErrImportInProgress is returned for a feed version whose import is still running.
var ErrImportInProgress = errors.New("feed version import is in progress; unimport it with force to override")

// checkUnimportAllowed refuses a feed version whose import is in flight. The copier commits as it
// goes, so deleting under it would remove rows the import has already written, and the import
// would then finalize success = true over the hole.
//
// A feed version with no import record is allowed: its entity rows can outlive the record, and
// unimport is the only thing that will remove them.
func checkUnimportAllowed(ctx context.Context, atx tldb.Adapter, id int, force bool) error {
	if force {
		return nil
	}
	fvi, err := feedmanager.NewDBFeedManager(atx).GetFeedVersionImport(ctx, id)
	if err != nil {
		return err
	}
	if fvi != nil && !fvi.Success && fvi.InProgress {
		return fmt.Errorf("feed version %d: %w", id, ErrImportInProgress)
	}
	return nil
}

// UnimportSchedule removes schedule data for a feed version and updates the import record.
// stops, routes, agencies, pathways, levels are not affected.
// Note: calendars and calendar_dates MAY be deleted in future versions.
func UnimportSchedule(ctx context.Context, atx tldb.Adapter, id int, opts UnimportOptions) error {
	if err := checkUnimportAllowed(ctx, atx, id, opts.Force); err != nil {
		return err
	}
	if err := setImportInProgress(ctx, atx, id); err != nil {
		return err
	}
	fvt := dmfr.GetFeedVersionTables()
	if err := deleteTables(ctx, atx, fvt.ScheduleTables(), id, false); err != nil {
		return err
	}
	// Clearing in_progress last makes the feed version visible again, now without schedule.
	_, err := atx.Sqrl().
		Update("feed_version_gtfs_imports").
		Set("schedule_removed", true).
		Set("in_progress", false).
		Where(sq.Eq{"feed_version_id": id}).
		ExecContext(ctx)
	return err
}

// UnimportFeedVersion unimports a feed version and removes the feed_version_gtfs_import record.
func UnimportFeedVersion(ctx context.Context, atx tldb.Adapter, id int, opts UnimportOptions) error {
	if err := checkUnimportAllowed(ctx, atx, id, opts.Force); err != nil {
		return err
	}
	// Hiding the feed version in its own commit is what lets the deletes run without a
	// transaction: one spanning every entity table would pin the xmin horizon and stall
	// autovacuum database-wide. The deletes are idempotent, so a failure part way through leaves
	// hidden rows for a later run to remove.
	if err := setImportInProgress(ctx, atx, id); err != nil {
		return err
	}
	fvt := dmfr.GetFeedVersionTables()
	if err := deleteTables(ctx, atx, slices.Concat(opts.ExtraTables, fvt.GtfsExtTables), id, true); err != nil {
		return err
	}
	if err := deleteTables(ctx, atx, fvt.ImportedTables(), id, false); err != nil {
		return err
	}
	// Deactivation and dropping the import record commit together: the record is the only marker
	// that this feed version still needs an unimport, so it must not disappear unless the
	// deactivation happened too.
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

// ErrFeedVersionImported is returned for a feed version that still holds imported data.
var ErrFeedVersionImported = errors.New("feed version is still imported; unimport it first")

// CheckFeedVersionUnimported returns ErrFeedVersionImported if the feed version still has an
// import record, in any state.
func CheckFeedVersionUnimported(ctx context.Context, atx tldb.Adapter, id int) error {
	fvi, err := feedmanager.NewDBFeedManager(atx).GetFeedVersionImport(ctx, id)
	if err != nil {
		return err
	}
	if fvi != nil {
		return fmt.Errorf("feed version %d: %w", id, ErrFeedVersionImported)
	}
	return nil
}

// DeleteFeedVersion removes everything belonging to a feed version and soft deletes it. The feed
// version must already be unimported.
func DeleteFeedVersion(ctx context.Context, atx tldb.Adapter, id int, extraTables []string) error {
	if err := CheckFeedVersionUnimported(ctx, atx, id); err != nil {
		return err
	}
	// A missing import record is only a proxy for "not imported". One lost to a crash leaves
	// entity rows with nothing pointing at them, and can leave feed_states still pointing here.
	// This is the last thing to run, so it sweeps every table rather than trust the proxy.
	if err := feedstate.NewManager(atx).DeactivateFeedVersion(ctx, id); err != nil {
		return err
	}
	fvt := dmfr.GetFeedVersionTables()
	if err := deleteTables(ctx, atx, slices.Concat(extraTables, fvt.GtfsExtTables), id, true); err != nil {
		return err
	}
	if err := deleteTables(ctx, atx, fvt.AllTables(), id, false); err != nil {
		return err
	}
	// Soft delete feed version
	_, err := atx.Sqrl().
		Update("feed_versions").
		Where(sq.Eq{"id": id}).
		Where(sq.Eq{"deleted_at": nil}).
		Set("deleted_at", tt.NewTime(time.Now().UTC())).
		ExecContext(ctx)
	return err
}

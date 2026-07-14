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
//
// Only a successful record is flagged. A failed import is already hidden -- the gate requires
// success -- and flagging it would make it indistinguishable from an import in flight, which is
// the one state unimport refuses. An unimport interrupted after flagging one would then be
// locked out of retrying itself.
func setImportInProgress(ctx context.Context, atx tldb.Adapter, id int) error {
	_, err := atx.Sqrl().
		Update("feed_version_gtfs_imports").
		Set("in_progress", true).
		Set("updated_at", time.Now().UTC()).
		Where(sq.Eq{"feed_version_id": id}).
		Where(sq.Eq{"success": true}).
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

// ErrImportInProgress is returned for a feed version whose import record is marked in progress
// and not successful.
var ErrImportInProgress = errors.New("feed version import is in progress; unimport it with force to override")

// CheckUnimportAllowed refuses a feed version whose import is in flight. The copier commits as it
// goes, so deleting under it would remove rows the import has already written, and the import
// would then finalize success = true over the hole.
//
// A feed version with no import record is allowed: its entity rows can outlive the record, and
// unimport and delete are the only things that will remove them.
func CheckUnimportAllowed(ctx context.Context, atx tldb.Adapter, id int, force bool) error {
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

// UnimportFeedVersion unimports a feed version and removes the feed_version_gtfs_import record.
func UnimportFeedVersion(ctx context.Context, atx tldb.Adapter, id int, opts UnimportOptions) error {
	if err := CheckUnimportAllowed(ctx, atx, id, opts.Force); err != nil {
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
//
// Unlike the unimport, this runs in a transaction. Requiring a prior unimport is what makes that
// affordable: the imported tables are already empty, so a delete touches only the rows a fetch
// wrote, and the transaction stays short. Atomicity is worth having, because a delete that failed
// part way through would leave a feed version with no rows, no import record, and deleted_at still
// null -- which is exactly what the import command selects, so it would be imported all over again.
func DeleteFeedVersion(ctx context.Context, atx tldb.Adapter, id int, extraTables []string) error {
	return atx.Tx(func(atx tldb.Adapter) error {
		if err := CheckFeedVersionUnimported(ctx, atx, id); err != nil {
			return err
		}
		// A missing import record is only a proxy for "not imported": one lost to a crash leaves
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
	})
}

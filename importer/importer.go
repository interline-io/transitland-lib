package importer

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/tldb"
)

// Options sets various options for importing a feed.
type Options struct {
	FeedVersionID int
	Storage       string
	Activate      bool
	// ImportSource records whether the import was initiated automatically (by a
	// maintenance/queue process) or manually (by a user). See the
	// dmfr.ImportSource* constants. Defaults to "automatic" when empty.
	ImportSource string
	// ErrorThreshold sets the maximum error percentage (0-100) allowed per file.
	// The key is the filename (e.g., "stops.txt") or "*" for the default threshold.
	// If any file exceeds its threshold, the import is considered failed.
	// Example: {"*": 10, "stops.txt": 5} means 10% default, 5% for stops.txt.
	ErrorThreshold map[string]float64
	// AllowPartial imports partial feeds and skips the required minimum-entity check.
	AllowPartial bool
	copier.Options
}

// Result contains the results of a feed import.
type Result struct {
	FeedVersionImport dmfr.FeedVersionImport
}

// ActivateFeedVersion sets the feed version as active and refreshes materialized tables
func ActivateFeedVersion(ctx context.Context, atx tldb.Adapter, fvid int) error {
	return feedmanager.NewDBFeedManager(atx).ActivateFeedVersion(ctx, fvid)
}

// ImportFeedVersion creates the import record and runs the Copier. The FeedManager supplies
// the metadata bookkeeping and entity-write sink; pass feedmanager.NewDBFeedManager(adapter)
// for the database backend.
//
// Not transactional: the import record is committed first with in_progress = true, which
// hides the feed version from entity queries, and cleared only once everything is written.
func ImportFeedVersion(ctx context.Context, fm feedmanager.FeedManager, opts Options) (Result, error) {
	// Get FV
	importSource := opts.ImportSource
	if importSource == "" {
		importSource = dmfr.ImportSourceAutomatic
	}
	fvi := dmfr.FeedVersionImport{InProgress: true, ImportSource: importSource}
	fvi.FeedVersionID = opts.FeedVersionID
	fv, err := fm.GetFeedVersion(ctx, opts.FeedVersionID)
	if err != nil {
		return Result{FeedVersionImport: fvi}, err
	}
	// Check FVI
	if existing, err := fm.GetFeedVersionImport(ctx, fv.ID); err != nil {
		// Serious error
		return Result{FeedVersionImport: fvi}, err
	} else if existing != nil {
		// Reimporting means unimporting first, as a separate step: the import record left by a
		// crashed or failed import is what makes this a no-op.
		fvi.ExceptionLog = "FeedVersionImport record already exists, skipping"
		return Result{FeedVersionImport: fvi}, nil
	}
	// Create FVI
	if _, err := fm.CreateFeedVersionImport(ctx, &fvi); err != nil {
		// Serious error
		log.For(ctx).Error().Msgf("Error creating FeedVersionImport: %s", err.Error())
		return Result{FeedVersionImport: fvi}, err
	}
	// The copier writes without an enclosing transaction. One transaction spanning millions
	// of entity rows holds its snapshot open for the whole import, which pins the xmin
	// horizon and stops autovacuum reclaiming dead tuples across the entire database.
	// The import record above is already committed with in_progress = true, and entity
	// queries exclude such feed versions, so nothing written below is reachable until the
	// final update clears the flag.
	fviresult, errImport := importFeedVersion(ctx, fm, *fv, opts)
	if errImport == nil && opts.Activate {
		log.For(ctx).Info().Msgf("Activating feed version")
		if err := fm.ActivateFeedVersion(ctx, fv.ID); err != nil {
			errImport = fmt.Errorf("error activating feed version: %s", err.Error())
		}
	}
	if errImport != nil {
		// There is no transaction to roll back, so remove whatever was written. This is
		// best effort: the rows stay unreachable regardless, because the import is recorded
		// as failed, but nothing else would ever collect them.
		cleanupFailedImport(ctx, fm, fv.ID, nil)
		fvi.Success = false
		fvi.InProgress = false
		fvi.ExceptionLog = errImport.Error()
		if err := fm.UpdateFeedVersionImport(ctx, &fvi); err != nil {
			// Serious error
			log.For(ctx).Error().Msgf("Error saving FeedVersionImport: %s", err.Error())
			return Result{FeedVersionImport: fvi}, err
		}
		return Result{FeedVersionImport: fvi}, errImport
	}

	// Clearing in_progress is the last thing that happens, and the only thing that makes
	// this feed version's data visible to the API.
	log.For(ctx).Info().Msgf("Finalizing import")
	fviresult.ID = fvi.ID
	fviresult.CreatedAt = fvi.CreatedAt
	fviresult.FeedVersionID = fv.ID
	fviresult.ImportSource = fvi.ImportSource
	fviresult.ImportLevel = 4
	fviresult.Success = true
	fviresult.InProgress = false
	fviresult.ExceptionLog = ""
	if err := fm.UpdateFeedVersionImport(ctx, &fviresult); err != nil {
		// Serious error
		log.For(ctx).Error().Msgf("Error saving FeedVersionImport: %s", err.Error())
		return Result{FeedVersionImport: fviresult}, err
	}
	return Result{FeedVersionImport: fviresult}, nil
}

// hasAdapter is implemented by feed managers backed by a database. Import and unimport of
// existing data need the handle directly; managers without one persist nothing.
type hasAdapter interface {
	Adapter() tldb.Adapter
}

// cleanupFailedImport removes the entity rows an import wrote before it failed. Best effort:
// the rows are unreachable either way, since the import is recorded as failed, but nothing
// else would ever collect them.
func cleanupFailedImport(ctx context.Context, fm feedmanager.FeedManager, fvid int, extraTables []string) {
	ha, ok := fm.(hasAdapter)
	if !ok {
		return
	}
	if err := deleteFeedVersionEntities(ctx, ha.Adapter(), fvid, extraTables); err != nil {
		log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("could not clean up after failed import")
	}
}

type canSetAllowPartial interface {
	SetAllowPartial(bool)
}

// importFeedVersion runs the Copier from the feed version's reader into the entity sink,
// both vended by the FeedManager, returning the import counts. Writes are committed as
// they go; the caller keeps them unreachable by leaving the import record in_progress.
func importFeedVersion(ctx context.Context, fm feedmanager.FeedManager, fv dmfr.FeedVersion, opts Options) (dmfr.FeedVersionImport, error) {
	fvi := dmfr.FeedVersionImport{}
	fvi.FeedVersionID = fv.ID
	// Get Reader
	reader, err := fm.OpenReader(ctx, &fv, opts.Storage)
	if err != nil {
		return fvi, err
	}
	if r, ok := reader.(canSetAllowPartial); ok {
		r.SetAllowPartial(opts.AllowPartial)
	}
	if err := reader.Open(); err != nil {
		return fvi, err
	}
	defer reader.Close()

	// Non-settable options
	opts.Options.AllowEntityErrors = false
	opts.Options.AllowReferenceErrors = false
	opts.Options.NormalizeServiceIDs = true
	for _, b := range builders.DefaultImportBuilders() {
		opts.Options.AddExtension(b)
	}
	fvi.InProgress = false

	// Go
	cpResult, cpErr := copier.CopyWithOptions(ctx, reader, fm.EntityWriter(fv.ID), opts.Options)
	if cpErr != nil {
		return fvi, cpErr
	}
	if cpResult == nil {
		return fvi, fmt.Errorf("copier returned nil result")
	}

	// Check error threshold
	if len(opts.ErrorThreshold) > 0 {
		thresholdResult := cpResult.CheckErrorThreshold(opts.ErrorThreshold)
		if !thresholdResult.OK {
			var exceededFiles []string
			for fn, detail := range thresholdResult.Details {
				if !detail.OK {
					log.For(ctx).Error().Str("filename", fn).Float64("error_percent", detail.ErrorPercent).Float64("threshold", detail.Threshold).Int("error_count", detail.ErrorCount).Int("total_count", detail.TotalCount).Msg("file exceeded error threshold")
					exceededFiles = append(exceededFiles, fn)
				}
			}
			sort.Strings(exceededFiles)
			var errMsgs []string
			for _, fn := range exceededFiles {
				detail := thresholdResult.Details[fn]
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %.2f%% errors (threshold: %.2f%%)", fn, detail.ErrorPercent, detail.Threshold))
			}
			return fvi, fmt.Errorf("error threshold exceeded: %s", strings.Join(errMsgs, "; "))
		}
	}

	// Check required files have at least minimum entities (skipped for partial feeds).
	if !opts.AllowPartial {
		requiredMinEntities := map[string]int{"agency.txt": 1, "routes.txt": 1}
		minEntitiesResult := cpResult.CheckRequiredMinEntities(requiredMinEntities)
		if !minEntitiesResult.OK {
			var failedFiles []string
			for fn, detail := range minEntitiesResult.Details {
				if !detail.OK {
					log.For(ctx).Error().Str("filename", fn).Int("total_count", detail.TotalCount).Int("required", detail.Required).Msg("file did not meet required minimum entities")
					failedFiles = append(failedFiles, fn)
				}
			}
			sort.Strings(failedFiles)
			var errMsgs []string
			for _, fn := range failedFiles {
				detail := minEntitiesResult.Details[fn]
				errMsgs = append(errMsgs, fmt.Sprintf("%s: %d entities (required: %d)", fn, detail.TotalCount, detail.Required))
			}
			return fvi, fmt.Errorf("required minimum entities not met: %s", strings.Join(errMsgs, "; "))
		}
	}

	// Save feed version import
	counts := copyResultCounts(*cpResult)
	fvi.Success = true
	fvi.InterpolatedStopTimeCount = counts.InterpolatedStopTimeCount
	fvi.EntityCount = counts.EntityCount
	fvi.WarningCount = counts.WarningCount
	fvi.GeneratedCount = counts.GeneratedCount
	fvi.SkipEntityErrorCount = counts.SkipEntityErrorCount
	fvi.SkipEntityReferenceCount = counts.SkipEntityReferenceCount
	fvi.SkipEntityFilterCount = counts.SkipEntityFilterCount
	fvi.SkipEntityMarkedCount = counts.SkipEntityMarkedCount
	return fvi, nil
}

func copyResultCounts(result copier.Result) dmfr.FeedVersionImport {
	fvi := dmfr.NewFeedVersionImport()
	fvi.InterpolatedStopTimeCount = result.InterpolatedStopTimeCount
	for k, v := range result.EntityCount {
		fvi.EntityCount[k] = v
	}
	for k, v := range result.GeneratedCount {
		fvi.GeneratedCount[k] = v
	}
	for k, v := range result.SkipEntityErrorCount {
		fvi.SkipEntityErrorCount[k] = v
	}
	for k, v := range result.SkipEntityReferenceCount {
		fvi.SkipEntityReferenceCount[k] = v
	}
	for k, v := range result.SkipEntityFilterCount {
		fvi.SkipEntityFilterCount[k] = v
	}
	for k, v := range result.SkipEntityMarkedCount {
		fvi.SkipEntityMarkedCount[k] = v
	}
	for _, e := range result.Warnings {
		fvi.WarningCount[e.Filename] += e.Count
	}
	return *fvi
}

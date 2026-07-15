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

// ImportFeedVersion creates the import record and runs the Copier. Pass
// feedmanager.NewDBFeedManager(adapter) for the database backend.
//
// The copy is not transactional; a failure leaves the rows the copier wrote, hidden behind a
// failed import record. Activation, when requested, runs afterwards in its own transaction.
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
		// Any existing import record blocks a reimport -- including one left by a failed or
		// crashed import -- so unimport must run first.
		fvi.ExceptionLog = "FeedVersionImport record already exists, skipping"
		return Result{FeedVersionImport: fvi}, nil
	}
	// Create FVI
	if _, err := fm.CreateFeedVersionImport(ctx, &fvi); err != nil {
		// Serious error
		log.For(ctx).Error().Msgf("Error creating FeedVersionImport: %s", err.Error())
		return Result{FeedVersionImport: fvi}, err
	}
	// No enclosing transaction: one spanning millions of entity rows would hold its snapshot
	// open for the whole import, pinning the xmin horizon and stopping autovacuum from
	// reclaiming dead tuples database-wide.
	fviresult, errImport := importFeedVersion(ctx, fm, *fv, opts)

	// The import record is what hides a partial import, so it has to be written even when ctx is
	// already cancelled -- a client disconnecting mid-import is the common way that happens. On the
	// cancelled ctx these updates would fail too, leaving the record marked in progress forever:
	// invisible, and refused by unimport as an import still in flight.
	finishCtx := context.WithoutCancel(ctx)

	if errImport != nil {
		// Rows the copier already wrote stay in place; recording the import as failed keeps them
		// hidden from entity queries until an unimport removes them.
		fvi.Success = false
		fvi.InProgress = false
		fvi.ExceptionLog = errImport.Error()
		if err := fm.UpdateFeedVersionImport(finishCtx, &fvi); err != nil {
			// Serious error
			log.For(ctx).Error().Msgf("Error saving FeedVersionImport: %s", err.Error())
			return Result{FeedVersionImport: fvi}, err
		}
		return Result{FeedVersionImport: fvi}, errImport
	}

	// This update sets success and clears in_progress, which is what makes this feed version's
	// data visible.
	log.For(ctx).Info().Msgf("Finalizing import")
	fviresult.ID = fvi.ID
	fviresult.CreatedAt = fvi.CreatedAt
	fviresult.FeedVersionID = fv.ID
	fviresult.ImportSource = fvi.ImportSource
	fviresult.ImportLevel = 4
	fviresult.Success = true
	fviresult.InProgress = false
	fviresult.ExceptionLog = ""
	if err := fm.UpdateFeedVersionImport(finishCtx, &fviresult); err != nil {
		// The finalize did not persist, so the record is still success=false, in_progress=true:
		// the import stays hidden and needs an unimport. Report that state rather than the success
		// we failed to write.
		log.For(ctx).Error().Msgf("Error saving FeedVersionImport: %s", err.Error())
		fviresult.Success = false
		fviresult.InProgress = true
		return Result{FeedVersionImport: fviresult}, err
	}

	// Activation is its own transaction: a failure here leaves the feed version imported but not
	// active, rather than undoing a good import.
	if opts.Activate {
		log.For(ctx).Info().Msgf("Activating feed version")
		if err := fm.ActivateFeedVersion(ctx, fv.ID); err != nil {
			return Result{FeedVersionImport: fviresult}, fmt.Errorf("error activating feed version: %s", err.Error())
		}
	}
	return Result{FeedVersionImport: fviresult}, nil
}

type canSetAllowPartial interface {
	SetAllowPartial(bool)
}

// importFeedVersion runs the Copier from the feed version's reader into the entity sink,
// returning the import counts.
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

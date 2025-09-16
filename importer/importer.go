package importer

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// Options sets various options for importing a feed.
type Options struct {
	FeedVersionID int
	Storage       string
	Activate      bool
	copier.Options
}

// Result contains the results of a feed import.
type Result struct {
	FeedVersionImport dmfr.FeedVersionImport
}

// ActivateFeedVersion .
func ActivateFeedVersion(ctx context.Context, atx tldb.Adapter, feedId int, fvid int) error {
	// Check FeedState exists
	if _, err := stats.GetFeedState(ctx, atx, feedId); err != nil {
		return err
	}
	// sqlite3 only supports "UPDATE ... FROM" in versions 3.33 and higher
	_, err := atx.DBX().ExecContext(ctx, "UPDATE feed_states SET feed_version_id = $1 WHERE feed_id = (SELECT feed_id FROM feed_versions WHERE id = $2)", fvid, fvid)
	return err
}

func MainImportFeedVersion(ctx context.Context, adapter tldb.Adapter, opts Options) (Result, error) {
	return ImportFeedVersion(ctx, adapter, opts)
}

// ImportFeedVersion create FVI and run Copier inside a Tx.
func ImportFeedVersion(ctx context.Context, adapter tldb.Adapter, opts Options) (Result, error) {
	// Get FV
	fvi := dmfr.FeedVersionImport{InProgress: true}
	fvi.FeedVersionID = opts.FeedVersionID
	fv := dmfr.FeedVersion{}
	fv.ID = opts.FeedVersionID
	if err := adapter.Find(ctx, &fv); err != nil {
		return Result{FeedVersionImport: fvi}, err
	}
	// Check FVI
	checkfviid := 0
	if err := adapter.Get(ctx, &checkfviid, `SELECT id FROM feed_version_gtfs_imports WHERE feed_version_id = ?`, fv.ID); err == sql.ErrNoRows {
		// ok
	} else if err == nil {
		fvi.ExceptionLog = "FeedVersionImport record already exists, skipping"
		return Result{FeedVersionImport: fvi}, nil
	} else {
		// Serious error
		return Result{FeedVersionImport: fvi}, err
	}
	// Create FVI
	if fviid, err := adapter.Insert(ctx, &fvi); err == nil {
		// note: handle OK first
		fvi.ID = fviid
	} else {
		// Serious error
		log.For(ctx).Error().Msgf("Error creating FeedVersionImport: %s", err.Error())
		return Result{FeedVersionImport: fvi}, err
	}
	// Import
	fviresult := dmfr.FeedVersionImport{} // keep result
	errImport := adapter.Tx(func(atx tldb.Adapter) error {
		var err error
		fviresult, err = importFeedVersionTx(ctx, atx, fv, opts)
		if err != nil {
			return err
		}
		required := []string{"agency.txt", "stops.txt"}
		for _, fn := range required {
			if c := fviresult.EntityCount[fn]; c == 0 {
				return fmt.Errorf("failed to import any entities from required file '%s'", fn)
			}
		}
		// Update route_stops, agency_geometries, etc...
		log.For(ctx).Info().Msgf("Finalizing import")
		if opts.Activate {
			log.For(ctx).Info().Msgf("Activating feed version")
			if err := ActivateFeedVersion(ctx, atx, fv.FeedID, fv.ID); err != nil {
				return fmt.Errorf("error activating feed version: %s", err.Error())
			}
		}
		// Update FVI with results, inside tx
		fviresult.ID = fvi.ID
		fviresult.CreatedAt = fvi.CreatedAt
		fviresult.FeedVersionID = fv.ID
		fviresult.ImportLevel = 4
		fviresult.Success = true
		fviresult.InProgress = false
		fviresult.ExceptionLog = ""
		if err := atx.Update(ctx, &fviresult); err != nil {
			// Serious error
			log.For(ctx).Error().Msgf("Error saving FeedVersionImport: %s", err.Error())
			return err
		}
		return err
	})
	// FVI error handling has to be outside of above tx, which will have aborted
	if errImport != nil {
		fvi.Success = false
		fvi.InProgress = false
		fvi.ExceptionLog = errImport.Error()
		if err := adapter.Update(ctx, &fvi); err != nil {
			// Serious error
			log.For(ctx).Error().Msgf("Error saving FeedVersionImport: %s", err.Error())
			return Result{FeedVersionImport: fvi}, err
		}
		return Result{FeedVersionImport: fvi}, errImport
	}
	return Result{FeedVersionImport: fviresult}, nil
}

// importFeedVersion .
func importFeedVersionTx(ctx context.Context, atx tldb.Adapter, fv dmfr.FeedVersion, opts Options) (dmfr.FeedVersionImport, error) {
	fvi := dmfr.FeedVersionImport{}
	fvi.FeedVersionID = fv.ID
	// Get Reader
	tladapter, err := tlcsv.NewStoreAdapter(ctx, opts.Storage, fv.File, fv.Fragment.Val)
	if err != nil {
		return fvi, err
	}
	reader, err := tlcsv.NewReaderFromAdapter(tladapter)
	if err != nil {
		return fvi, err
	}
	if err := reader.Open(); err != nil {
		return fvi, err
	}
	defer reader.Close()

	// Get writer with existing tx
	writer := &tldb.Writer{Adapter: atx, FeedVersionID: fv.ID}

	// Non-settable options
	opts.Options.AllowEntityErrors = false
	opts.Options.AllowReferenceErrors = false
	opts.Options.NormalizeServiceIDs = true
	opts.Options.AddExtension(builders.NewRouteGeometryBuilder())
	opts.Options.AddExtension(builders.NewRouteStopBuilder())
	opts.Options.AddExtension(builders.NewRouteHeadwayBuilder())
	opts.Options.AddExtension(builders.NewConvexHullBuilder())
	opts.Options.AddExtension(builders.NewAgencyPlaceBuilder())
	fvi.InProgress = false

	// Go
	cpResult, cpErr := copier.CopyWithOptions(ctx, reader, writer, opts.Options)
	if cpErr != nil {
		return fvi, cpErr
	}
	if cpResult == nil {
		return fvi, fmt.Errorf("copier returned nil result")
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

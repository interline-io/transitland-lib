package dmfr

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// ImportOptions sets various options for importing a feed.
type ImportOptions struct {
	FeedVersionID        int
	Extensions           []string
	Directory            string
	S3                   string
	Activate             bool
	CreateMissingShapes  bool
	InterpolateStopTimes bool
}

// ImportResult contains the results of a feed import.
type ImportResult struct {
	FeedVersionImport FeedVersionImport
}

type canContext interface {
	Context() *causes.Context
}

func copyResultCounts(result copier.CopyResult) FeedVersionImport {
	fvi := FeedVersionImport{}
	fvi.EntityCount = EntityCounter{}
	fvi.WarningCount = EntityCounter{}
	fvi.GeneratedCount = EntityCounter{}
	fvi.SkipEntityErrorCount = EntityCounter{}
	fvi.SkipEntityReferenceCount = EntityCounter{}
	fvi.SkipEntityFilterCount = EntityCounter{}
	fvi.SkipEntityMarkedCount = EntityCounter{}
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
		fn := ""
		if a, ok := e.(canContext); ok {
			fn = a.Context().Filename
		}
		fvi.WarningCount[fn]++
	}
	return fvi
}

// ActivateFeedVersion .
func ActivateFeedVersion(atx tldb.Adapter, fvid int) error {
	// Ensure runs in a txn
	_, err := atx.DBX().Exec("SELECT activate_feed_version($1)", fvid)
	return err
}

// AfterFeedVersionImport .
func AfterFeedVersionImport(atx tldb.Adapter, fvid int) error {
	// Ensure runs in a txn
	_, err := atx.DBX().Exec("SELECT after_feed_version_import($1)", fvid)
	return err
}

// FindImportableFeeds .
func FindImportableFeeds(adapter tldb.Adapter) ([]int, error) {
	fvids := []int{}
	qstr, qargs, err := adapter.Sqrl().
		Select("feed_versions.id").
		From("feed_versions").
		LeftJoin("feed_version_gtfs_imports ON feed_versions.id = feed_version_gtfs_imports.feed_version_id").
		Where("feed_version_gtfs_imports.id IS NULL").
		ToSql()
	if err != nil {
		return fvids, err
	}
	if err = adapter.Select(&fvids, qstr, qargs...); err != nil {
		return fvids, err
	}
	return fvids, nil
}

// MainImportFeedVersion create FVI and run Copier inside a Tx.
func MainImportFeedVersion(adapter tldb.Adapter, opts ImportOptions) (ImportResult, error) {
	// Get FV
	fvi := FeedVersionImport{FeedVersionID: opts.FeedVersionID, InProgress: true}
	fv := tl.FeedVersion{ID: opts.FeedVersionID}
	if err := adapter.Find(&fv); err != nil {
		return ImportResult{FeedVersionImport: fvi}, err
	}
	// Check FVI
	checkfviid := 0
	if err := adapter.Get(&checkfviid, `SELECT id FROM feed_version_gtfs_imports WHERE feed_version_id = ?`, fv.ID); err == sql.ErrNoRows {
		// ok
	} else if err == nil {
		fvi.ExceptionLog = "FeedVersionImport record already exists, skipping"
		return ImportResult{FeedVersionImport: fvi}, nil
	} else {
		// Serious error
		return ImportResult{FeedVersionImport: fvi}, err
	}
	// Create FVI
	fvi.UpdateTimestamps()
	if fviid, err := adapter.Insert(&fvi); err == nil {
		// note: handle OK first
		fvi.ID = fviid
	} else {
		// Serious error
		log.Error("Error creating FeedVersionImport: %s", err.Error())
		return ImportResult{FeedVersionImport: fvi}, err
	}
	// Import
	fviresult := FeedVersionImport{} // keep result
	errImport := adapter.Tx(func(atx tldb.Adapter) error {
		var err error
		fviresult, err = ImportFeedVersion(atx, fv, opts)
		if err != nil {
			return err
		}
		required := []string{"agency.txt", "routes.txt", "stops.txt", "trips.txt", "stop_times.txt"}
		for _, fn := range required {
			if c := fviresult.EntityCount[fn]; c == 0 {
				return fmt.Errorf("failed to import any entities from required file '%s'", fn)
			}
		}
		// Update route_stops, agency_geometries, etc...
		log.Info("Finalizing import")
		if err := AfterFeedVersionImport(atx, fv.ID); err != nil {
			return fmt.Errorf("error finalizing import: %s", err.Error())
		}
		if opts.Activate {
			log.Info("Activating feed version")
			if err := ActivateFeedVersion(adapter, opts.FeedVersionID); err != nil {
				return fmt.Errorf("error activating feed version: %s", err.Error())
			}
		}
		// Update FVI with results, inside tx
		fviresult.ID = fvi.ID
		fviresult.CreatedAt = fvi.CreatedAt
		fviresult.FeedVersionID = opts.FeedVersionID
		fviresult.ImportLevel = 4
		fviresult.Success = true
		fviresult.InProgress = false
		fviresult.ExceptionLog = ""
		fviresult.UpdateTimestamps()
		if err := adapter.Update(&fviresult); err != nil {
			// Serious error
			log.Error("Error saving FeedVersionImport: %s", err.Error())
			return err
		}
		return err
	})
	// FVI error handling has to be outside of above tx, which will have aborted
	if errImport != nil {
		fvi.Success = false
		fvi.InProgress = false
		fvi.ExceptionLog = errImport.Error()
		fvi.UpdateTimestamps()
		if err := adapter.Update(&fvi); err != nil {
			// Serious error
			log.Error("Error saving FeedVersionImport: %s", err.Error())
			return ImportResult{FeedVersionImport: fvi}, err
		}
		return ImportResult{FeedVersionImport: fvi}, errImport
	}
	return ImportResult{FeedVersionImport: fviresult}, nil
}

// ImportFeedVersion .
func ImportFeedVersion(atx tldb.Adapter, fv tl.FeedVersion, opts ImportOptions) (FeedVersionImport, error) {
	fvi := FeedVersionImport{FeedVersionID: fv.ID}
	// Get Reader
	url := fv.File
	if opts.S3 != "" {
		url = opts.S3 + "/" + fv.File
	} else if opts.Directory != "" {
		url = filepath.Join(opts.Directory, fv.File)
	}
	urlsplit := strings.SplitN(fv.URL, "#", 2)
	if len(urlsplit) > 1 {
		url = url + "#" + urlsplit[1]
	}
	reader, err := tlcsv.NewReader(url)
	if err != nil {
		return fvi, err
	}
	if err := reader.Open(); err != nil {
		return fvi, err
	}
	defer reader.Close()
	// Get writer with existing tx
	writer := tldb.Writer{Adapter: atx, FeedVersionID: fv.ID}
	// Import, run in txn
	cp := copier.NewCopier(reader, &writer)
	for _, e := range opts.Extensions {
		ext, err := ext.GetExtension(e)
		if err != nil {
			panic("Extension not found")
		}
		cp.AddExtension(ext)
	}
	// Settable options
	cp.CreateMissingShapes = opts.CreateMissingShapes
	cp.InterpolateStopTimes = opts.InterpolateStopTimes
	// Non-settable options
	cp.AllowEntityErrors = false
	cp.AllowReferenceErrors = false
	cp.NormalizeServiceIDs = true
	// Go
	cpresult := cp.Copy()
	if cpresult == nil {
		return fvi, errors.New("copy result was nil")
	}
	if cpresult.WriteError != nil {
		return fvi, cpresult.WriteError
	}
	cpresult.DisplaySummary()
	counts := copyResultCounts(*cpresult)
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

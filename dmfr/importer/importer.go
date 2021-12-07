package importer

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/ext/builders"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

// Options sets various options for importing a feed.
type Options struct {
	FeedVersionID int
	Directory     string
	S3            string
	Activate      bool
	copier.Options
}

// Result contains the results of a feed import.
type Result struct {
	FeedVersionImport dmfr.FeedVersionImport
}

type canContext interface {
	Context() *causes.Context
}

func getFileLines(fn string) ([]string, error) {
	ret := []string{}
	file, err := os.Open(fn)
	if err != nil {
		return ret, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if t := scanner.Text(); t != "" {
			ret = append(ret, strings.TrimSpace(t))
		}
	}
	if err := scanner.Err(); err != nil {
		return ret, err
	}
	return ret, nil
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

// ActivateFeedVersion .
func ActivateFeedVersion(atx tldb.Adapter, fvid int) error {
	// sqlite3 only supports "UPDATE ... FROM" in versions 3.33 and higher
	_, err := atx.DBX().Exec("UPDATE feed_states SET feed_version_id = $1 WHERE feed_id = (SELECT feed_id FROM feed_versions WHERE id = $2)", fvid, fvid)
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
func MainImportFeedVersion(adapter tldb.Adapter, opts Options) (Result, error) {
	// Get FV
	fvi := dmfr.FeedVersionImport{FeedVersionID: opts.FeedVersionID, InProgress: true}
	fv := tl.FeedVersion{ID: opts.FeedVersionID}
	if err := adapter.Find(&fv); err != nil {
		return Result{FeedVersionImport: fvi}, err
	}
	// Check FVI
	checkfviid := 0
	if err := adapter.Get(&checkfviid, `SELECT id FROM feed_version_gtfs_imports WHERE feed_version_id = ?`, fv.ID); err == sql.ErrNoRows {
		// ok
	} else if err == nil {
		fvi.ExceptionLog = "FeedVersionImport record already exists, skipping"
		return Result{FeedVersionImport: fvi}, nil
	} else {
		// Serious error
		return Result{FeedVersionImport: fvi}, err
	}
	// Create FVI
	fvi.UpdateTimestamps()
	if fviid, err := adapter.Insert(&fvi); err == nil {
		// note: handle OK first
		fvi.ID = fviid
	} else {
		// Serious error
		log.Error("Error creating FeedVersionImport: %s", err.Error())
		return Result{FeedVersionImport: fvi}, err
	}
	// Import
	fviresult := dmfr.FeedVersionImport{} // keep result
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
		if opts.Activate {
			log.Info("Activating feed version")
			if err := ActivateFeedVersion(atx, opts.FeedVersionID); err != nil {
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
		if err := atx.Update(&fviresult); err != nil {
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
			return Result{FeedVersionImport: fvi}, err
		}
		return Result{FeedVersionImport: fvi}, errImport
	}
	return Result{FeedVersionImport: fviresult}, nil
}

// ImportFeedVersion .
func ImportFeedVersion(atx tldb.Adapter, fv tl.FeedVersion, opts Options) (dmfr.FeedVersionImport, error) {
	fvi := dmfr.FeedVersionImport{FeedVersionID: fv.ID}
	// Get Reader
	reader, err := tlcsv.NewReader(dmfrGetReaderURL(opts.S3, opts.Directory, fv.File, fv.SHA1))
	if err != nil {
		return fvi, err
	}
	if err := reader.Open(); err != nil {
		return fvi, err
	}
	defer reader.Close()

	// Get writer with existing tx
	writer := &tldb.Writer{Adapter: atx, FeedVersionID: fv.ID}

	// Create copier
	// Non-settable options
	opts.Options.AllowEntityErrors = false
	opts.Options.AllowReferenceErrors = false
	opts.Options.NormalizeServiceIDs = true
	cp, err := copier.NewCopier(reader, writer, opts.Options)
	if err != nil {
		return fvi, err
	}
	cp.AddExtension(builders.NewRouteGeometryBuilder())
	cp.AddExtension(builders.NewRouteStopBuilder())
	cp.AddExtension(builders.NewRouteHeadwayBuilder())
	cp.AddExtension(builders.NewConvexHullBuilder())
	cp.AddExtension(builders.NewOnestopIDBuilder())
	cp.AddExtension(builders.NewAgencyPlaceBuilder())

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

// dmfrGetReaderURL helps load a file from an S3 or Directory location
func dmfrGetReaderURL(s3 string, directory string, url string, sha1 string) string {
	if s3 != "" && sha1 != "" {
		url = fmt.Sprintf("%s/%s.zip", s3, sha1)
	} else if directory != "" {
		url = filepath.Join(directory, url)
	}
	urlsplit := strings.SplitN(url, "#", 2)
	if len(urlsplit) > 1 && !strings.HasSuffix(url, ".zip") {
		// Add fragment back only if fragment does not end in ".zip"
		url = url + "#" + urlsplit[1]
	}
	return url
}

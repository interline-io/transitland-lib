package dmfr

import (
	"errors"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

type canContext interface {
	Context() *causes.Context
}

func copyResultCounts(result copier.CopyResult) FeedVersionImport {
	fvi := FeedVersionImport{}
	fvi.EntityCount = EntityCounter{}
	fvi.ErrorCount = EntityCounter{}
	fvi.WarningCount = EntityCounter{}
	for k, v := range result.Count {
		fvi.EntityCount[k] = v
	}
	for _, e := range result.Errors {
		fn := ""
		if a, ok := e.(canContext); ok {
			fn = a.Context().Filename
		}
		fvi.ErrorCount[fn]++
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

// FindImportableFeeds .
func FindImportableFeeds(adapter gtdb.Adapter) ([]int, error) {
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
func MainImportFeedVersion(adapter gtdb.Adapter, fvid int) (FeedVersionImport, error) {
	// Get FV
	fvi := FeedVersionImport{FeedVersionID: fvid, InProgress: true}
	fv := gotransit.FeedVersion{ID: fvid}
	if err := adapter.Find(&fv); err != nil {
		return fvi, err
	}
	// Create FVI
	if fviid, err := adapter.Insert(&fvi); err == nil {
		// note: handle OK first
		fvi.ID = fviid // TODO: why isn't this set in insert?
	} else {
		// Serious error
		log.Info("Error creating FeedVersionImport: %s", err.Error())
		return fvi, err
	}
	// Import
	fviresult := FeedVersionImport{} // keep result
	errImport := adapter.Tx(func(atx gtdb.Adapter) error {
		var err error
		fviresult, err = ImportFeedVersion(atx, fv)
		// Update FVI with results, inside tx
		fviresult.ID = fvi.ID
		fviresult.FeedVersionID = fvid
		fviresult.ImportLevel = 4
		fviresult.Success = true
		fviresult.InProgress = false
		fviresult.ExceptionLog = ""
		if err := adapter.Update(&fviresult); err != nil {
			// Serious error
			log.Info("Error saving FeedVersionImport: %s", err.Error())
			return err
		}
		return err
	})
	// FVI error handling has to be outside of above tx, which will have aborted
	if errImport != nil {
		fvi.Success = false
		fvi.InProgress = false
		fvi.ExceptionLog = errImport.Error()
		if err := adapter.Update(&fvi); err != nil {
			// Serious error
			log.Info("Error saving FeedVersionImport: %s", err.Error())
			return fvi, err
		}
		return fvi, errImport
	}
	return fviresult, nil
}

// ImportFeedVersion .
func ImportFeedVersion(atx gtdb.Adapter, fv gotransit.FeedVersion) (FeedVersionImport, error) {
	fvi := FeedVersionImport{FeedVersionID: fv.ID}
	// Get Reader
	reader, err := gtcsv.NewReader(fv.File)
	if err != nil {
		return fvi, err
	}
	if err := reader.Open(); err != nil {
		return fvi, err
	}
	defer reader.Close()
	// Get writer with existing tx
	writer := gtdb.Writer{Adapter: atx, FeedVersionID: fv.ID}
	// Import, run in txn
	cp := copier.NewCopier(reader, &writer)
	cp.AllowEntityErrors = false
	cp.AllowReferenceErrors = false
	cp.NormalizeServiceIDs = true
	cpresult := cp.Copy()
	if cpresult == nil {
		return fvi, errors.New("copy result was nil")
	}
	counts := copyResultCounts(*cpresult)
	fvi.EntityCount = counts.EntityCount
	fvi.ErrorCount = counts.ErrorCount
	fvi.WarningCount = counts.WarningCount
	return fvi, nil
}

package dmfr

import (
	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// MainImportFeedVersion create FVI and run Copier inside a Tx. May panic on errors creating/updating FVI.
func MainImportFeedVersion(adapter gtdb.Adapter, fvid int) error {
	// Create FVI
	var err error
	fvi := FeedVersionImport{
		FeedVersionID: fvid,
		ImportLevel:   4, // back compat
		InProgress:    true,
		Success:       false,
	}
	fvi.ID, err = adapter.Insert(&fvi)
	if err != nil {
		panic(err) // Serious error
	}
	// Import
	err = adapter.Tx(func(atx gtdb.Adapter) error {
		return ImportFeedVersion(atx, fvid)
	})
	if err != nil {
		fvi.InProgress = false
		fvi.Success = false
		fvi.ExceptionLog = err.Error()
		errTx := adapter.Update(&fvi, "in_progress", "success", "exception_log")
		if errTx != nil {
			panic(err) // Serious error
		}
		return err
	}
	// Update with success
	fvi.Success = true
	fvi.InProgress = false
	fvi.ExceptionLog = ""
	err = adapter.Update(&fvi, "in_progress", "success")
	if err != nil {
		panic(err) // Serious error
	}
	return nil
}

// ImportFeedVersion .
func ImportFeedVersion(atx gtdb.Adapter, fvid int) error {
	// Get file
	fv := gotransit.FeedVersion{}
	fv.ID = fvid
	err := atx.Find(&fv)
	if err != nil {
		return err
	}
	// Get Reader
	reader, err := gtcsv.NewReader(fv.File)
	if err != nil {
		return err
	}
	if err := reader.Open(); err != nil {
		return err
	}
	defer reader.Close()
	// Get writer with existing tx
	writer := gtdb.Writer{Adapter: atx, FeedVersionID: fvid}
	// Import, run in txn
	cp := copier.NewCopier(reader, &writer)
	cp.AllowEntityErrors = false
	cp.AllowReferenceErrors = false
	cp.NormalizeServiceIDs = true
	result := cp.Copy()
	for _, cperr := range result.Errors {
		log.Info("Error: %s", cperr.Error())
	}
	for _, cperr := range result.Warnings {
		log.Info("Warning: %s", cperr.Error())
	}
	for k, v := range result.Count {
		log.Info("Imported %s: %d", k, v)
	}
	return nil
}

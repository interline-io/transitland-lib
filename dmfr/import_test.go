package dmfr

import (
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/gtdb"
)

func TestMainImportFeedVersion(t *testing.T) {
	setup := func(atx gtdb.Adapter, filename string) int {
		// Create FV
		fv := gotransit.FeedVersion{}
		fv.File = filename
		fvid, err := atx.Insert(&fv)
		if err != nil {
			t.Error(err)
		}
		return fvid
	}
	t.Run("Success", func(t *testing.T) {
		WithAdapterRollback(func(atx gtdb.Adapter) error {
			fvid := setup(atx, "../testdata/example")
			atx2 := AdapterIgnoreTx{Adapter: atx}
			err := MainImportFeedVersion(&atx2, fvid)
			if err != nil {
				t.Error(err)
			}
			// Check results
			fvi := FeedVersionImport{}
			err = atx.Get(&fvi, "SELECT * FROM feed_version_imports WHERE feed_version_id = ?", fvid)
			if err != nil {
				t.Error(err)
			}
			if fvi.Success != true {
				t.Errorf("expected success = true")
			}
			if fvi.ExceptionLog != "" {
				t.Errorf("expected empty, got %s", fvi.ExceptionLog)
			}
			if fvi.InProgress != false {
				t.Errorf("expected in_progress = false")
			}
			return nil
		})
	})
	t.Run("Failed", func(t *testing.T) {
		WithAdapterRollback(func(atx gtdb.Adapter) error {
			fvid := setup(atx, "../testdata/does-not-exist")
			atx2 := AdapterIgnoreTx{Adapter: atx}
			err := MainImportFeedVersion(&atx2, fvid)
			if err == nil {
				t.Errorf("expected an error, got none")
			}
			fvi := FeedVersionImport{}
			err = atx.Get(&fvi, "SELECT * FROM feed_version_imports WHERE feed_version_id = ?", fvid)
			if err != nil {
				t.Error(err)
			}
			if fvi.Success != false {
				t.Errorf("expected success = false")
			}
			explog := "file does not exist"
			if fvi.ExceptionLog != explog {
				t.Errorf("got %s expected %s", fvi.ExceptionLog, explog)
			}
			if fvi.InProgress != false {
				t.Errorf("expected in_progress = false")
			}
			return nil
		})
	})
}

func TestImportFeedVersion(t *testing.T) {
	err := WithAdapterRollback(func(atx gtdb.Adapter) error {
		// Create FV
		fv := gotransit.FeedVersion{}
		fvid, err := atx.Insert(&fv)
		if err != nil {
			t.Error(err)
		}
		// Import
		err = ImportFeedVersion(atx, fvid)
		if err != nil {
			t.Error(err)
		}
		// Check
		return nil
	})
	if err != nil {
		t.Error(err)
	}
}

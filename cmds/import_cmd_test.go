package cmds

import (
	"context"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/importer"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testreader"
	"github.com/interline-io/transitland-lib/internal/testutil"
)

// ImportCommand must hand its importer options to every job, not just a subset.
func TestImportCommand_PassesImporterOptions(t *testing.T) {
	ctx := context.TODO()
	runImport := func(t *testing.T, file string, opts importer.Options) error {
		atx := testdb.TempSqliteAdapter()
		fv := testdb.CreateTestFeedVersion(atx, file)
		opts.Storage = "/"
		cmd := ImportCommand{
			ImportJobs: []ImportJob{{FeedVersionID: fv.ID}},
			Workers:    1,
			Fail:       true,
			Adapter:    atx,
			Options:    opts,
		}
		return cmd.Run(ctx)
	}

	t.Run("allow partial", func(t *testing.T) {
		zipPath := testutil.ZipDirToTemp(t, testpath.RelPath("testdata/gtfs-examples/example-partial"))
		defer os.Remove(zipPath)
		if err := runImport(t, zipPath, importer.Options{}); err == nil {
			t.Fatal("expected partial feed to fail without AllowPartial")
		}
		if err := runImport(t, zipPath, importer.Options{AllowPartial: true}); err != nil {
			t.Fatalf("expected partial feed to import with AllowPartial, got: %s", err.Error())
		}
	})

	t.Run("error threshold", func(t *testing.T) {
		// A negative threshold can never be met, so the import fails only if it is applied.
		if err := runImport(t, testreader.ExampleZip.URL, importer.Options{}); err != nil {
			t.Fatalf("expected import without threshold to succeed, got: %s", err.Error())
		}
		if err := runImport(t, testreader.ExampleZip.URL, importer.Options{ErrorThreshold: map[string]float64{"*": -1}}); err == nil {
			t.Fatal("expected import to fail on the error threshold")
		}
	})
}

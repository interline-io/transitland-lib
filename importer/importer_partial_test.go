package importer

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/feedmanager"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// TestImportFeedVersion_Partial covers --allow-partial: a feed of only
// stops/levels/pathways fails the minimum-entity check normally, but imports
// successfully with AllowPartial.
func TestImportFeedVersion_Partial(t *testing.T) {
	ctx := context.TODO()
	zipPath := testutil.ZipDirToTemp(t, testpath.RelPath("testdata/gtfs-examples/example-partial"))
	defer os.Remove(zipPath)
	doImport := func(t *testing.T, allowPartial bool) (dmfr.FeedVersionImport, error) {
		var fvi dmfr.FeedVersionImport
		var runErr error
		testdb.TempSqlite(func(atx tldb.Adapter) error {
			fv := dmfr.FeedVersion{File: zipPath}
			fv.EarliestCalendarDate = tt.NewDate(time.Now())
			fv.LatestCalendarDate = tt.NewDate(time.Now())
			fvid := testdb.ShouldInsert(t, atx, &fv)
			atx2 := testdb.AdapterIgnoreTx{Adapter: atx}
			res, err := ImportFeedVersion(ctx, feedmanager.NewDBFeedManager(&atx2), Options{
				FeedVersionID: fvid, Storage: "/", AllowPartial: allowPartial,
			})
			fvi, runErr = res.FeedVersionImport, err
			return nil
		})
		return fvi, runErr
	}

	t.Run("rejected without allow-partial", func(t *testing.T) {
		if _, err := doImport(t, false); err == nil {
			t.Fatal("expected import to fail the minimum-entity check")
		}
	})

	t.Run("accepted with allow-partial", func(t *testing.T) {
		fvi, err := doImport(t, true)
		if err != nil {
			t.Fatalf("expected import to succeed, got: %s", err.Error())
		}
		if !fvi.Success {
			t.Error("expected import success")
		}
		if got := fvi.EntityCount["stops.txt"]; got != 4 {
			t.Errorf("stops.txt: got %d, want 4", got)
		}
		if got := fvi.EntityCount["levels.txt"]; got != 2 {
			t.Errorf("levels.txt: got %d, want 2", got)
		}
		if got := fvi.EntityCount["pathways.txt"]; got != 2 {
			t.Errorf("pathways.txt: got %d, want 2", got)
		}
	})
}

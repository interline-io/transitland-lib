package cmds

import (
	"context"
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// The unimport selector must skip a feed version whose import is in flight: the copier commits as
// it goes, so deleting under it would remove rows the import has already written.
func TestUnimportCommand_InProgress(t *testing.T) {
	ctx := context.TODO()

	// UnimportFeedVersion removes the import record last, so its presence afterwards says
	// whether the feed version was selected at all.
	setup := func(t *testing.T, success bool, inProgress bool) (tldb.Adapter, int) {
		atx := testdb.TempSqliteAdapter()
		fv := testdb.CreateTestFeedVersion(atx, "test.zip")
		testdb.CreateTestFeedVersionImport(atx, fv.ID, success, inProgress)
		return atx, fv.ID
	}
	imports := func(atx tldb.Adapter, fvid int) int {
		count := 0
		testdb.MustGet(atx, &count, "SELECT count(*) FROM feed_version_gtfs_imports WHERE feed_version_id = ?", fvid)
		return count
	}
	unimport := func(atx tldb.Adapter, fvid int, force bool) error {
		cmd := UnimportCommand{
			FVIDs:   []string{strconv.Itoa(fvid)},
			Workers: 1,
			Adapter: atx,
			Force:   force,
		}
		return cmd.Run(ctx)
	}

	tcs := []struct {
		name       string
		success    bool
		inProgress bool
		unimported bool
	}{
		{"imported", true, false, true},
		{"unimport interrupted", true, true, true},
		{"import in flight", false, true, false},
		{"import failed", false, false, true},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			atx, fvid := setup(t, tc.success, tc.inProgress)
			if err := unimport(atx, fvid, false); err != nil {
				t.Fatal(err)
			}
			got := imports(atx, fvid) == 0
			if got != tc.unimported {
				t.Errorf("unimported = %v, want %v (success=%v in_progress=%v)", got, tc.unimported, tc.success, tc.inProgress)
			}
		})
	}

	// --force is the way to clean up after an import that died mid-run, which is
	// indistinguishable from one still running.
	t.Run("import in flight, force", func(t *testing.T) {
		atx, fvid := setup(t, false, true)
		if err := unimport(atx, fvid, true); err != nil {
			t.Fatal(err)
		}
		if imports(atx, fvid) != 0 {
			t.Error("--force should have unimported it")
		}
	})

	// Entity rows can outlive their import record. Nothing else can reach them: the default
	// selector needs a record to match on, and the import selector reads a missing record as
	// "never imported" and would re-import on top of them.
	t.Run("no import record, force", func(t *testing.T) {
		atx := testdb.TempSqliteAdapter()
		fv := testdb.CreateTestFeedVersion(atx, "test.zip")

		// An orphaned entity row, with no import record pointing at it.
		stop := gtfs.Stop{StopID: tt.NewString("orphan"), StopName: tt.NewString("Orphan")}
		stop.FeedVersionID = fv.ID
		testdb.MustInsert(atx, &stop)

		if err := unimport(atx, fv.ID, false); err != nil {
			t.Fatal(err)
		}
		count := 0
		testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fv.ID)
		if count == 0 {
			t.Fatal("without --force the orphan should be unreachable, so this test proves nothing")
		}

		if err := unimport(atx, fv.ID, true); err != nil {
			t.Fatal(err)
		}
		testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fv.ID)
		if count != 0 {
			t.Errorf("--force should have removed the orphaned rows, %d remain", count)
		}
	})
}

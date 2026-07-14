package cmds

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// The refusal itself is importer.DeleteFeedVersion's, and is covered there against every import
// record state. This checks the command surfaces it rather than swallowing it, and that a feed
// version with no import record still deletes.
func TestDeleteCommand(t *testing.T) {
	ctx := context.TODO()

	setup := func(t *testing.T, imported bool) (tldb.Adapter, int) {
		atx := testdb.TempSqliteAdapter()
		feed := testdb.CreateTestFeed(atx, fmt.Sprintf("feed-%s", t.Name()))
		fv := dmfr.FeedVersion{SHA1: t.Name(), File: "test.zip"}
		fv.FeedID = feed.ID
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fv.ID = testdb.MustInsert(atx, &fv)
		if imported {
			fvi := dmfr.FeedVersionImport{Success: true}
			fvi.FeedVersionID = fv.ID
			testdb.MustInsert(atx, &fvi)
		}
		return atx, fv.ID
	}
	deleted := func(atx tldb.Adapter, fvid int) bool {
		count := 0
		testdb.MustGet(atx, &count, "SELECT count(*) FROM feed_versions WHERE id = ? AND deleted_at IS NOT NULL", fvid)
		return count > 0
	}
	del := func(atx tldb.Adapter, fvid int) error {
		cmd := DeleteCommand{FVID: fvid, Adapter: atx}
		return cmd.Run(ctx)
	}

	t.Run("still imported", func(t *testing.T) {
		atx, fvid := setup(t, true)
		if err := del(atx, fvid); err == nil {
			t.Fatal("expected the command to surface the refusal")
		}
		if deleted(atx, fvid) {
			t.Error("feed version was soft deleted anyway")
		}
	})

	// A dry run must report the refusal, not promise a delete that the real run declines.
	t.Run("still imported, dry run", func(t *testing.T) {
		atx, fvid := setup(t, true)
		cmd := DeleteCommand{FVID: fvid, Adapter: atx, DryRun: true}
		if err := cmd.Run(ctx); err == nil {
			t.Error("dry run reported it would delete a feed version that is still imported")
		}
	})

	t.Run("unimported", func(t *testing.T) {
		atx, fvid := setup(t, false)
		if err := del(atx, fvid); err != nil {
			t.Fatal(err)
		}
		if !deleted(atx, fvid) {
			t.Error("an unimported feed version should be deleted")
		}
	})

	// A nonexistent id must not slip past the import check as "not imported".
	t.Run("missing feed version", func(t *testing.T) {
		atx, fvid := setup(t, false)
		if err := del(atx, fvid+1000); err == nil {
			t.Error("expected an error for a feed version that does not exist")
		}
	})
}

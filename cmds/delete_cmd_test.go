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

// Delete refuses a feed version that still has an import record, whatever state that record is
// in. Unimport is the command that knows how to remove imported data safely -- and an import
// record may belong to an import still in flight, whose rows the copier is committing as delete
// runs.
func TestDeleteCommand_RequiresUnimport(t *testing.T) {
	ctx := context.TODO()

	setup := func(t *testing.T, fvi *dmfr.FeedVersionImport) (tldb.Adapter, int) {
		atx := testdb.TempSqliteAdapter()
		feed := testdb.CreateTestFeed(atx, fmt.Sprintf("feed-%s", t.Name()))
		fv := dmfr.FeedVersion{SHA1: t.Name(), File: "test.zip"}
		fv.FeedID = feed.ID
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fv.ID = testdb.MustInsert(atx, &fv)
		if fvi != nil {
			fvi.FeedVersionID = fv.ID
			testdb.MustInsert(atx, fvi)
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

	// Any import record blocks the delete, whatever its state.
	tcs := []struct {
		name string
		fvi  *dmfr.FeedVersionImport
	}{
		{"imported", &dmfr.FeedVersionImport{Success: true}},
		{"import in flight", &dmfr.FeedVersionImport{InProgress: true}},
		{"import failed", &dmfr.FeedVersionImport{}},
		{"unimport interrupted", &dmfr.FeedVersionImport{Success: true, InProgress: true}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			atx, fvid := setup(t, tc.fvi)
			if err := del(atx, fvid); err == nil {
				t.Fatal("expected delete to refuse a feed version that is still imported")
			}
			if deleted(atx, fvid) {
				t.Error("feed version was soft deleted anyway")
			}
		})
	}

	t.Run("unimported", func(t *testing.T) {
		atx, fvid := setup(t, nil)
		if err := del(atx, fvid); err != nil {
			t.Fatal(err)
		}
		if !deleted(atx, fvid) {
			t.Error("an unimported feed version should be deleted")
		}
	})

	// A nonexistent id must not slip past the import check as "not imported".
	t.Run("missing feed version", func(t *testing.T) {
		atx, fvid := setup(t, nil)
		if err := del(atx, fvid+1000); err == nil {
			t.Error("expected an error for a feed version that does not exist")
		}
	})
}

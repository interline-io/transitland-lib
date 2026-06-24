package feedmanager

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
)

// TestDBFeedManager proves the metadata transcriptions behave like the
// direct adapter calls they replace, against an in-memory sqlite tldb.Adapter.
// The adapter handed to the callback is already inside a transaction, so the
// manager is constructed with inTx=true.
func TestDBFeedManager(t *testing.T) {
	if err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		ctx := context.Background()
		fm := &DBFeedManager{adapter: atx, inTx: true}

		// Seed a feed + feed version.
		feedID, err := atx.Insert(ctx, &dmfr.Feed{FeedID: "test", Spec: "gtfs"})
		if err != nil {
			t.Fatalf("seed feed: %v", err)
		}
		fv := dmfr.FeedVersion{SHA1: "abc123", File: "feed.zip"}
		fv.FeedID = feedID
		fvid, err := atx.Insert(ctx, &fv)
		if err != nil {
			t.Fatalf("seed feed_version: %v", err)
		}

		// GetFeedVersion round-trips the seeded row.
		gotFV, err := fm.GetFeedVersion(ctx, fvid)
		if err != nil {
			t.Fatalf("GetFeedVersion: %v", err)
		}
		if gotFV.SHA1 != "abc123" {
			t.Errorf("GetFeedVersion sha1 = %q, want abc123", gotFV.SHA1)
		}

		// No import yet → (nil, nil), not an error.
		if imp, err := fm.GetFeedVersionImport(ctx, fvid); err != nil || imp != nil {
			t.Errorf("GetFeedVersionImport(none) = %v, %v; want nil, nil", imp, err)
		}

		// Create sets the new id on the struct and returns it.
		fvi := &dmfr.FeedVersionImport{InProgress: true, ImportSource: dmfr.ImportSourceManual}
		fvi.FeedVersionID = fvid
		id, err := fm.CreateFeedVersionImport(ctx, fvi)
		if err != nil {
			t.Fatalf("CreateFeedVersionImport: %v", err)
		}
		if id == 0 || fvi.ID != id {
			t.Errorf("CreateFeedVersionImport id = %d, fvi.ID = %d; want equal, nonzero", id, fvi.ID)
		}

		// Now the import is found.
		imp, err := fm.GetFeedVersionImport(ctx, fvid)
		if err != nil {
			t.Fatalf("GetFeedVersionImport: %v", err)
		}
		if imp == nil || imp.FeedVersionID != fvid || !imp.InProgress {
			t.Fatalf("GetFeedVersionImport = %+v; want fvid=%d in_progress=true", imp, fvid)
		}

		// Update writes the result fields.
		fvi.Success = true
		fvi.InProgress = false
		if err := fm.UpdateFeedVersionImport(ctx, fvi); err != nil {
			t.Fatalf("UpdateFeedVersionImport: %v", err)
		}
		imp, _ = fm.GetFeedVersionImport(ctx, fvid)
		if imp == nil || !imp.Success || imp.InProgress {
			t.Errorf("after update = %+v; want success=true in_progress=false", imp)
		}

		// WithTx on an in-tx manager joins the existing tx (re-entrant) and runs fn.
		ran := false
		if err := fm.WithTx(ctx, func(ctx context.Context, tx FeedManager) error {
			ran = true
			if tx != fm {
				t.Error("re-entrant WithTx should pass the same manager")
			}
			return nil
		}); err != nil {
			t.Fatalf("WithTx: %v", err)
		}
		if !ran {
			t.Error("WithTx did not run fn")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

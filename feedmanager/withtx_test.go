package feedmanager

import (
	"context"
	"errors"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
)

// TestDBFeedManager_WithTx exercises the real-transaction branch (inTx=false):
// a failing callback rolls back its writes, a succeeding one commits them.
func TestDBFeedManager_WithTx(t *testing.T) {
	adapter := testdb.TempSqliteAdapter()
	ctx := context.Background()

	feedID, err := adapter.Insert(ctx, &dmfr.Feed{FeedID: "test", Spec: "gtfs"})
	if err != nil {
		t.Fatalf("seed feed: %v", err)
	}
	fv := dmfr.FeedVersion{SHA1: "abc123", File: "feed.zip"}
	fv.FeedID = feedID
	fvid, err := adapter.Insert(ctx, &fv)
	if err != nil {
		t.Fatalf("seed feed_version: %v", err)
	}

	fm := NewDBFeedManager(adapter) // inTx=false → WithTx opens a real Tx

	newImport := func(ctx context.Context, tx FeedManager) error {
		fvi := &dmfr.FeedVersionImport{InProgress: true, ImportSource: dmfr.ImportSourceManual}
		fvi.FeedVersionID = fvid
		_, err := tx.CreateFeedVersionImport(ctx, fvi)
		return err
	}

	// Rollback: the callback creates an import then fails — nothing should persist.
	boom := errors.New("boom")
	err = fm.WithTx(ctx, func(ctx context.Context, tx FeedManager) error {
		if err := newImport(ctx, tx); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("WithTx returned %v, want boom", err)
	}
	if imp, _ := fm.GetFeedVersionImport(ctx, fvid); imp != nil {
		t.Fatal("rollback failed: import persisted after the tx returned an error")
	}

	// Commit: the callback succeeds — the import is durable.
	if err := fm.WithTx(ctx, newImport); err != nil {
		t.Fatalf("WithTx (commit): %v", err)
	}
	if imp, _ := fm.GetFeedVersionImport(ctx, fvid); imp == nil {
		t.Fatal("commit failed: import missing after a successful tx")
	}
}

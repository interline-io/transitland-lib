package stats

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestEnsureFeedState(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new feed state with public=true by default", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			fs, err := EnsureFeedState(ctx, atx, feed.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !fs.Public {
				t.Errorf("expected new feed state to be public by default, got public=%v", fs.Public)
			}
			if fs.FeedID != feed.ID {
				t.Errorf("expected FeedID=%d, got %d", feed.ID, fs.FeedID)
			}
			if fs.ID == 0 {
				t.Errorf("expected ID to be set after insert")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("returns existing feed state without modification", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create existing feed state with public=false
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: false}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)

			// EnsureFeedState should return existing state unchanged
			fs, err := EnsureFeedState(ctx, atx, feed.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.Public {
				t.Errorf("expected public to remain false, got %v", fs.Public)
			}
			if fs.ID != existingFs.ID {
				t.Errorf("expected same ID=%d, got %d", existingFs.ID, fs.ID)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			fs1, err := EnsureFeedState(ctx, atx, feed.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			fs2, err := EnsureFeedState(ctx, atx, feed.ID)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs1.ID != fs2.ID {
				t.Errorf("expected same ID on repeated calls, got %d and %d", fs1.ID, fs2.ID)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestSetFeedStatePublic(t *testing.T) {
	ctx := context.Background()

	t.Run("sets public=true on existing feed state", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create feed state with public=false
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: false}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)

			// Set to public
			err := SetFeedStatePublic(ctx, atx, feed.ID, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify in database
			var dbFs dmfr.FeedState
			testdb.MustGet(atx, &dbFs, "SELECT * FROM feed_states WHERE feed_id = ?", feed.ID)
			if !dbFs.Public {
				t.Errorf("expected public=true, got %v", dbFs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("sets public=false on existing feed state", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create feed state with public=true
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: true}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)

			// Set to private
			err := SetFeedStatePublic(ctx, atx, feed.ID, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify in database
			var dbFs dmfr.FeedState
			testdb.MustGet(atx, &dbFs, "SELECT * FROM feed_states WHERE feed_id = ?", feed.ID)
			if dbFs.Public {
				t.Errorf("expected public=false, got %v", dbFs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("no-op when value already matches", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create feed state with public=true
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: true}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)
			originalUpdatedAt := existingFs.UpdatedAt

			// Set to same value
			err := SetFeedStatePublic(ctx, atx, feed.ID, true)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify UpdatedAt hasn't changed
			var dbFs dmfr.FeedState
			testdb.MustGet(atx, &dbFs, "SELECT * FROM feed_states WHERE feed_id = ?", feed.ID)
			if !dbFs.UpdatedAt.Equal(originalUpdatedAt) {
				t.Errorf("expected UpdatedAt to remain unchanged when value matches")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("returns error if feed state does not exist", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")
			// Don't create feed state

			err := SetFeedStatePublic(ctx, atx, feed.ID, true)
			if err == nil {
				t.Error("expected error when feed state does not exist")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

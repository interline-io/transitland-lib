package stats

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestUpdateFeedStatePublic(t *testing.T) {
	ctx := context.Background()

	t.Run("new feed state with nil setPublic defaults to public=true", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			// Create a feed first
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Call with nil setPublic
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !fs.Public {
				t.Errorf("expected new feed state to be public by default, got public=%v", fs.Public)
			}
			if fs.FeedID != feed.ID {
				t.Errorf("expected FeedID=%d, got %d", feed.ID, fs.FeedID)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("new feed state with setPublic=true", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			setPublic := true
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, &setPublic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !fs.Public {
				t.Errorf("expected public=true, got %v", fs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("new feed state with setPublic=false", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			setPublic := false
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, &setPublic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.Public {
				t.Errorf("expected public=false, got %v", fs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("existing feed state with nil setPublic remains unchanged (public)", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create existing feed state with public=true
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: true}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)

			// Call with nil setPublic
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !fs.Public {
				t.Errorf("expected public to remain true, got %v", fs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("existing feed state with nil setPublic remains unchanged (private)", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create existing feed state with public=false
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: false}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)

			// Call with nil setPublic
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.Public {
				t.Errorf("expected public to remain false, got %v", fs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("existing feed state updated when setPublic differs (false to true)", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create existing feed state with public=false
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: false}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)

			// Call with setPublic=true
			setPublic := true
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, &setPublic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !fs.Public {
				t.Errorf("expected public to be updated to true, got %v", fs.Public)
			}

			// Verify in database
			var dbFs dmfr.FeedState
			testdb.MustGet(atx, &dbFs, "SELECT * FROM feed_states WHERE feed_id = ?", feed.ID)
			if !dbFs.Public {
				t.Errorf("expected database public to be true, got %v", dbFs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("existing feed state updated when setPublic differs (true to false)", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create existing feed state with public=true
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: true}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)

			// Call with setPublic=false
			setPublic := false
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, &setPublic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if fs.Public {
				t.Errorf("expected public to be updated to false, got %v", fs.Public)
			}

			// Verify in database
			var dbFs dmfr.FeedState
			testdb.MustGet(atx, &dbFs, "SELECT * FROM feed_states WHERE feed_id = ?", feed.ID)
			if dbFs.Public {
				t.Errorf("expected database public to be false, got %v", dbFs.Public)
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("existing feed state not updated when setPublic matches current value", func(t *testing.T) {
		err := testdb.TempSqlite(func(atx tldb.Adapter) error {
			feed := testdb.CreateTestFeed(atx, "test-feed")

			// Create existing feed state with public=true
			existingFs := dmfr.FeedState{FeedID: feed.ID, Public: true}
			existingFs.ID = testdb.MustInsert(atx, &existingFs)
			originalUpdatedAt := existingFs.UpdatedAt

			// Call with setPublic=true (same value)
			setPublic := true
			fs, err := UpdateFeedStatePublic(ctx, atx, feed.ID, &setPublic)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !fs.Public {
				t.Errorf("expected public to remain true, got %v", fs.Public)
			}

			// Verify UpdatedAt hasn't changed (no unnecessary update)
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
}

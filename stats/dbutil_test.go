package stats

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tt"
)

// gtfs_stops.parent_station references gtfs_stops, and the constraint is not deferrable, so
// Postgres checks it at the end of every statement. A batch that removed a station while the
// platforms hanging from it were still present would fail -- and since batches are taken in
// physical order, the same batch would be retried forever. Deleting a hierarchy with a batch
// size below the number of stops must still drain it.
//
// Postgres only: the sqlite schema declares the foreign key but never enables enforcement.
func TestFeedVersionTableDelete_StopHierarchy(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL not set")
	}
	ctx := context.Background()

	// One row per statement, so a boundary falls between every level of the hierarchy.
	defer func(n int) { feedVersionDeleteBatchSize = n }(feedVersionDeleteBatchSize)
	feedVersionDeleteBatchSize = 1

	err := testdb.TempPostgres(dburl, func(atx tldb.Adapter) error {
		// TempPostgres commits, so the fixture has to be unique per run.
		key := fmt.Sprintf("stop-hierarchy-%d", time.Now().UnixNano())
		feed := testdb.CreateTestFeed(atx, key)
		fv := dmfr.FeedVersion{SHA1: key, File: key + ".zip"}
		fv.FeedID = feed.ID
		fv.EarliestCalendarDate = tt.NewDate(time.Now())
		fv.LatestCalendarDate = tt.NewDate(time.Now())
		fv.ID = testdb.MustInsert(atx, &fv)

		// Parents are inserted first, so they occupy the earlier physical positions -- exactly
		// the rows a batch grabs first, and the ones that cannot go first.
		insertStop := func(stopID string, locationType int, parent *int) int {
			id := 0
			err := atx.Sqrl().
				Insert("gtfs_stops").
				Columns("feed_version_id", "stop_id", "stop_name", "location_type", "parent_station").
				Values(fv.ID, stopID, stopID, locationType, parent).
				Suffix("RETURNING id").
				QueryRowContext(ctx).
				Scan(&id)
			if err != nil {
				t.Fatal(err)
			}
			return id
		}
		station := insertStop("station", 1, nil)
		platform := insertStop("platform", 0, &station)
		insertStop("boarding-area", 4, &platform)

		if err := FeedVersionTableDelete(ctx, atx, "gtfs_stops", fv.ID, false); err != nil {
			t.Fatalf("deleting a stop hierarchy: %v", err)
		}
		count := 0
		testdb.MustGet(atx, &count, "SELECT count(*) FROM gtfs_stops WHERE feed_version_id = ?", fv.ID)
		if count != 0 {
			t.Errorf("%d stops remain, want 0", count)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// A feed version's rows are deleted in bounded batches, not one statement, and a batch
// removes only the rows it was asked for.
func TestFeedVersionTableDelete_Batched(t *testing.T) {
	ctx := context.Background()
	const table = "feed_version_service_levels"

	err := testdb.TempSqlite(func(atx tldb.Adapter) error {
		feed := testdb.CreateTestFeed(atx, "test-feed")
		newFv := func(sha1 string) dmfr.FeedVersion {
			fv := dmfr.FeedVersion{SHA1: sha1, File: sha1 + ".zip"}
			fv.FeedID = feed.ID
			fv.ID = testdb.MustInsert(atx, &fv)
			return fv
		}
		addRows := func(fvid int, n int) {
			for i := 0; i < n; i++ {
				fvsl := dmfr.FeedVersionServiceLevel{Monday: i}
				fvsl.FeedVersionID = fvid
				testdb.MustInsert(atx, &fvsl)
			}
		}
		count := func(fvid int) int {
			n := 0
			testdb.MustGet(atx, &n, "SELECT count(*) FROM feed_version_service_levels WHERE feed_version_id = ?", fvid)
			return n
		}

		fv := newFv("aaa")
		other := newFv("bbb")
		addRows(fv.ID, 5)
		addRows(other.ID, 3)

		// The limit is honored: one call removes exactly 2 of the 5.
		n, err := atx.DeleteFeedVersionBatch(ctx, table, fv.ID, 2)
		if err != nil {
			t.Fatal(err)
		}
		if n != 2 {
			t.Errorf("DeleteFeedVersionBatch removed %d rows, want 2", n)
		}
		if got := count(fv.ID); got != 3 {
			t.Errorf("after one batch: %d rows remain, want 3", got)
		}

		// And the loop drains the rest, leaving other feed versions alone.
		if err := FeedVersionTableDelete(ctx, atx, table, fv.ID, false); err != nil {
			t.Fatal(err)
		}
		if got := count(fv.ID); got != 0 {
			t.Errorf("after delete: %d rows remain, want 0", got)
		}
		if got := count(other.ID); got != 3 {
			t.Errorf("other feed version lost rows: %d remain, want 3", got)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

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

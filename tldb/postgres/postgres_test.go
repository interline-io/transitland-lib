package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/tldb/tldbtest"
)

func TestPostgresAdapter(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL is not set")
		return
	}
	adapter := &PostgresAdapter{DBURL: dburl}
	tldbtest.AdapterTest(context.TODO(), t, adapter)
}

// Tx joins an already open transaction rather than starting a second one, and only the
// outermost call commits. Import, unimport and feed version activation all rely on this to
// compose: activation opens its own transaction, and callers that already hold one expect it
// to fall inside theirs.
func TestPostgresAdapter_NestedTx(t *testing.T) {
	dburl := os.Getenv("TL_TEST_DATABASE_URL")
	if dburl == "" {
		t.Skip("TL_TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	adapter := &PostgresAdapter{DBURL: dburl}
	if err := adapter.Open(); err != nil {
		t.Fatal(err)
	}
	// Registered first so it runs last: cleanups are LIFO, and the row cleanup below still
	// needs an open adapter.
	t.Cleanup(func() { adapter.Close() })

	// Every other column has a default.
	countFeed := func(onestopID string) int {
		count := 0
		if err := adapter.Get(ctx, &count, "SELECT count(*) FROM current_feeds WHERE onestop_id = ?", onestopID); err != nil {
			t.Fatal(err)
		}
		return count
	}
	insertFeed := func(atx Adapter, onestopID string) error {
		_, err := atx.DBX().ExecContext(ctx, atx.DBX().Rebind("INSERT INTO current_feeds (onestop_id) VALUES (?)"), onestopID)
		return err
	}

	const rollbackID = "nested-tx-rollback"
	const commitID = "nested-tx-commit"
	t.Cleanup(func() {
		if _, err := adapter.DBX().ExecContext(ctx,
			adapter.DBX().Rebind("DELETE FROM current_feeds WHERE onestop_id IN (?, ?)"),
			rollbackID, commitID); err != nil {
			t.Fatal(err)
		}
	})

	// A write made by the inner Tx must not survive the outer one rolling back: if the inner
	// call had started and committed its own transaction, the row would still be there.
	err := adapter.Tx(func(outer Adapter) error {
		if err := outer.Tx(func(inner Adapter) error {
			return insertFeed(inner, rollbackID)
		}); err != nil {
			return err
		}
		return errors.New("outer fails after the inner Tx returned")
	})
	if err == nil {
		t.Fatal("expected the outer error to propagate")
	}
	if n := countFeed(rollbackID); n != 0 {
		t.Errorf("inner Tx committed independently: found %d rows, want 0", n)
	}

	// And the outermost call is what commits.
	if err := adapter.Tx(func(outer Adapter) error {
		return outer.Tx(func(inner Adapter) error {
			return insertFeed(inner, commitID)
		})
	}); err != nil {
		t.Fatal(err)
	}
	if n := countFeed(commitID); n != 1 {
		t.Errorf("outermost Tx did not commit: found %d rows, want 1", n)
	}
}

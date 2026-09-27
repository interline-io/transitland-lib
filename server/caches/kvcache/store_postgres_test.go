package kvcache_test

import (
	"context"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/caches/kvcache/storetest"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
)

func TestPostgresStore(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	db := testutil.MustOpenTestDB(t)
	storetest.Run(t, func(t *testing.T) kvcache.Store {
		// Clear conformance-suite rows from previous subtests.
		if _, err := db.Exec(db.Rebind("DELETE FROM tl_kv_cache WHERE key LIKE ?"), "storetest:%"); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(db.Rebind("DELETE FROM tl_kv_hash WHERE hash_key LIKE ?"), "storetest:%"); err != nil {
			t.Fatal(err)
		}
		return kvcache.NewPostgresStore(db)
	})
}

func TestPostgresStore_Sweep(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	ctx := context.Background()
	db := testutil.MustOpenTestDB(t)
	if _, err := db.Exec(db.Rebind("DELETE FROM tl_kv_cache WHERE key LIKE ?"), "sweeptest:%"); err != nil {
		t.Fatal(err)
	}
	store := kvcache.NewPostgresStore(db)

	assert.NoError(t, store.Set(ctx, "sweeptest:gone", []byte("x"), 20*time.Millisecond))
	assert.NoError(t, store.Set(ctx, "sweeptest:keep", []byte("y"), 0))

	// Once lapsed, the expired key reads as a miss (filter-on-read).
	assert.Eventually(t, func() bool {
		_, ok, _ := store.Get(ctx, "sweeptest:gone")
		return !ok
	}, 3*time.Second, 50*time.Millisecond)

	if _, err := store.Sweep(ctx); err != nil {
		t.Fatal(err)
	}
	// The expired row is physically deleted; the no-expiry row survives.
	var gone int
	if err := db.Get(&gone, db.Rebind("SELECT count(*) FROM tl_kv_cache WHERE key = ?"), "sweeptest:gone"); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, gone, "expired row should be swept")
	_, ok, err := store.Get(ctx, "sweeptest:keep")
	assert.NoError(t, err)
	assert.True(t, ok, "no-expiry row should survive the sweep")
}

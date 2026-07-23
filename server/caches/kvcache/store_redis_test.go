package kvcache_test

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/caches/kvcache/storetest"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
)

func TestRedisStore(t *testing.T) {
	if _, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip("TL_TEST_REDIS_URL not set")
	}
	client := testutil.MustOpenTestRedisClient(t)
	storetest.Run(t, func(t *testing.T) kvcache.Store {
		// Clear conformance-suite keys from previous subtests.
		keys, err := client.Keys(context.Background(), "storetest:*").Result()
		if err != nil {
			t.Fatal(err)
		}
		if len(keys) > 0 {
			if err := client.Del(context.Background(), keys...).Err(); err != nil {
				t.Fatal(err)
			}
		}
		return kvcache.NewRedisStore(client)
	})
}

func TestRedisStore_NilClient(t *testing.T) {
	ctx := context.Background()
	store := kvcache.NewRedisStore(nil)
	assert.NoError(t, store.Set(ctx, "storetest:a", []byte("x"), 0))
	_, ok, err := store.Get(ctx, "storetest:a")
	assert.NoError(t, err)
	assert.False(t, ok)
	got, err := store.GetMulti(ctx, []string{"storetest:a"})
	assert.NoError(t, err)
	assert.Empty(t, got)
}

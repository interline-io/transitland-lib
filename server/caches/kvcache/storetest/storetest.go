// Package storetest provides a conformance suite for kvcache.Store
// implementations.
package storetest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/stretchr/testify/assert"
)

// Run exercises a Store implementation against the interface contract.
// mk is called per subtest and should return an empty store.
func Run(t *testing.T, mk func(t *testing.T) kvcache.Store) {
	ctx := context.Background()
	t.Run("SetGet", func(t *testing.T) {
		store := mk(t)
		assert.NoError(t, store.Set(ctx, "storetest:a", []byte("hello"), 0))
		val, ok, err := store.Get(ctx, "storetest:a")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []byte("hello"), val)
	})
	t.Run("MissingKey", func(t *testing.T) {
		store := mk(t)
		_, ok, err := store.Get(ctx, "storetest:absent")
		assert.NoError(t, err)
		assert.False(t, ok)
	})
	t.Run("Overwrite", func(t *testing.T) {
		store := mk(t)
		assert.NoError(t, store.Set(ctx, "storetest:a", []byte("one"), 0))
		assert.NoError(t, store.Set(ctx, "storetest:a", []byte("two"), 0))
		val, ok, err := store.Get(ctx, "storetest:a")
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []byte("two"), val)
	})
	t.Run("GetMulti", func(t *testing.T) {
		store := mk(t)
		assert.NoError(t, store.Set(ctx, "storetest:a", []byte("one"), 0))
		assert.NoError(t, store.Set(ctx, "storetest:b", []byte("two"), 0))
		got, err := store.GetMulti(ctx, []string{"storetest:a", "storetest:b", "storetest:absent"})
		assert.NoError(t, err)
		assert.Equal(t, map[string][]byte{
			"storetest:a": []byte("one"),
			"storetest:b": []byte("two"),
		}, got)
	})
	t.Run("GetMultiEmpty", func(t *testing.T) {
		store := mk(t)
		got, err := store.GetMulti(ctx, nil)
		assert.NoError(t, err)
		assert.Empty(t, got)
	})
	t.Run("TTLExpiry", func(t *testing.T) {
		store := mk(t)
		assert.NoError(t, store.Set(ctx, "storetest:ttl", []byte("gone"), 500*time.Millisecond))
		_, ok, err := store.Get(ctx, "storetest:ttl")
		assert.NoError(t, err)
		assert.True(t, ok, "value should be present before ttl")
		assert.Eventually(t, func() bool {
			_, ok, err := store.Get(ctx, "storetest:ttl")
			return err == nil && !ok
		}, 3*time.Second, 50*time.Millisecond, "value should expire after ttl")
	})
	t.Run("Concurrent", func(t *testing.T) {
		store := mk(t)
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					_ = store.Set(ctx, "storetest:c", []byte("v"), 0)
					_, _, _ = store.Get(ctx, "storetest:c")
					_, _ = store.GetMulti(ctx, []string{"storetest:c"})
				}
			}()
		}
		wg.Wait()
	})
}

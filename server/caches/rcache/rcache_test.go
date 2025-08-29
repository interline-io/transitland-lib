package rcache

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
)

type rcTestKey struct {
	Key  string
	Time int64
}

func (k rcTestKey) String() string {
	a := fmt.Sprintf("%s:%d", k.Key, k.Time)
	return a
}

type rcTestItem struct{ Value string }

func TestCache(t *testing.T) {
	// redis jobs and cache
	if a, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip(a)
		return
	}
	redisClient := testutil.MustOpenTestRedisClient(t)
	pfx := func() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
	now := func() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }

	testKey := func() rcTestKey {
		n := time.Now().UnixNano()
		return rcTestKey{
			Key:  fmt.Sprintf("%s:%d", "test", time.Now().UnixNano()),
			Time: n,
		}
	}
	retKey := func(ctx context.Context, key rcTestKey) (rcTestItem, error) {
		return rcTestItem{Value: key.Key}, nil
	}
	retTime := func(ctx context.Context, key rcTestKey) (rcTestItem, error) {
		return rcTestItem{now()}, nil
	}

	t.Run("simple get", func(t *testing.T) {
		key := testKey()
		rc := NewCache[rcTestKey, rcTestItem](retKey, pfx(), redisClient)
		if a, ok := rc.Get(context.Background(), key); ok {
			assert.Equal(t, key.Key, a.Value)
		} else {
			t.Error("expected ok read")
		}
	})

	t.Run("get, refresh failed", func(t *testing.T) {
		key := testKey()
		refreshFn := func(ctx context.Context, key rcTestKey) (rcTestItem, error) {
			return rcTestItem{key.Key}, errors.New("fail")
		}
		rc := NewCache[rcTestKey, rcTestItem](refreshFn, pfx(), redisClient)
		if _, ok := rc.Get(context.Background(), key); ok {
			t.Error("expected failed read")
		}
	})

	t.Run("get, refresh timeout", func(t *testing.T) {
		key := testKey()
		refreshFn := func(ctx context.Context, key rcTestKey) (rcTestItem, error) {
			time.Sleep(2 * time.Second)
			return rcTestItem{key.Key}, nil
		}
		rc := NewCache[rcTestKey, rcTestItem](refreshFn, pfx(), redisClient)
		rc.RefreshTimeout = time.Duration(1 * time.Second)
		if _, ok := rc.Get(context.Background(), key); ok {
			t.Error("expected failed read")
		}
	})

	t.Run("get, refresh not timed out", func(t *testing.T) {
		key := testKey()
		refreshFn := func(ctx context.Context, key rcTestKey) (rcTestItem, error) {
			time.Sleep(2 * time.Second)
			return rcTestItem{key.Key}, nil
		}
		rc := NewCache[rcTestKey, rcTestItem](refreshFn, pfx(), redisClient)
		rc.RefreshTimeout = time.Duration(10 * time.Second)
		if a, ok := rc.Get(context.Background(), key); ok {
			assert.Equal(t, key.Key, a.Value)
		} else {
			t.Error("expeced ok read")
		}
	})

	t.Run("check, redis read ok", func(t *testing.T) {
		key := testKey()
		rc := NewCache[rcTestKey, rcTestItem](retKey, pfx(), redisClient)

		// Check no value
		if _, ok := rc.Check(context.Background(), key); ok {
			t.Error("expected failed read")
		}

		// Manually set redis
		testItem, _ := retKey(context.Background(), key)
		cacheItem := Item[rcTestItem]{
			Value:     testItem,
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		rc.setRedis(context.Background(), key, cacheItem)

		// Check again
		if a, ok := rc.Check(context.Background(), key); ok {
			assert.Equal(t, key.Key, a.Value)
		} else {
			t.Error("expected ok read")
		}

		// Check local value
		if a, ok := rc.getLocal(key); ok {
			assert.Equal(t, testItem, a.Value)
		} else {
			t.Error("expected ok read")
		}
	})

	t.Run("check, not expired item", func(t *testing.T) {
		key := testKey()
		rc := NewCache[rcTestKey, rcTestItem](retKey, pfx(), redisClient)
		rc.Expires = 10 * time.Second

		// OK
		if a, ok := rc.Get(context.Background(), key); ok {
			assert.Equal(t, key.Key, a.Value)
		} else {
			t.Error("expected ok read")
		}

		// Wait
		time.Sleep(2 * time.Second)

		// Check again
		if a, ok := rc.Check(context.Background(), key); ok {
			assert.Equal(t, key.Key, a.Value)
		} else {
			t.Error("expected ok read")
		}
	})

	t.Run("check, expired item", func(t *testing.T) {
		key := testKey()
		rc := NewCache[rcTestKey, rcTestItem](retKey, pfx(), redisClient)
		rc.Expires = 1 * time.Second

		// OK
		if a, ok := rc.Get(context.Background(), key); ok {
			assert.Equal(t, key.Key, a.Value)
		} else {
			t.Error("expected ok read")
		}

		// Wait
		time.Sleep(2 * time.Second)

		// Check again
		if _, ok := rc.Check(context.Background(), key); ok {
			t.Error("expected failed read")
		}
	})

	t.Run("recheck", func(t *testing.T) {
		key := testKey()
		// Set refresh interval to 1 second
		rc := NewCache[rcTestKey, rcTestItem](retTime, pfx(), redisClient)
		rc.Recheck = 3 * time.Second
		rc.Expires = 10 * time.Second
		rc.Start(1 * time.Second)

		// OK
		firstTime := ""
		if a, ok := rc.Get(context.Background(), key); ok {
			firstTime = a.Value
		} else {
			t.Error("expected ok read")
		}

		// Wait
		time.Sleep(5 * time.Second)

		// Check again
		if a, ok := rc.Check(context.Background(), key); ok {
			assert.Greater(t, a.Value, firstTime, "expected value to be increased by recheck timer")
		} else {
			t.Error("expected ok read")
		}
	})

}

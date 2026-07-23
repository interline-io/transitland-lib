package kvcache_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/stretchr/testify/assert"
)

// testClock is a settable clock for driving TTL transitions.
type testClock struct {
	lock sync.Mutex
	t    time.Time
}

func newTestClock() *testClock {
	return &testClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *testClock) Now() time.Time {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.t
}

func (c *testClock) Advance(d time.Duration) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.t = c.t.Add(d)
}

// recordingStore wraps a Store and records keys and operation counts.
type recordingStore struct {
	kvcache.Store
	lock    sync.Mutex
	setKeys []string
	gets    int
}

func (s *recordingStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s.lock.Lock()
	s.gets++
	s.lock.Unlock()
	return s.Store.Get(ctx, key)
}

func (s *recordingStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.lock.Lock()
	s.setKeys = append(s.setKeys, key)
	s.lock.Unlock()
	return s.Store.Set(ctx, key, value, ttl)
}

func (s *recordingStore) getCount() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.gets
}

func TestCache_LocalSetGet(t *testing.T) {
	ctx := context.Background()
	c := kvcache.NewCache[string, string](nil, "test")
	_, ok := c.Get(ctx, "a")
	assert.False(t, ok)
	assert.NoError(t, c.SetTTL(ctx, "a", "hello", time.Minute, time.Hour))
	v, ok := c.Get(ctx, "a")
	assert.True(t, ok)
	assert.Equal(t, "hello", v)
}

func TestCache_SharedStore(t *testing.T) {
	// Two caches sharing one store emulate two processes.
	ctx := context.Background()
	store := kvcache.NewMemoryStore()
	a := kvcache.NewCache[string, string](store, "topic")
	b := kvcache.NewCache[string, string](store, "topic")
	assert.NoError(t, a.SetTTL(ctx, "k", "from-a", time.Minute, time.Hour))
	v, ok := b.Get(ctx, "k")
	assert.True(t, ok)
	assert.Equal(t, "from-a", v)
}

func TestCache_LocalExpiry(t *testing.T) {
	ctx := context.Background()
	clock := newTestClock()
	c := kvcache.NewCache[string, string](nil, "test")
	c.Clock = clock.Now
	assert.NoError(t, c.SetTTL(ctx, "a", "hello", time.Minute, time.Hour))
	_, ok := c.Get(ctx, "a")
	assert.True(t, ok)
	clock.Advance(2 * time.Hour)
	_, ok = c.Get(ctx, "a")
	assert.False(t, ok, "expired local entry must not be served")
}

func TestCache_StorageMissNotPoisoned(t *testing.T) {
	// A storage miss must not install a zero-value local entry.
	ctx := context.Background()
	store := kvcache.NewMemoryStore()
	c := kvcache.NewCache[string, string](store, "test")
	_, ok := c.Get(ctx, "a")
	assert.False(t, ok)
	_, ok = c.Get(ctx, "a")
	assert.False(t, ok, "second read after miss must still miss")
	assert.Empty(t, c.LocalKeys())
}

func TestCache_WireFormat(t *testing.T) {
	// The envelope and key must match the legacy ecache format so old
	// and new pods share warm caches during a rolling deploy. Delete
	// this test when ecache/rcache are removed.
	type legacyItem struct {
		Value     string
		ExpiresAt time.Time
		RecheckAt time.Time
	}
	n := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	want, err := json.Marshal(legacyItem{Value: "x", ExpiresAt: n.Add(time.Hour), RecheckAt: n.Add(time.Minute)})
	assert.NoError(t, err)
	got, err := json.Marshal(kvcache.Item[string]{Value: "x", ExpiresAt: n.Add(time.Hour), RecheckAt: n.Add(time.Minute)})
	assert.NoError(t, err)
	assert.Equal(t, string(want), string(got))

	// Key composition: <prefix>:<topic>:<key> with default prefix "ecache".
	ctx := context.Background()
	store := &recordingStore{Store: kvcache.NewMemoryStore()}
	c := kvcache.NewCache[string, string](store, "gbfs")
	assert.NoError(t, c.Set(ctx, "some-feed:en", "v"))
	assert.Equal(t, []string{"ecache:gbfs:some-feed:en"}, store.setKeys)
}

func TestCache_RefreshReadThrough(t *testing.T) {
	ctx := context.Background()
	var calls atomic.Int64
	c := kvcache.NewRefreshCache[string, int](kvcache.NewMemoryStore(), "test", func(ctx context.Context, key string) (int, error) {
		calls.Add(1)
		return len(key), nil
	})
	v, ok := c.Get(ctx, "abc")
	assert.True(t, ok)
	assert.Equal(t, 3, v)
	assert.EqualValues(t, 1, calls.Load())
	// Second get is a local hit.
	_, ok = c.Get(ctx, "abc")
	assert.True(t, ok)
	assert.EqualValues(t, 1, calls.Load())
}

func TestCache_RefreshError(t *testing.T) {
	ctx := context.Background()
	c := kvcache.NewRefreshCache[string, int](nil, "test", func(ctx context.Context, key string) (int, error) {
		return 0, errors.New("upstream down")
	})
	_, ok := c.Get(ctx, "a")
	assert.False(t, ok)
	assert.Empty(t, c.LocalKeys(), "failed refresh must not install an entry")
}

func TestCache_Singleflight(t *testing.T) {
	// N concurrent cold reads produce one refresh call.
	ctx := context.Background()
	var calls atomic.Int64
	c := kvcache.NewRefreshCache[string, string](kvcache.NewMemoryStore(), "test", func(ctx context.Context, key string) (string, error) {
		calls.Add(1)
		time.Sleep(50 * time.Millisecond)
		return "v", nil
	})
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, ok := c.Get(ctx, "k")
			assert.True(t, ok)
			assert.Equal(t, "v", v)
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, calls.Load())
}

func TestCache_SingleflightLeaderCancel(t *testing.T) {
	// The flight leader's canceled context must not fail the waiters.
	leaderCtx, cancelLeader := context.WithCancel(context.Background())
	release := make(chan struct{})
	c := kvcache.NewRefreshCache[string, string](nil, "test", func(ctx context.Context, key string) (string, error) {
		cancelLeader()
		<-release
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return "v", nil
	})
	c.RefreshTimeout = 5 * time.Second

	var wg sync.WaitGroup
	var waiterOk atomic.Bool
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Leader: enters the flight, cancels its own ctx inside refreshFn.
		c.Get(leaderCtx, "k")
	}()
	// Give the leader time to enter the flight, then join it and release.
	time.Sleep(20 * time.Millisecond)
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		close(release)
	}()
	v, ok := c.Get(context.Background(), "k")
	wg.Wait()
	waiterOk.Store(ok)
	assert.True(t, waiterOk.Load(), "waiter must receive the refreshed value despite leader cancellation")
	assert.Equal(t, "v", v)
}

func TestCache_NegativeRefresh(t *testing.T) {
	// ErrNotFound with NegativeTTL installs a tombstone that suppresses
	// refresh calls until it lapses.
	ctx := context.Background()
	clock := newTestClock()
	var calls atomic.Int64
	c := kvcache.NewRefreshCache[string, string](kvcache.NewMemoryStore(), "test", func(ctx context.Context, key string) (string, error) {
		calls.Add(1)
		return "", fmt.Errorf("no such user: %w", kvcache.ErrNotFound)
	})
	c.Clock = clock.Now
	c.NegativeTTL = time.Minute

	_, ok := c.Get(ctx, "ghost")
	assert.False(t, ok)
	assert.EqualValues(t, 1, calls.Load())
	_, ok = c.Get(ctx, "ghost")
	assert.False(t, ok)
	assert.EqualValues(t, 1, calls.Load(), "tombstone must suppress refresh")
	it, ok := c.GetItem(ctx, "ghost")
	assert.True(t, ok)
	assert.True(t, it.Missing)

	clock.Advance(2 * time.Minute)
	_, ok = c.Get(ctx, "ghost")
	assert.False(t, ok)
	assert.EqualValues(t, 2, calls.Load(), "lapsed tombstone must allow refresh")
}

func TestCache_SetMissing(t *testing.T) {
	// Hand-rolled read-through consumers can tombstone explicitly and
	// distinguish tombstones via GetItem.
	ctx := context.Background()
	store := kvcache.NewMemoryStore()
	c := kvcache.NewCache[string, string](store, "test")
	assert.Error(t, c.SetMissing(ctx, "ghost"), "requires NegativeTTL")
	c.NegativeTTL = time.Minute
	assert.NoError(t, c.SetMissing(ctx, "ghost"))
	_, ok := c.Get(ctx, "ghost")
	assert.False(t, ok)
	it, ok := c.GetItem(ctx, "ghost")
	assert.True(t, ok)
	assert.True(t, it.Missing)
	// The tombstone is shared: a second cache on the same store sees it.
	c2 := kvcache.NewCache[string, string](store, "test")
	it, ok = c2.GetItem(ctx, "ghost")
	assert.True(t, ok)
	assert.True(t, it.Missing)
}

func TestCache_NegativeLocalOnly(t *testing.T) {
	// Without a refresh function, NegativeTTL suppresses repeat storage
	// reads for absent keys locally without writing tombstones to storage.
	ctx := context.Background()
	store := &recordingStore{Store: kvcache.NewMemoryStore()}
	c := kvcache.NewCache[string, string](store, "test")
	c.NegativeTTL = time.Minute
	_, ok := c.Get(ctx, "absent")
	assert.False(t, ok)
	_, ok = c.Get(ctx, "absent")
	assert.False(t, ok)
	assert.Equal(t, 1, store.getCount(), "second read must be suppressed by the local tombstone")
	assert.Empty(t, store.setKeys, "local-only tombstone must not be written to storage")
}

func TestCache_RecheckConvergence(t *testing.T) {
	// Pod B adopts pod A's refresh via the scan instead of re-fetching.
	ctx := context.Background()
	clockA, clockB := newTestClock(), newTestClock()
	store := kvcache.NewMemoryStore()
	a := kvcache.NewCache[string, string](store, "topic")
	a.Clock = clockA.Now
	b := kvcache.NewCache[string, string](store, "topic")
	b.Clock = clockB.Now

	assert.NoError(t, a.SetTTL(ctx, "k", "v1", 5*time.Minute, 24*time.Hour))
	_, ok := b.Get(ctx, "k")
	assert.True(t, ok)

	// Both advance past the recheck; A refreshes first.
	clockA.Advance(6 * time.Minute)
	clockB.Advance(6 * time.Minute)
	assert.NoError(t, a.SetTTL(ctx, "k", "v2", 5*time.Minute, 24*time.Hour))

	// B's scan adopts A's fresher envelope; the key is no longer due.
	due := b.GetRecheckKeys(ctx)
	assert.Empty(t, due)
	v, ok := b.Get(ctx, "k")
	assert.True(t, ok)
	assert.Equal(t, "v2", v)
}

func TestCache_ScanPrunesExpired(t *testing.T) {
	ctx := context.Background()
	clock := newTestClock()
	c := kvcache.NewCache[string, string](kvcache.NewMemoryStore(), "test")
	c.Clock = clock.Now
	assert.NoError(t, c.SetTTL(ctx, "a", "v", time.Minute, time.Hour))
	clock.Advance(2 * time.Hour)
	c.GetRecheckKeys(ctx)
	assert.Empty(t, c.LocalKeys(), "expired entries must be pruned by the scan")
}

func TestCache_GetRecheckKeysDue(t *testing.T) {
	ctx := context.Background()
	clock := newTestClock()
	c := kvcache.NewCache[string, string](nil, "test")
	c.Clock = clock.Now
	assert.NoError(t, c.SetTTL(ctx, "a", "v", 5*time.Minute, 24*time.Hour))
	assert.Empty(t, c.GetRecheckKeys(ctx))
	clock.Advance(6 * time.Minute)
	assert.Equal(t, []string{"a"}, c.GetRecheckKeys(ctx))
}

func TestCache_StartStop(t *testing.T) {
	// The ticker refreshes due keys and Stop halts it; Start/Stop are
	// idempotent and Start-after-Stop restarts.
	ctx := context.Background()
	var calls atomic.Int64
	c := kvcache.NewRefreshCache[string, string](nil, "test", func(ctx context.Context, key string) (string, error) {
		calls.Add(1)
		return "v", nil
	})
	c.Recheck = 1 * time.Millisecond
	c.Expires = time.Hour
	_, ok := c.Get(ctx, "k")
	assert.True(t, ok)
	before := calls.Load()

	c.Start(5 * time.Millisecond)
	c.Start(5 * time.Millisecond) // no-op
	assert.Eventually(t, func() bool { return calls.Load() > before }, 2*time.Second, 5*time.Millisecond)
	c.Stop()
	c.Stop() // no-op
	after := calls.Load()
	time.Sleep(30 * time.Millisecond)
	assert.Equal(t, after, calls.Load(), "no refreshes after Stop")

	c.Start(5 * time.Millisecond)
	assert.Eventually(t, func() bool { return calls.Load() > after }, 2*time.Second, 5*time.Millisecond)
	c.Stop()
}

type stringerKey struct{ A, B string }

func (k stringerKey) String() string { return k.A + ":" + k.B }

type plainKey struct{ A int }

func TestCache_KeyStringification(t *testing.T) {
	ctx := context.Background()
	// fmt.Stringer keys use String().
	s1 := &recordingStore{Store: kvcache.NewMemoryStore()}
	c1 := kvcache.NewCache[stringerKey, string](s1, "t")
	assert.NoError(t, c1.Set(ctx, stringerKey{"x", "y"}, "v"))
	assert.Equal(t, []string{"ecache:t:x:y"}, s1.setKeys)
	// Other comparable keys use JSON.
	s2 := &recordingStore{Store: kvcache.NewMemoryStore()}
	c2 := kvcache.NewCache[plainKey, string](s2, "t")
	assert.NoError(t, c2.Set(ctx, plainKey{7}, "v"))
	assert.Equal(t, []string{`ecache:t:{"A":7}`}, s2.setKeys)
}

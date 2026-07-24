package kvcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/interline-io/log"
	"golang.org/x/sync/singleflight"
)

// ErrNotFound is returned by refresh functions to indicate a key is
// authoritatively absent; with NegativeTTL configured, the absence is
// cached as a tombstone.
var ErrNotFound = errors.New("not found")

// Item is the stored envelope for a cached value. The JSON encoding is
// wire-compatible with the ecache/rcache envelope.
type Item[V any] struct {
	Value     V
	ExpiresAt time.Time
	RecheckAt time.Time
	Missing   bool `json:",omitempty"`
}

// Cache is a generic two-tier cache: a per-process local map in front of
// an optional shared Store. Entries carry a soft recheck time and a hard
// expiry; expired entries are never served from either tier.
type Cache[K comparable, V any] struct {
	// Recheck and Expires are the TTLs applied by Set and Refresh
	// (defaults 1h / 1h).
	Recheck time.Duration
	Expires time.Duration
	// RefreshTimeout bounds each refresh function call (default 1s).
	RefreshTimeout time.Duration
	// NegativeTTL enables tombstone caching of authoritatively absent
	// keys — via SetMissing or a refresh function returning ErrNotFound
	// (default off). Legacy ecache/rcache readers do not understand
	// tombstones and see them as zero-value hits; do not enable shared
	// tombstones on a topic until all legacy readers are retired.
	NegativeTTL time.Duration
	// KeyPrefix namespaces storage keys as "<prefix>:<topic>:<key>". The
	// default "ecache" preserves the legacy wire format.
	KeyPrefix string
	// Clock overrides time.Now, for tests.
	Clock func() time.Time

	topic     string
	store     Store
	refreshFn func(context.Context, K) (V, error)

	lock  sync.RWMutex
	items map[K]Item[V]
	sf    singleflight.Group

	tickerLock sync.Mutex
	tickerStop chan struct{}
	tickerWg   sync.WaitGroup
}

// NewCache returns a two-tier cache; store may be nil for local-only use.
func NewCache[K comparable, V any](store Store, topic string) *Cache[K, V] {
	return &Cache[K, V]{
		Recheck:        1 * time.Hour,
		Expires:        1 * time.Hour,
		RefreshTimeout: 1 * time.Second,
		KeyPrefix:      "ecache",
		topic:          topic,
		store:          store,
		items:          map[K]Item[V]{},
	}
}

// NewRefreshCache returns a cache that populates values read-through on
// miss and in the background via Start.
func NewRefreshCache[K comparable, V any](store Store, topic string, refreshFn func(context.Context, K) (V, error)) *Cache[K, V] {
	c := NewCache[K, V](store, topic)
	c.refreshFn = refreshFn
	return c
}

// Get returns the cached value for key. Tombstoned (negative) entries
// read as misses.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, bool) {
	if it, ok := c.GetItem(ctx, key); ok && !it.Missing {
		return it.Value, true
	}
	var zero V
	return zero, false
}

// GetItem returns the full envelope for key; ok is true when a valid
// envelope is known, including negative tombstones (check Missing).
func (c *Cache[K, V]) GetItem(ctx context.Context, key K) (Item[V], bool) {
	c.lock.RLock()
	it, ok := c.items[key]
	c.lock.RUnlock()
	if ok && it.ExpiresAt.After(c.now()) {
		return it, true
	}
	return c.loadOrRefresh(ctx, key)
}

// Set stores value in both tiers using the configured Recheck/Expires TTLs.
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V) error {
	return c.SetTTL(ctx, key, value, c.Recheck, c.Expires)
}

// SetTTL stores value in both tiers. The local tier serves it until
// expireTTL; the shared tier additionally applies expireTTL as its
// backend TTL. Storage writes are best-effort: the local tier is always
// updated.
func (c *Cache[K, V]) SetTTL(ctx context.Context, key K, value V, recheckTTL time.Duration, expireTTL time.Duration) error {
	n := c.now()
	it := Item[V]{
		Value:     value,
		RecheckAt: n.Add(recheckTTL),
		ExpiresAt: n.Add(expireTTL),
	}
	c.setLocal(key, it)
	return c.setStore(ctx, key, it, expireTTL)
}

// SetMissing stores a negative tombstone for key in both tiers for
// NegativeTTL, suppressing lookups until it lapses. See NegativeTTL for
// the legacy-reader caveat.
func (c *Cache[K, V]) SetMissing(ctx context.Context, key K) error {
	if c.NegativeTTL <= 0 {
		return errors.New("kvcache: SetMissing requires NegativeTTL")
	}
	it := c.missingItem(c.NegativeTTL)
	c.setLocal(key, it)
	return c.setStore(ctx, key, it, c.NegativeTTL)
}

// Refresh calls the refresh function for key and stores the result.
func (c *Cache[K, V]) Refresh(ctx context.Context, key K) (V, error) {
	var zero V
	if c.refreshFn == nil {
		return zero, errors.New("kvcache: no refresh function")
	}
	it, err := c.runRefresh(ctx, key)
	if err != nil {
		return zero, err
	}
	return it.Value, nil
}

// GetRecheckKeys reconciles the local tier against the shared tier in a
// single multi-get — adopting entries refreshed by other processes and
// pruning expired ones — and returns the keys still due for refresh.
func (c *Cache[K, V]) GetRecheckKeys(ctx context.Context) []K {
	return c.scan(ctx)
}

// LocalKeys returns a snapshot of locally known keys.
func (c *Cache[K, V]) LocalKeys() []K {
	c.lock.RLock()
	defer c.lock.RUnlock()
	ret := make([]K, 0, len(c.items))
	for k := range c.items {
		ret = append(ret, k)
	}
	return ret
}

// Start launches a background goroutine that periodically reconciles
// with the shared tier and, when a refresh function is configured,
// refreshes due keys. Start is a no-op if already running. It panics on
// a non-positive interval, like time.NewTicker.
func (c *Cache[K, V]) Start(interval time.Duration) {
	if interval <= 0 {
		panic("kvcache: non-positive interval for Start")
	}
	c.tickerLock.Lock()
	defer c.tickerLock.Unlock()
	if c.tickerStop != nil {
		return
	}
	stop := make(chan struct{})
	c.tickerStop = stop
	c.tickerWg.Add(1)
	go func() {
		defer c.tickerWg.Done()
		for {
			select {
			case <-time.After(jitter(interval)):
				c.tick()
			case <-stop:
				return
			}
		}
	}()
}

// Stop halts the background goroutine and waits for an in-flight tick.
// The cache remains usable, and Start may be called again.
func (c *Cache[K, V]) Stop() {
	// The lock is held across Wait so a concurrent Start cannot slip in
	// between shutdown and the wait; the ticker goroutine never takes
	// this lock, so this cannot deadlock.
	c.tickerLock.Lock()
	defer c.tickerLock.Unlock()
	if c.tickerStop == nil {
		return
	}
	close(c.tickerStop)
	c.tickerStop = nil
	c.tickerWg.Wait()
}

func (c *Cache[K, V]) tick() {
	ctx := context.Background()
	due := c.scan(ctx)
	if c.refreshFn == nil {
		return
	}
	for _, key := range due {
		if _, err := c.runRefresh(ctx, key); err != nil {
			log.Trace().Err(err).Str("key", c.storeKey(key)).Msg("kvcache: refresh failed")
		}
	}
}

// loadOrRefresh handles a local miss: consult the shared tier, then the
// refresh function. Concurrent callers for one key share a single flight.
func (c *Cache[K, V]) loadOrRefresh(ctx context.Context, key K) (Item[V], bool) {
	type result struct {
		item Item[V]
		ok   bool
	}
	v, _, _ := c.sf.Do(c.keyString(key), func() (any, error) {
		// The flight's result is shared by all waiters, so its work is
		// detached from the leader's cancellation; backends apply their
		// own per-op timeouts.
		fctx := context.WithoutCancel(ctx)
		if it, ok := c.getStore(fctx, key); ok {
			c.setLocal(key, it)
			return result{item: it, ok: true}, nil
		}
		if c.refreshFn != nil {
			if it, err := c.runRefresh(fctx, key); err == nil {
				return result{item: it, ok: true}, nil
			}
			c.deleteExpiredLocal(key)
			return result{}, nil
		}
		// Definite miss with no refresh source. Absence is cached only
		// when stated authoritatively (SetMissing, or a refresh function
		// returning ErrNotFound) — a bare storage miss may just mean
		// "not fetched yet", and tombstoning it would turn a concurrent
		// first read into a spurious cached miss.
		c.deleteExpiredLocal(key)
		return result{}, nil
	})
	r, _ := v.(result)
	return r.item, r.ok
}

// runRefresh calls refreshFn detached from the caller's cancellation so
// a canceled flight leader cannot fail the waiters sharing its result.
func (c *Cache[K, V]) runRefresh(ctx context.Context, key K) (Item[V], error) {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), c.RefreshTimeout)
	defer cancel()
	value, err := c.refreshFn(rctx, key)
	// The storage write gets its own budget (the backend's per-op
	// timeout), not whatever the refresh call left of RefreshTimeout.
	sctx := context.WithoutCancel(rctx)
	if err == nil {
		n := c.now()
		it := Item[V]{
			Value:     value,
			RecheckAt: n.Add(c.Recheck),
			ExpiresAt: n.Add(c.Expires),
		}
		c.setLocal(key, it)
		_ = c.setStore(sctx, key, it, c.Expires)
		return it, nil
	}
	if errors.Is(err, ErrNotFound) && c.NegativeTTL > 0 {
		it := c.missingItem(c.NegativeTTL)
		c.setLocal(key, it)
		_ = c.setStore(sctx, key, it, c.NegativeTTL)
		return it, nil
	}
	return Item[V]{}, err
}

// scan reconciles all local keys against the shared tier in one
// GetMulti, adopts fresher envelopes written by other processes, prunes
// expired entries, and returns the keys still past their RecheckAt.
func (c *Cache[K, V]) scan(ctx context.Context) []K {
	c.lock.RLock()
	keys := make([]K, 0, len(c.items))
	for k := range c.items {
		keys = append(keys, k)
	}
	c.lock.RUnlock()
	if len(keys) == 0 {
		return nil
	}
	fetched := map[string][]byte{}
	if c.store != nil {
		skeys := make([]string, len(keys))
		for i, k := range keys {
			skeys[i] = c.storeKey(k)
		}
		var err error
		fetched, err = c.store.GetMulti(ctx, skeys)
		if err != nil {
			log.Trace().Err(err).Msg("kvcache: storage multi-read failed")
			fetched = map[string][]byte{}
		}
	}
	n := c.now()
	var due []K
	c.lock.Lock()
	for _, k := range keys {
		it, ok := c.items[k]
		if !ok {
			continue
		}
		if data, ok := fetched[c.storeKey(k)]; ok {
			sIt := Item[V]{}
			if err := json.Unmarshal(data, &sIt); err == nil && sIt.ExpiresAt.After(n) {
				// Adopt when fresher, or when the local copy is expired.
				if sIt.RecheckAt.After(it.RecheckAt) || !it.ExpiresAt.After(n) {
					c.items[k] = sIt
					it = sIt
				}
			}
		}
		if !it.ExpiresAt.After(n) {
			delete(c.items, k)
			continue
		}
		if !it.RecheckAt.After(n) {
			due = append(due, k)
		}
	}
	c.lock.Unlock()
	return due
}

func (c *Cache[K, V]) getStore(ctx context.Context, key K) (Item[V], bool) {
	if c.store == nil {
		return Item[V]{}, false
	}
	skey := c.storeKey(key)
	data, ok, err := c.store.Get(ctx, skey)
	if err != nil {
		log.Trace().Err(err).Str("key", skey).Msg("kvcache: storage read failed")
		return Item[V]{}, false
	}
	if !ok {
		return Item[V]{}, false
	}
	it := Item[V]{}
	if err := json.Unmarshal(data, &it); err != nil {
		log.Trace().Err(err).Str("key", skey).Msg("kvcache: storage read failed during unmarshal")
		return Item[V]{}, false
	}
	if !it.ExpiresAt.After(c.now()) {
		return Item[V]{}, false
	}
	return it, true
}

func (c *Cache[K, V]) setLocal(key K, it Item[V]) {
	c.lock.Lock()
	c.items[key] = it
	c.lock.Unlock()
}

// deleteExpiredLocal removes key's local entry if it is still expired,
// so keys that never re-materialize don't accumulate between scans. The
// re-check under the write lock avoids clobbering a concurrent Set.
func (c *Cache[K, V]) deleteExpiredLocal(key K) {
	n := c.now()
	c.lock.Lock()
	if it, ok := c.items[key]; ok && !it.ExpiresAt.After(n) {
		delete(c.items, key)
	}
	c.lock.Unlock()
}

func (c *Cache[K, V]) setStore(ctx context.Context, key K, it Item[V], ttl time.Duration) error {
	if c.store == nil {
		return nil
	}
	// A non-positive ttl means the envelope is expired on arrival; on
	// backends ttl<=0 means "no expiry", which would strand a dead key.
	if ttl <= 0 {
		return nil
	}
	skey := c.storeKey(key)
	data, err := json.Marshal(it)
	if err != nil {
		return err
	}
	if err := c.store.Set(ctx, skey, data, ttl); err != nil {
		log.Trace().Err(err).Str("key", skey).Msg("kvcache: storage write failed")
		return err
	}
	return nil
}

func (c *Cache[K, V]) missingItem(ttl time.Duration) Item[V] {
	n := c.now()
	return Item[V]{
		Missing:   true,
		RecheckAt: n.Add(ttl),
		ExpiresAt: n.Add(ttl),
	}
}

func (c *Cache[K, V]) storeKey(key K) string {
	return fmt.Sprintf("%s:%s:%s", c.KeyPrefix, c.topic, c.keyString(key))
}

func (c *Cache[K, V]) keyString(key K) string {
	switch v := any(key).(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	}
	data, err := json.Marshal(key)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func (c *Cache[K, V]) now() time.Time {
	if c.Clock != nil {
		return c.Clock()
	}
	return time.Now().In(time.UTC)
}

// jitter spreads a ticker interval by ±10% so a fleet of processes
// desynchronizes instead of refreshing in lockstep.
func jitter(d time.Duration) time.Duration {
	if d <= 0 {
		return d
	}
	return time.Duration(float64(d) * (0.9 + 0.2*rand.Float64()))
}

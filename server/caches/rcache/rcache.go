package rcache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/interline-io/log"
)

type Item[T any] struct {
	Value     T
	ExpiresAt time.Time
	RecheckAt time.Time
}

type Cache[K comparable, T any] struct {
	RedisTimeout   time.Duration
	RefreshTimeout time.Duration
	Recheck        time.Duration
	Expires        time.Duration
	refreshFn      func(context.Context, K) (T, error)
	topic          string
	items          map[K]Item[T]
	lock           sync.Mutex
	redisClient    *redis.Client
}

func NewCache[K comparable, T any](refreshFn func(context.Context, K) (T, error), keyPrefix string, redisClient *redis.Client) *Cache[K, T] {
	topic := "test"
	rc := Cache[K, T]{
		refreshFn:      refreshFn,
		topic:          topic,
		redisClient:    redisClient,
		items:          map[K]Item[T]{},
		Recheck:        1 * time.Hour,
		Expires:        1 * time.Hour,
		RefreshTimeout: 1 * time.Second,
		RedisTimeout:   1 * time.Second,
	}
	return &rc
}

func (rc *Cache[K, T]) Start(t time.Duration) {
	ticker := time.NewTicker(t)
	go func() {
		for t := range ticker.C {
			_ = t
			ctx := context.Background()
			keys := rc.GetRecheckKeys(ctx)
			for _, key := range keys {
				rc.Refresh(ctx, key)
			}
		}
	}()
}

func (rc *Cache[K, T]) Check(ctx context.Context, key K) (T, bool) {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	return rc.check(ctx, key)
}

func (rc *Cache[K, T]) check(ctx context.Context, key K) (T, bool) {
	a, ok := rc.getLocal(key)
	if ok {
		return a.Value, ok
	}
	b, ok := rc.getRedis(ctx, key)
	if ok {
		rc.setLocal(key, b)
	}
	return b.Value, ok
}

func (rc *Cache[K, T]) Get(ctx context.Context, key K) (T, bool) {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	a, ok := rc.check(ctx, key)
	if !ok {
		if val, err := rc.refresh(ctx, key); err == nil {
			a = val
			ok = true
		}
	}
	return a, ok
}

func (rc *Cache[K, T]) SetTTL(ctx context.Context, key K, value T, ttl1 time.Duration, ttl2 time.Duration) error {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	return rc.setTTL(ctx, key, value, ttl1, ttl2)
}

func (rc *Cache[K, T]) setTTL(ctx context.Context, key K, value T, ttl1 time.Duration, ttl2 time.Duration) error {
	n := time.Now().In(time.UTC)
	item := Item[T]{
		Value:     value,
		RecheckAt: n.Add(ttl1),
		ExpiresAt: n.Add(ttl2),
	}
	rc.setLocal(key, item)
	rc.setRedis(ctx, key, item)
	return nil
}

func (rc *Cache[K, T]) GetRecheckKeys(ctx context.Context) []K {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	t := time.Now().In(time.UTC)
	var ret []K
	for k, v := range rc.items {
		// Update?
		if v.RecheckAt.After(t) {
			continue
		}
		// Refresh local cache from redis
		if a, ok := rc.getRedis(ctx, k); ok {
			v = a
			rc.items[k] = v
		}
		// Check again
		if v.RecheckAt.After(t) {
			continue
		}
		ret = append(ret, k)
	}
	return ret
}

func (rc *Cache[K, T]) Refresh(ctx context.Context, key K) (T, error) {
	rc.lock.Lock()
	defer rc.lock.Unlock()
	return rc.refresh(ctx, key)
}

func (rc *Cache[K, T]) refresh(ctx context.Context, key K) (T, error) {
	kstr := toString(key)
	type rt struct {
		item T
		err  error
	}
	result := make(chan rt, 1)
	go func(ctx context.Context, key K) {
		item, err := rc.refreshFn(ctx, key)
		result <- rt{item: item, err: err}
	}(ctx, key)
	var err error
	var item T
	select {
	case <-time.After(rc.RefreshTimeout):
		err = errors.New("timed out")
	case ret := <-result:
		err = ret.err
		item = ret.item
	}
	if err != nil {
		log.Error().Err(err).Str("key", kstr).Msg("refresh: failed to refresh")
		return item, err
	}
	err = rc.setTTL(ctx, key, item, rc.Recheck, rc.Expires)
	if err != nil {
		log.Error().Err(err).Str("key", kstr).Msg("refresh: failed to set TTL")
		return item, err
	}
	log.Trace().Str("key", kstr).Msg("refresh: ok")
	return item, nil
}

func (rc *Cache[K, T]) getLocal(key K) (Item[T], bool) {
	kstr := toString(key)
	log.Trace().Str("key", kstr).Msg("local read: start")
	a, ok := rc.items[key]
	if !ok {
		log.Trace().Str("key", kstr).Msg("local read: not present")
		return a, false
	}
	if a.ExpiresAt.Before(time.Now()) {
		log.Trace().Str("key", kstr).Msg("local read: expired")
		return a, false
	}
	log.Trace().Str("key", kstr).Msg("local read: ok")
	return a, ok
}

func (rc *Cache[K, T]) getRedis(ctx context.Context, key K) (Item[T], bool) {
	ekey := rc.redisKey(key)
	log.Trace().Str("key", ekey).Msg("redis read: start")
	if rc.redisClient == nil {
		log.Trace().Str("key", ekey).Msg("redis read: no redis client")
		return Item[T]{}, false
	}
	rctx, cc := context.WithTimeout(ctx, rc.RedisTimeout)
	defer cc()
	lastData := rc.redisClient.Get(rctx, ekey)
	if err := lastData.Err(); err != nil {
		log.Trace().Err(err).Str("key", ekey).Msg("redis read: not present")
		return Item[T]{}, false
	}
	a, err := lastData.Bytes()
	if err != nil {
		log.Error().Err(err).Str("key", ekey).Msg("redis read: bytes failed")
		return Item[T]{}, false
	}
	t := time.Now().In(time.UTC)
	ld := Item[T]{
		ExpiresAt: t,
		RecheckAt: t,
	}
	if err := json.Unmarshal(a, &ld); err != nil {
		log.Error().Err(err).Str("key", ekey).Msg("redis read: failed during unmarshal")
	}
	if ld.ExpiresAt.Before(time.Now()) {
		log.Trace().Str("key", ekey).Msg("redis read: expired")
		return ld, false
	}
	log.Trace().Str("key", ekey).Msg("redis read: ok")
	return ld, true
}

func (rc *Cache[K, T]) setLocal(key K, item Item[T]) error {
	kstr := toString(key)
	log.Trace().Str("key", kstr).Msg("local write: ok")
	rc.items[key] = item
	return nil
}

func (rc *Cache[K, T]) setRedis(ctx context.Context, key K, item Item[T]) error {
	ekey := rc.redisKey(key)
	log.Trace().Str("key", ekey).Msg("redis write: start")
	if rc.redisClient == nil {
		log.Trace().Str("key", ekey).Msg("redis write: no redis client")
		return nil
	}
	rctx, cc := context.WithTimeout(ctx, rc.RedisTimeout)
	defer cc()
	data, err := json.Marshal(item)
	if err != nil {
		log.Error().Err(err).Str("key", ekey).Msg("redis write: failed during marshal")
		return err
	}
	log.Trace().Str("key", ekey).Str("data", string(data)).Msg("redis write: data")
	if err := rc.redisClient.Set(rctx, ekey, data, rc.Expires).Err(); err != nil {
		log.Error().Err(err).Str("key", ekey).Msg("redis write: failed")
	}
	log.Trace().Str("key", ekey).Msg("redis write: ok")
	return nil
}

func (rc *Cache[K, T]) redisKey(key K) string {
	kstr := toString(key)
	return fmt.Sprintf("ecache:%s:%s", rc.topic, kstr)
}

type canString interface {
	String() string
}

func toString(item any) string {
	if v, ok := item.(canString); ok {
		return v.String()
	}
	data, err := json.Marshal(item)
	if err != nil {
		panic(err)
	}
	return string(data)
}

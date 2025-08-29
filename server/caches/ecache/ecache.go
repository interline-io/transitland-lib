package ecache

import (
	"context"
	"encoding/json"
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

type Cache[T any] struct {
	RedisTimeout time.Duration
	topic        string
	m            map[string]Item[T]
	lock         sync.Mutex
	redis        *redis.Client
}

func NewCache[T any](client *redis.Client, topic string) *Cache[T] {
	return &Cache[T]{
		topic:        topic,
		redis:        client,
		m:            map[string]Item[T]{},
		RedisTimeout: 1 * time.Second,
	}
}

func (e *Cache[T]) GetRecheckKeys(ctx context.Context) []string {
	e.lock.Lock()
	defer e.lock.Unlock()
	t := time.Now().In(time.UTC)
	var ret []string
	for k, v := range e.m {
		// Refresh local cache
		if a, ok := e.getRedis(ctx, k); ok {
			v = a
			e.m[k] = v
		}
		// Update?
		if v.RecheckAt.IsZero() {
			continue
		}
		if v.RecheckAt.Before(t) {
			ret = append(ret, k)
		}
	}
	return ret
}

func (e *Cache[T]) Get(ctx context.Context, key string) (T, bool) {
	e.lock.Lock()
	defer e.lock.Unlock()
	if a, ok := e.getLocal(key); ok {
		return a.Value, true
	}
	v, ok := e.getRedis(ctx, key)
	e.setLocal(key, v, 0)
	return v.Value, ok
}

func (e *Cache[T]) LocalKeys() []string {
	e.lock.Lock()
	defer e.lock.Unlock()
	var ret []string
	for k := range e.m {
		ret = append(ret, k)
	}
	return ret
}

func (e *Cache[T]) SetTTL(ctx context.Context, key string, value T, ttl1 time.Duration, ttl2 time.Duration) error {
	e.lock.Lock()
	defer e.lock.Unlock()
	n := time.Now().In(time.UTC)
	item := Item[T]{
		Value:     value,
		RecheckAt: n.Add(ttl1),
		ExpiresAt: n.Add(ttl2),
	}
	e.setLocal(key, item, ttl1)
	e.setRedis(ctx, key, item, ttl2)
	return nil
}

func (e *Cache[T]) redisKey(key string) string {
	return fmt.Sprintf("ecache:%s:%s", e.topic, key)
}

func (e *Cache[T]) getLocal(key string) (Item[T], bool) {
	a, ok := e.m[key]
	return a, ok
}

func (e *Cache[T]) getRedis(ctx context.Context, key string) (Item[T], bool) {
	t := time.Now().In(time.UTC)
	ld := Item[T]{
		ExpiresAt: t,
		RecheckAt: t,
	}
	if e.redis == nil {
		return ld, false
	}
	rctx, cc := context.WithTimeout(ctx, e.RedisTimeout)
	defer cc()
	ekey := e.redisKey(key)
	lastData := e.redis.Get(rctx, ekey)
	if err := lastData.Err(); err != nil {
		log.Trace().Err(err).Str("key", ekey).Msg("redis read failed")
		return ld, false
	}
	a, err := lastData.Bytes()
	if err != nil {
		log.Trace().Err(err).Str("key", ekey).Msg("redis read failed")
		return ld, false
	}
	if err := json.Unmarshal(a, &ld); err != nil {
		log.Trace().Err(err).Str("key", ekey).Msg("redis read failed during unmarshal")
	}
	log.Trace().Str("key", ekey).Msg("redis read ok")
	return ld, true
}

func (e *Cache[T]) setLocal(key string, item Item[T], ttl time.Duration) error {
	e.m[key] = item
	return nil
}

func (e *Cache[T]) setRedis(ctx context.Context, key string, item Item[T], ttl time.Duration) error {
	if e.redis == nil {
		return nil
	}
	rctx, cc := context.WithTimeout(ctx, e.RedisTimeout)
	defer cc()
	ekey := e.redisKey(key)
	data, err := json.Marshal(item)
	if err != nil {
		log.Trace().Err(err).Str("key", ekey).Msg("redis write failed during marshal")
		return err
	}
	if err := e.redis.Set(rctx, ekey, data, ttl).Err(); err != nil {
		log.Trace().Err(err).Str("key", ekey).Msg("redis write failed")
	}
	log.Trace().Str("key", ekey).Msg("redis write ok")
	return nil
}

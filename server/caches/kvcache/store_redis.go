package kvcache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisStore implements the optional PubSubStore and HashStore capabilities
// in addition to Store.
var (
	_ Store       = (*RedisStore)(nil)
	_ PubSubStore = (*RedisStore)(nil)
	_ HashStore   = (*RedisStore)(nil)
)

// RedisStore adapts a *redis.Client to Store. A nil client is a no-op
// store: all reads miss and writes are discarded, preserving the
// historical nil-client local-only mode.
type RedisStore struct {
	// Timeout bounds each Redis operation (default 1s).
	Timeout time.Duration
	client  *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		Timeout: 1 * time.Second,
		client:  client,
	}
}

func (s *RedisStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if s.client == nil {
		return nil, false, nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	data, err := s.client.Get(rctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}
	return data, true, nil
}

func (s *RedisStore) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	ret := map[string][]byte{}
	if s.client == nil || len(keys) == 0 {
		return ret, nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	vals, err := s.client.MGet(rctx, keys...).Result()
	if err != nil {
		return nil, err
	}
	for i, val := range vals {
		if i >= len(keys) {
			break
		}
		// MGET returns nil for absent keys; present values decode as
		// string or []byte depending on client version.
		switch v := val.(type) {
		case string:
			ret[keys[i]] = []byte(v)
		case []byte:
			ret[keys[i]] = v
		}
	}
	return ret, nil
}

func (s *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if s.client == nil {
		return nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	if ttl < 0 {
		ttl = 0
	}
	return s.client.Set(rctx, key, value, ttl).Err()
}

func (s *RedisStore) Publish(ctx context.Context, channel string, payload []byte) error {
	if s.client == nil {
		return nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	return s.client.Publish(rctx, channel, payload).Err()
}

func (s *RedisStore) Subscribe(ctx context.Context, channel string) (Subscription, error) {
	if s.client == nil {
		// No backend to deliver from; hand back a subscription that stays
		// idle until ctx is canceled rather than an error that would spin a
		// caller's reconnect loop.
		return newDeadSubscription(ctx), nil
	}
	sub := s.client.Subscribe(ctx, channel)
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	if _, err := sub.Receive(rctx); err != nil {
		_ = sub.Close()
		return nil, err
	}
	return newRedisSubscription(ctx, sub), nil
}

func (s *RedisStore) HSet(ctx context.Context, key string, field string, value []byte) error {
	if s.client == nil {
		return nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	return s.client.HSet(rctx, key, field, value).Err()
}

func (s *RedisStore) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	ret := map[string][]byte{}
	if s.client == nil {
		return ret, nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	m, err := s.client.HGetAll(rctx, key).Result()
	if err != nil {
		return nil, err
	}
	for k, v := range m {
		ret[k] = []byte(v)
	}
	return ret, nil
}

// redisSubscription adapts a *redis.PubSub to Subscription, translating
// *redis.Message into kvcache.Message on a pump goroutine.
type redisSubscription struct {
	sub *redis.PubSub
	out chan Message
}

func newRedisSubscription(ctx context.Context, sub *redis.PubSub) *redisSubscription {
	rs := &redisSubscription{sub: sub, out: make(chan Message)}
	go rs.pump(ctx)
	return rs
}

func (rs *redisSubscription) Messages() <-chan Message { return rs.out }

func (rs *redisSubscription) Close() error { return rs.sub.Close() }

func (rs *redisSubscription) pump(ctx context.Context) {
	defer close(rs.out)
	ch := rs.sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			select {
			case rs.out <- Message{Channel: msg.Channel, Data: []byte(msg.Payload)}:
			case <-ctx.Done():
				return
			}
		}
	}
}

// deadSubscription delivers nothing and unblocks its consumer only when
// ctx is canceled. Used for the nil-client RedisStore.
type deadSubscription struct {
	out chan Message
}

func newDeadSubscription(ctx context.Context) *deadSubscription {
	ds := &deadSubscription{out: make(chan Message)}
	go func() {
		<-ctx.Done()
		close(ds.out)
	}()
	return ds
}

func (ds *deadSubscription) Messages() <-chan Message { return ds.out }

func (ds *deadSubscription) Close() error { return nil }

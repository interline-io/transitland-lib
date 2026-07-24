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
		// No backend to deliver from; hand back an idle subscription rather
		// than an error that would spin a caller's reconnect loop.
		return deadSubscription{}, nil
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

func (s *RedisStore) HSet(ctx context.Context, key string, field string, value string) error {
	if s.client == nil {
		return nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	return s.client.HSet(rctx, key, field, value).Err()
}

func (s *RedisStore) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	if s.client == nil {
		return map[string]string{}, nil
	}
	rctx, cancel := context.WithTimeout(ctx, s.Timeout)
	defer cancel()
	return s.client.HGetAll(rctx, key).Result()
}

// redisSubscription adapts a *redis.PubSub to Subscription, forwarding each
// message payload on a pump goroutine.
type redisSubscription struct {
	sub *redis.PubSub
	out chan []byte
}

func newRedisSubscription(ctx context.Context, sub *redis.PubSub) *redisSubscription {
	rs := &redisSubscription{sub: sub, out: make(chan []byte)}
	go rs.pump(ctx)
	return rs
}

func (rs *redisSubscription) Messages() <-chan []byte { return rs.out }

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
			case rs.out <- []byte(msg.Payload):
			case <-ctx.Done():
				return
			}
		}
	}
}

// deadSubscription delivers nothing; its nil message channel blocks the
// consumer, which shuts down via its own context. Used for the nil-client
// RedisStore, which has no backend to subscribe to.
type deadSubscription struct{}

func (deadSubscription) Messages() <-chan []byte { return nil }

func (deadSubscription) Close() error { return nil }

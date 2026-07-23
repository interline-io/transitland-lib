package kvcache

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
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
		// MGET returns nil for absent keys and strings for present ones.
		if sval, ok := val.(string); ok && i < len(keys) {
			ret[keys[i]] = []byte(sval)
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

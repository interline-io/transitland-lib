package kvcache

import (
	"context"
	"sync"
	"time"
)

// MemoryStore is an in-process Store for tests and single-process use.
// Two Caches sharing one MemoryStore emulate cross-process behavior.
type MemoryStore struct {
	now  func() time.Time
	lock sync.RWMutex
	m    map[string]memoryEntry
}

type memoryEntry struct {
	value     []byte
	expiresAt time.Time // zero = no expiry
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		now: time.Now,
		m:   map[string]memoryEntry{},
	}
}

func (s *MemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s.lock.RLock()
	ent, ok := s.m[key]
	s.lock.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if s.expired(ent) {
		s.deleteExpired(key)
		return nil, false, nil
	}
	// Copy so callers cannot mutate stored state, matching RedisStore.
	return append([]byte(nil), ent.value...), true, nil
}

func (s *MemoryStore) GetMulti(ctx context.Context, keys []string) (map[string][]byte, error) {
	ret := map[string][]byte{}
	var dead []string
	s.lock.RLock()
	for _, key := range keys {
		if ent, ok := s.m[key]; ok {
			if s.expired(ent) {
				dead = append(dead, key)
				continue
			}
			ret[key] = append([]byte(nil), ent.value...)
		}
	}
	s.lock.RUnlock()
	for _, key := range dead {
		s.deleteExpired(key)
	}
	return ret, nil
}

// deleteExpired evicts key if it is still expired under the write lock.
func (s *MemoryStore) deleteExpired(key string) {
	s.lock.Lock()
	if ent, ok := s.m[key]; ok && s.expired(ent) {
		delete(s.m, key)
	}
	s.lock.Unlock()
}

func (s *MemoryStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ent := memoryEntry{value: append([]byte(nil), value...)}
	if ttl > 0 {
		ent.expiresAt = s.now().Add(ttl)
	}
	s.lock.Lock()
	s.m[key] = ent
	s.lock.Unlock()
	return nil
}

func (s *MemoryStore) expired(ent memoryEntry) bool {
	return !ent.expiresAt.IsZero() && !ent.expiresAt.After(s.now())
}

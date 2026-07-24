// Package kvcache provides a generic two-tier cache: a per-process local
// tier in front of a shared Store such as Redis. It replaces the older
// ecache and rcache packages.
package kvcache

import (
	"context"
	"time"
)

// Store is a shared key-value tier for a Cache. Keys arrive fully
// namespaced; values are opaque byte envelopes. Implementations must be
// safe for concurrent use.
//
// Store is intentionally minimal. Future backend capabilities (deletion,
// pub/sub notification for the realtime store) will arrive as optional
// sibling interfaces discovered by type assertion, never as new methods
// on Store.
type Store interface {
	// Get returns the value for key. ok is false on a normal miss; err
	// reports a backend failure. Callers treat errors as misses.
	Get(ctx context.Context, key string) (value []byte, ok bool, err error)
	// GetMulti returns values for the subset of keys that are present.
	GetMulti(ctx context.Context, keys []string) (map[string][]byte, error)
	// Set stores value. The backend may evict the entry after ttl; a ttl
	// of zero or less means no backend-enforced expiry.
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

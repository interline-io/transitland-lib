// Package kvcache provides a generic two-tier cache: a per-process local
// tier in front of a shared Store such as Redis. It replaced the removed
// ecache and rcache packages; their storage key prefix and envelope are
// retained as the wire format.
package kvcache

import (
	"context"
	"time"
)

// Store is a shared key-value tier for a Cache. Keys arrive fully
// namespaced; values are opaque byte envelopes. Implementations must be
// safe for concurrent use.
//
// Store is intentionally minimal. Additional backend capabilities arrive
// as optional sibling interfaces (PubSubStore, HashStore) discovered by
// type assertion, never as new methods on Store.
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

// PubSubStore is an optional Store capability providing publish/subscribe
// over named channels, used by the realtime store to fan feed updates out
// to every process.
//
// Payloads are pointers, not data: a message names a key whose value the
// receiver re-reads from the Store. This keeps messages small enough for
// backends with tight limits (Postgres NOTIFY caps payloads at ~8KB).
type PubSubStore interface {
	// Publish sends payload to channel. Keep payloads small (a key or
	// topic, not the value it points at).
	Publish(ctx context.Context, channel string, payload []byte) error
	// Subscribe streams messages published to channel until the returned
	// Subscription is closed or ctx is canceled.
	Subscribe(ctx context.Context, channel string) (Subscription, error)
}

// Message is a single payload delivered to a Subscription.
type Message struct {
	Channel string
	Data    []byte
}

// Subscription is a live stream of published messages. Close releases the
// underlying backend connection.
type Subscription interface {
	Messages() <-chan Message
	Close() error
}

// HashStore is an optional Store capability providing field-addressable
// hash maps, used for small secondary indexes such as the GBFS bbox index.
type HashStore interface {
	HSet(ctx context.Context, key string, field string, value []byte) error
	HGetAll(ctx context.Context, key string) (map[string][]byte, error)
}

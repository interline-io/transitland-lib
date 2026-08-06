package rtfinder

import (
	"context"
	"sync"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
)

const (
	// lastTTL bounds how long a topic's last-seen payload survives in the store.
	lastTTL = 5 * time.Minute
	// missingTTL is how long a topic the store could not answer for is
	// remembered as absent. Roughly a feed's publish cadence: checking more
	// often cannot find anything new.
	missingTTL = 1 * time.Minute
	// reconnectDelay paces re-subscription attempts.
	reconnectDelay = 1 * time.Second
	// updatesChannel carries topic pointers to the notify-then-read listeners.
	updatesChannel = "rtfetch:updates"
)

// storeCache is the RT Cache backed by a kvcache.Store. It keeps decoded
// Sources locally and, when the store supports pub/sub, learns of updates
// from other processes via notify-then-read: a publish carries only the
// topic, and the receiver re-reads the payload from the store.
//
// This is deliberately non-generic. The two-tier local map plus the
// notify-then-read loop could be lifted into kvcache as a generic
// PushCache[V] parameterized by a decoder (bytes -> V); it lives here
// concretely for simplicity while rtfinder is the only consumer, and is
// worth promoting only once a second push-distributed decoded cache appears.
type storeCache struct {
	store   kvcache.Store
	pubsub  kvcache.PubSubStore // nil when the store has no pub/sub
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	lock    sync.Mutex
	sources map[string]*Source
	// missing records topics the store had nothing for, so a caller looping
	// over dozens of associated feeds does not re-read each one every time.
	missing map[string]time.Time
}

func newStoreCache(store kvcache.Store) *storeCache {
	ctx, cancel := context.WithCancel(context.Background())
	c := &storeCache{
		store:   store,
		ctx:     ctx,
		cancel:  cancel,
		sources: map[string]*Source{},
		missing: map[string]time.Time{},
	}
	if ps, ok := store.(kvcache.PubSubStore); ok {
		c.pubsub = ps
		c.wg.Add(1)
		go func() {
			defer c.wg.Done()
			c.subscribe()
		}()
	}
	return c
}

func (c *storeCache) AddData(ctx context.Context, topic string, data []byte) error {
	// Persisting and distributing a fetched update is best-effort work that
	// should outlive a canceled request or job, so it is bounded by the
	// cache's own context (canceled only on Close), not the caller's.
	rctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()
	if err := c.store.Set(rctx, lastKey(topic), data, lastTTL); err != nil {
		return err
	}
	if c.pubsub != nil {
		// Notify all listeners (including this process, via loopback) that
		// the topic changed; they re-read the payload from the store.
		return c.pubsub.Publish(rctx, updatesChannel, []byte(topic))
	}
	// No shared distribution: decode straight into the local map.
	s, err := c.decode(ctx, topic, data)
	if err != nil {
		return err
	}
	c.putSource(topic, s)
	return nil
}

func (c *storeCache) GetSource(ctx context.Context, topic string) (*Source, bool) {
	c.lock.Lock()
	if s, ok := c.sources[topic]; ok {
		c.lock.Unlock()
		return s, true
	}
	if at, ok := c.missing[topic]; ok && time.Since(at) < missingTTL {
		c.lock.Unlock()
		return nil, false
	}
	c.lock.Unlock()
	// Cold read from the shared store without holding the lock.
	s, err := c.loadFromStore(ctx, topic)
	if s == nil {
		// A caller that has gone away taught us nothing about the topic, so
		// its cancellation must not be remembered as an absence.
		if ctx.Err() != nil {
			log.For(ctx).Trace().Str("topic", topic).Msg("rtcache: topic read abandoned by caller")
			return nil, false
		}
		c.lock.Lock()
		c.missing[topic] = time.Now()
		c.lock.Unlock()
		// Both outcomes are remembered so a caller looping over every
		// associated feed does not re-read each one, but they are logged
		// apart: absent is a quiet feed, failed is the store not answering.
		if err != nil {
			log.For(ctx).Trace().Str("topic", topic).Dur("retry_after", missingTTL).Msg("rtcache: topic read failed, not retried until this expires")
		} else {
			log.For(ctx).Trace().Str("topic", topic).Dur("retry_after", missingTTL).Msg("rtcache: topic absent from store, not retried until this expires")
		}
		return nil, false
	}
	// Only store reads are logged. The local and remembered paths are map
	// hits, and a caller looping over every associated feed for every stop
	// time makes logging them thousands of lines a request.
	log.For(ctx).Trace().Str("topic", topic).Msg("rtcache: topic read")
	c.lock.Lock()
	// Double-check: a concurrent update may have inserted it meanwhile.
	if existing, ok := c.sources[topic]; ok {
		c.lock.Unlock()
		return existing, true
	}
	c.sources[topic] = s
	c.lock.Unlock()
	return s, true
}

func (c *storeCache) Close() error {
	c.cancel()
	c.wg.Wait()
	return nil
}

// subscribe keeps a subscription to the updates channel alive, re-reading
// each announced topic. It retries on failure rather than exiting, so a
// transient subscribe error does not silence updates until restart.
func (c *storeCache) subscribe() {
	for c.ctx.Err() == nil {
		sub, err := c.pubsub.Subscribe(c.ctx, updatesChannel)
		if err != nil {
			log.For(c.ctx).Error().Err(err).Msg("rtcache: error subscribing to updates")
			if !sleepCtx(c.ctx, reconnectDelay) {
				return
			}
			continue
		}
		c.drain(sub)
		_ = sub.Close()
		if !sleepCtx(c.ctx, reconnectDelay) {
			return
		}
	}
}

func (c *storeCache) drain(sub kvcache.Subscription) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case msg, ok := <-sub.Messages():
			if !ok {
				return
			}
			topic := string(msg)
			// Every RT fetch in the fleet announces on this one channel, so a
			// process must take only what it serves. Without this each process
			// reads and decodes every feed in the system on every fetch.
			if !c.watching(topic) {
				continue
			}
			if s, _ := c.loadFromStore(c.ctx, topic); s != nil {
				c.putSource(topic, s)
				log.For(c.ctx).Trace().Str("topic", topic).Msg("rtcache: processed update")
			}
		}
	}
}

// loadFromStore reads and decodes a topic's last payload.
//
// A nil Source with a nil error means the store definitely held nothing; a
// non-nil error means it could not say. Callers treat both as a miss, but must
// check their own context before concluding anything: a read cut short by the
// caller says nothing about the topic.
func (c *storeCache) loadFromStore(ctx context.Context, topic string) (*Source, error) {
	rctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	data, ok, err := c.store.Get(rctx, lastKey(topic))
	if err != nil {
		// A read that failed because the caller went away is not a store
		// problem, and logging it as one floods on every client disconnect.
		if ctx.Err() == nil {
			log.For(ctx).Error().Err(err).Str("topic", topic).Msg("rtcache: error reading last data")
		}
		return nil, err
	}
	if !ok || len(data) == 0 {
		return nil, nil
	}
	s, err := c.decode(ctx, topic, data)
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Msg("rtcache: error decoding last data")
		return nil, err
	}
	return s, nil
}

// watching reports whether this process holds a snapshot for topic, which is
// true only of topics a caller has asked for.
func (c *storeCache) watching(topic string) bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	_, ok := c.sources[topic]
	return ok
}

// putSource installs s as the local snapshot for topic, replacing any prior one.
func (c *storeCache) putSource(topic string, s *Source) {
	c.lock.Lock()
	c.sources[topic] = s
	c.lock.Unlock()
}

func (c *storeCache) decode(ctx context.Context, topic string, data []byte) (*Source, error) {
	s, err := NewSource(topic)
	if err != nil {
		return nil, err
	}
	if err := s.process(ctx, data); err != nil {
		return nil, err
	}
	return s, nil
}

func lastKey(topic string) string {
	return "rtfetch:last:" + topic
}

// sleepCtx waits for d or ctx cancellation, reporting false if canceled.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

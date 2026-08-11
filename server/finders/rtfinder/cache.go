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
	// storeReadTimeout bounds one shared read and decode.
	storeReadTimeout = 1 * time.Second
	// updatesChannel carries topic pointers to the notify-then-read listeners.
	updatesChannel = "rtfetch:updates"
)

// storeCache is the RT Cache backed by a kvcache.Store. Decoded Sources are
// held in a kvcache.Cache and, when the store supports pub/sub, updates from
// other processes arrive by notify-then-read: a publish carries only the
// topic, and the receiver re-reads the payload from the store.
//
// The kvcache.Cache is local-tier only. Its shared tier would be a JSON
// envelope this cache wrote itself, whereas the payload here is raw protobuf
// written by the fetcher under its own key, and a decoded Source is not
// something worth sharing between processes anyway — re-decoding the protobuf
// is cheaper than encoding the result. So reading and decoding that payload is
// the refresh function, and distribution stays with pub/sub.
type storeCache struct {
	store   kvcache.Store
	pubsub  kvcache.PubSubStore // nil when the store has no pub/sub
	sources *kvcache.Cache[string, *Source]
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

func newStoreCache(store kvcache.Store) *storeCache {
	ctx, cancel := context.WithCancel(context.Background())
	c := &storeCache{
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}
	c.sources = kvcache.NewRefreshCache[string, *Source](nil, "rtsource", c.readTopic)
	// A decoded Source is only as good as the payload behind it, which the
	// store drops after lastTTL. Nothing rechecks in the background — updates
	// arrive by publication — so Recheck never has to come due.
	c.sources.Expires = lastTTL
	c.sources.Recheck = lastTTL
	c.sources.NegativeTTL = missingTTL
	c.sources.RefreshTimeout = storeReadTimeout
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
	// No shared distribution: decode straight into the local tier, replacing
	// any record of absence.
	s, err := c.decode(ctx, topic, data)
	if err != nil {
		return err
	}
	return c.sources.Set(ctx, topic, s)
}

func (c *storeCache) GetSource(ctx context.Context, topic string) (*Source, bool) {
	// A caller that has already gone away is shed rather than started. A read is
	// detached from its caller by design, so beginning one here would spend a
	// store round trip on a result nobody waits for — and a client disconnecting
	// mid-request leaves a resolver still looping over dozens of topics.
	if ctx.Err() != nil {
		return nil, false
	}
	return c.sources.Get(ctx, topic)
}

func (c *storeCache) Close() error {
	c.cancel()
	c.wg.Wait()
	return nil
}

// readTopic reads and decodes a topic's last payload. It is the cache's
// refresh function, so concurrent callers for one topic share a single call
// and its result is stored for them.
//
// Every failure reports kvcache.ErrNotFound, which records the topic as absent
// for NegativeTTL. A caller loops over every feed associated with a feed
// version, so a store that cannot answer must not be asked again for each of
// them; the cases are logged apart, since absent is a quiet feed and failed is
// the store not answering.
func (c *storeCache) readTopic(ctx context.Context, topic string) (*Source, error) {
	data, ok, err := c.store.Get(ctx, lastKey(topic))
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Dur("retry_after", missingTTL).Msg("rtcache: topic read failed, not retried until this expires")
		return nil, kvcache.ErrNotFound
	}
	if !ok || len(data) == 0 {
		log.For(ctx).Trace().Str("topic", topic).Dur("retry_after", missingTTL).Msg("rtcache: topic absent from store, not retried until this expires")
		return nil, kvcache.ErrNotFound
	}
	s, err := c.decode(ctx, topic, data)
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Dur("retry_after", missingTTL).Msg("rtcache: topic decode failed, not retried until this expires")
		return nil, kvcache.ErrNotFound
	}
	// Only store reads are logged, and only the one that happened: local hits
	// and the callers that shared this read are not each an event.
	log.For(ctx).Trace().Str("topic", topic).Msg("rtcache: topic read")
	return s, nil
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
			if s, err := c.sources.Refresh(c.ctx, topic); err == nil && s != nil {
				log.For(c.ctx).Trace().Str("topic", topic).Msg("rtcache: processed update")
			}
		}
	}
}

// watching reports whether this process holds an entry for topic, which is
// true only of topics a caller has asked for. A topic found absent counts:
// somebody wants it, so an announcement that it now has data is worth taking
// rather than waiting out the rest of missingTTL.
func (c *storeCache) watching(topic string) bool {
	return c.sources.Has(topic)
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

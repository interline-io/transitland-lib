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
	// reconnectDelay paces re-subscription attempts.
	reconnectDelay = 1 * time.Second
	// updatesChannel carries topic pointers to the notify-then-read listeners.
	updatesChannel = "rtfetch:updates"
)

// storeCache is the RT Cache backed by a kvcache.Store. It keeps decoded
// Sources locally and, when the store supports pub/sub, learns of updates
// from other processes via notify-then-read: a publish carries only the
// topic, and the receiver re-reads the payload from the store.
type storeCache struct {
	store   kvcache.Store
	pubsub  kvcache.PubSubStore // nil when the store has no pub/sub
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	lock    sync.Mutex
	sources map[string]*Source
}

func newStoreCache(store kvcache.Store) *storeCache {
	ctx, cancel := context.WithCancel(context.Background())
	c := &storeCache{
		store:   store,
		ctx:     ctx,
		cancel:  cancel,
		sources: map[string]*Source{},
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
	// No shared distribution: decode straight into the local Source.
	return c.setSource(ctx, topic, data)
}

func (c *storeCache) GetSource(ctx context.Context, topic string) (*Source, bool) {
	c.lock.Lock()
	if s, ok := c.sources[topic]; ok {
		c.lock.Unlock()
		return s, true
	}
	c.lock.Unlock()
	// Cold read from the shared store without holding the lock.
	s := c.loadFromStore(ctx, topic)
	if s == nil {
		return nil, false
	}
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
			topic := string(msg.Data)
			if s := c.loadFromStore(c.ctx, topic); s != nil {
				c.lock.Lock()
				c.sources[topic] = s
				c.lock.Unlock()
				log.For(c.ctx).Trace().Str("topic", topic).Msg("rtcache: processed update")
			}
		}
	}
}

// loadFromStore reads a topic's last payload and decodes it into a fresh
// Source. It returns nil on a miss or any error (treated as a miss).
func (c *storeCache) loadFromStore(ctx context.Context, topic string) *Source {
	rctx, cancel := context.WithTimeout(c.ctx, 1*time.Second)
	defer cancel()
	data, ok, err := c.store.Get(rctx, lastKey(topic))
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Msg("rtcache: error reading last data")
		return nil
	}
	if !ok || len(data) == 0 {
		return nil
	}
	s, err := c.decode(ctx, topic, data)
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Msg("rtcache: error decoding last data")
		return nil
	}
	return s
}

// setSource decodes data into a fresh Source and installs it locally,
// replacing any prior snapshot for the topic.
func (c *storeCache) setSource(ctx context.Context, topic string, data []byte) error {
	s, err := c.decode(ctx, topic, data)
	if err != nil {
		return err
	}
	if s == nil {
		return nil
	}
	c.lock.Lock()
	c.sources[topic] = s
	c.lock.Unlock()
	return nil
}

func (c *storeCache) decode(ctx context.Context, topic string, data []byte) (*Source, error) {
	if len(data) == 0 {
		return nil, nil
	}
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

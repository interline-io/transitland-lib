package rtfinder

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/rt/pb"
)

type listener struct {
	source *Source
	ctx    context.Context
	cancel context.CancelFunc
}

func newListener(s *Source, parent context.Context) *listener {
	cc, cf := context.WithCancel(parent)
	return &listener{
		source: s,
		ctx:    cc,
		cancel: cf,
	}
}

type RedisCache struct {
	ctx       context.Context
	lock      sync.Mutex
	client    *redis.Client
	listeners map[string]*listener
}

func NewRedisCache(client *redis.Client) *RedisCache {
	ctx := context.Background()
	f := RedisCache{
		client:    client,
		listeners: map[string]*listener{},
		ctx:       ctx,
	}
	return &f
}

func (f *RedisCache) Subscribe() (chan string, func()) {
	ch := make(chan string, 100)
	subCtx, subCancel := context.WithCancel(f.ctx)
	// Use Redis PSUBSCRIBE to watch for all RT data updates
	psub := f.client.PSubscribe(subCtx, "rtfetch:sub:*")
	go func() {
		defer psub.Close()
		pch := psub.Channel()
		for {
			select {
			case <-subCtx.Done():
				return
			case msg, ok := <-pch:
				if !ok {
					return
				}
				// Extract topic from channel name (strip "rtfetch:sub:" prefix)
				topic := strings.TrimPrefix(msg.Channel, "rtfetch:sub:")
				// Also ensure a listener exists so GetSource/GetVehiclePositions works
				f.ensureListener(subCtx, topic)
				select {
				case ch <- topic:
				default:
				}
			}
		}
	}()
	cancel := func() {
		subCancel()
		close(ch)
	}
	return ch, cancel
}

func (f *RedisCache) GetSourceKeys() []string {
	// Scan Redis for cached RT data keys
	rctx, cc := context.WithTimeout(f.ctx, 2*time.Second)
	defer cc()
	prefix := "rtfetch:last:"
	var keys []string
	iter := f.client.Scan(rctx, 0, prefix+"*", 1000).Iterator()
	for iter.Next(rctx) {
		topic := strings.TrimPrefix(iter.Val(), prefix)
		keys = append(keys, topic)
	}
	return keys
}

func (f *RedisCache) ensureListener(ctx context.Context, topic string) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if _, ok := f.listeners[topic]; ok {
		return
	}
	a, err := f.startListener(ctx, topic)
	if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Msg("cache: error creating listener")
		return
	}
	f.listeners[topic] = a
}

func (f *RedisCache) GetSource(ctx context.Context, topic string) (*Source, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if a, ok := f.listeners[topic]; ok {
		return a.source, true
	}
	a, err := f.startListener(ctx, topic)
	if err != nil {
		return nil, false
	}
	f.listeners[topic] = a
	return a.source, true
}

func (f *RedisCache) AddFeedMessage(ctx context.Context, topic string, rtmsg *pb.FeedMessage) error {
	return nil
}

func (f *RedisCache) AddData(ctx context.Context, topic string, data []byte) error {
	rctx, cc := context.WithTimeout(f.ctx, 5*time.Second)
	defer cc()
	// Set last seen value with 5 min ttl
	if err := f.client.Set(rctx, lastKey(topic), data, 5*time.Minute).Err(); err != nil {
		return err
	}
	// Publish to subscribers
	if err := f.client.Publish(rctx, subKey(topic), data).Err(); err != nil {
		return err
	}
	log.For(ctx).Trace().Str("topic", topic).Int("bytes", len(data)).Msg("cache: added data")
	return nil
}

func (f *RedisCache) Close() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	for k, ls := range f.listeners {
		ls.cancel()
		delete(f.listeners, k)
	}
	return nil
}

func lastKey(topic string) string {
	return fmt.Sprintf("rtfetch:last:%s", topic)
}

func subKey(topic string) string {
	return fmt.Sprintf("rtfetch:sub:%s", topic)
}

func (f *RedisCache) startListener(ctx context.Context, topic string) (*listener, error) {
	// Create new source
	s, err := NewSource(topic)
	if err != nil {
		return nil, err
	}
	// Add subscription for future data
	ls := newListener(s, f.ctx)
	go func(client *redis.Client, topic string, lst *listener) {
		sub := client.Subscribe(lst.ctx, subKey(topic))
		defer sub.Close()
		subch := sub.Channel()
		for rmsg := range subch {
			if err := s.process(ctx, []byte(rmsg.Payload)); err != nil {
				log.For(ctx).Error().Err(err).Str("topic", topic).Int("bytes", len(rmsg.Payload)).Msg("cache: error processing update")
			} else {
				log.For(ctx).Trace().Str("topic", topic).Int("bytes", len(rmsg.Payload)).Msg("cache: processed update")
			}
		}
	}(f.client, topic, ls)
	log.For(ctx).Trace().Str("topic", topic).Msgf("cache: listener created")
	// get the first message
	rctx, cc := context.WithTimeout(f.ctx, 1*time.Second)
	defer cc()
	lastData := f.client.Get(rctx, lastKey(topic))
	if err := lastData.Err(); err == redis.Nil {
		// ok
	} else if err != nil {
		// also ok, hope we get data on future updates
		log.For(ctx).Error().Err(err).Str("topic", topic).Msg("cache: error getting last data for topic")
	} else {
		lb, _ := lastData.Bytes()
		s.process(ctx, lb)
	}
	return ls, nil
}

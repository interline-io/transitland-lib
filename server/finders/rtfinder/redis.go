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

type RedisCache struct {
	ctx     context.Context
	cancel  context.CancelFunc
	lock    sync.Mutex
	client  *redis.Client
	sources map[string]*Source
}

func NewRedisCache(client *redis.Client) *RedisCache {
	ctx, cancel := context.WithCancel(context.Background())
	f := RedisCache{
		client:  client,
		sources: map[string]*Source{},
		ctx:     ctx,
		cancel:  cancel,
	}
	// Start a single subscription for all RT topics
	go f.subscribeAll()
	return &f
}

func (f *RedisCache) GetSource(ctx context.Context, topic string) (*Source, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	if s, ok := f.sources[topic]; ok {
		return s, true
	}
	// Fetch last known data from Redis
	s, err := f.fetchLast(ctx, topic)
	if err != nil {
		return nil, false
	}
	if s != nil {
		f.sources[topic] = s
		return s, true
	}
	return nil, false
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
	f.cancel()
	return nil
}

func lastKey(topic string) string {
	return fmt.Sprintf("rtfetch:last:%s", topic)
}

func subKey(topic string) string {
	return fmt.Sprintf("rtfetch:sub:%s", topic)
}

// topicFromSubKey extracts the topic from a subscription channel key.
func topicFromSubKey(channel string) string {
	return strings.TrimPrefix(channel, "rtfetch:sub:")
}

// subscribeAll uses a single PSubscribe connection for all RT topics.
func (f *RedisCache) subscribeAll() {
	sub := f.client.PSubscribe(f.ctx, "rtfetch:sub:*")
	defer sub.Close()
	ch := sub.Channel()
	for msg := range ch {
		topic := topicFromSubKey(msg.Channel)
		if err := f.processMessage(topic, []byte(msg.Payload)); err != nil {
			log.For(f.ctx).Error().Err(err).Str("topic", topic).Int("bytes", len(msg.Payload)).Msg("cache: error processing update")
		} else {
			log.For(f.ctx).Trace().Str("topic", topic).Int("bytes", len(msg.Payload)).Msg("cache: processed update")
		}
	}
}

// processMessage updates or creates the Source for a given topic.
func (f *RedisCache) processMessage(topic string, data []byte) error {
	f.lock.Lock()
	s, ok := f.sources[topic]
	if !ok {
		var err error
		s, err = NewSource(topic)
		if err != nil {
			f.lock.Unlock()
			return err
		}
		f.sources[topic] = s
	}
	f.lock.Unlock()
	return s.process(f.ctx, data)
}

// fetchLast retrieves the last cached data from Redis for a topic.
func (f *RedisCache) fetchLast(ctx context.Context, topic string) (*Source, error) {
	rctx, cc := context.WithTimeout(f.ctx, 1*time.Second)
	defer cc()
	lastData := f.client.Get(rctx, lastKey(topic))
	if err := lastData.Err(); err == redis.Nil {
		return nil, nil
	} else if err != nil {
		log.For(ctx).Error().Err(err).Str("topic", topic).Msg("cache: error getting last data for topic")
		return nil, nil
	}
	lb, _ := lastData.Bytes()
	if len(lb) == 0 {
		return nil, nil
	}
	s, err := NewSource(topic)
	if err != nil {
		return nil, err
	}
	if err := s.process(ctx, lb); err != nil {
		return nil, err
	}
	return s, nil
}

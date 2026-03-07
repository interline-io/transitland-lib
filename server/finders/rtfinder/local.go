package rtfinder

import (
	"context"
	"sync"

	"github.com/interline-io/transitland-lib/rt/pb"
)

type LocalCache struct {
	lock        sync.Mutex
	sources     map[string]*Source
	subscribers map[chan string]struct{}
}

func NewLocalCache() *LocalCache {
	return &LocalCache{
		sources:     map[string]*Source{},
		subscribers: map[chan string]struct{}{},
	}
}

func (f *LocalCache) Subscribe() (chan string, func()) {
	f.lock.Lock()
	defer f.lock.Unlock()
	ch := make(chan string, 100)
	f.subscribers[ch] = struct{}{}
	cancel := func() {
		f.lock.Lock()
		defer f.lock.Unlock()
		delete(f.subscribers, ch)
		close(ch)
	}
	return ch, cancel
}

func (f *LocalCache) notifySubscribers(topic string) {
	for ch := range f.subscribers {
		select {
		case ch <- topic:
		default:
		}
	}
}

func (f *LocalCache) GetSource(ctx context.Context, topic string) (*Source, bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	a, ok := f.sources[topic]
	if ok {
		return a, true
	}
	return nil, false
}

func (f *LocalCache) AddFeedMessage(ctx context.Context, topic string, rtmsg *pb.FeedMessage) error {
	return nil
}

func (f *LocalCache) AddData(ctx context.Context, topic string, data []byte) error {
	f.lock.Lock()
	defer f.lock.Unlock()
	s, ok := f.sources[topic]
	if !ok {
		s, _ = NewSource(topic)
		f.sources[topic] = s
	}
	if err := s.process(ctx, data); err != nil {
		return err
	}
	f.notifySubscribers(topic)
	return nil
}

func (f *LocalCache) Close() error {
	return nil
}

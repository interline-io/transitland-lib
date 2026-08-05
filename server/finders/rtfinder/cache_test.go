package rtfinder

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

var feeds = []string{"BA", "SF", "AC", "CT"}

func TestStoreCache_Memory(t *testing.T) {
	testCache(t, newStoreCache(kvcache.NewMemoryStore()))
}

func TestStoreCache_Redis(t *testing.T) {
	if a, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip(a)
		return
	}
	client := testutil.MustOpenTestRedisClient(t)
	testCache(t, newStoreCache(kvcache.NewRedisStore(client)))
}

// TestStoreCache_PubSubUpdate exercises the notify-then-read distribution
// path: a published notification must refresh an already-cached Source that
// a plain GetSource (a local map hit) would not.
func TestStoreCache_PubSubUpdate(t *testing.T) {
	if a, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip(a)
		return
	}
	ctx := context.Background()
	store := kvcache.NewRedisStore(testutil.MustOpenTestRedisClient(t))
	c := newStoreCache(store)
	defer c.Close()

	topic := fmt.Sprintf("rtdist-%d", time.Now().UnixNano())
	const v1, v2 = uint64(1000), uint64(2000)

	// Seed v1 and pull it into the local map with a cold read.
	if err := store.Set(ctx, lastKey(topic), mkRTData(v1), lastTTL); err != nil {
		t.Fatal(err)
	}
	if s, ok := c.GetSource(ctx, topic); !ok || s.GetTimestamp() != v1 {
		t.Fatalf("cold read failed: ok=%v", ok)
	}

	// Replace the stored payload without notifying: the cached copy stays v1.
	if err := store.Set(ctx, lastKey(topic), mkRTData(v2), lastTTL); err != nil {
		t.Fatal(err)
	}
	if s, ok := c.GetSource(ctx, topic); !ok || s.GetTimestamp() != v1 {
		t.Fatal("cached source changed without a notification")
	}

	// A notification drives the subscribe goroutine to re-read v2. Republish
	// each tick so early attempts that race subscription setup are retried.
	assert.Eventually(t, func() bool {
		_ = store.Publish(ctx, updatesChannel, []byte(topic))
		s, ok := c.GetSource(ctx, topic)
		return ok && s.GetTimestamp() == v2
	}, 5*time.Second, 100*time.Millisecond, "notification should refresh cached source to v2")
}

func mkRTData(ts uint64) []byte {
	v := "2.0"
	data, _ := proto.Marshal(&pb.FeedMessage{Header: &pb.FeedHeader{GtfsRealtimeVersion: &v, Timestamp: &ts}})
	return data
}

func testCache(t *testing.T, rtCache Cache) {
	ctx := context.Background()
	var topics []string
	for _, feed := range feeds {
		topics = append(topics, fmt.Sprintf("%s-%d", feed, time.Now().UnixNano()))
	}
	for _, topic := range topics {
		if err := rtCache.AddData(ctx, topic, mkRTData(uint64(time.Now().UnixNano()))); err != nil {
			t.Fatal(err)
		}
	}
	var found []uint64
	for _, topic := range topics {
		if a, ok := rtCache.GetSource(ctx, topic); ok {
			found = append(found, a.GetTimestamp())
		}
	}
	rtCache.Close()
	if len(found) != len(feeds) {
		t.Errorf("got %d items, expected %d", len(found), len(feeds))
	}
}

// The updates channel carries every RT fetch in the fleet — 10 to 100 a second
// in production. A process must ignore announcements for topics it has not been
// asked for, or it reads and decodes every feed in the system.
func TestStoreCache_IgnoresUnwatchedTopics(t *testing.T) {
	if a, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip(a)
		return
	}
	ctx := context.Background()
	store := kvcache.NewRedisStore(testutil.MustOpenTestRedisClient(t))
	c := newStoreCache(store)
	defer c.Close()

	now := time.Now().UnixNano()
	watched := fmt.Sprintf("rtwatched-%d", now)
	unwatched := fmt.Sprintf("rtunwatched-%d", now)
	const v1, v2 = uint64(1000), uint64(2000)

	// Demand one topic so it is watched, and leave the other only in the store.
	if err := store.Set(ctx, lastKey(watched), mkRTData(v1), lastTTL); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.GetSource(ctx, watched); !ok {
		t.Fatal("cold read failed")
	}
	if err := store.Set(ctx, lastKey(watched), mkRTData(v2), lastTTL); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, lastKey(unwatched), mkRTData(v1), lastTTL); err != nil {
		t.Fatal(err)
	}

	// The watched topic refreshing proves the subscription delivered in this
	// window, which is what makes the unwatched assertion below meaningful
	// rather than a race that simply never fired.
	assert.Eventually(t, func() bool {
		_ = store.Publish(ctx, updatesChannel, []byte(unwatched))
		_ = store.Publish(ctx, updatesChannel, []byte(watched))
		s, ok := c.GetSource(ctx, watched)
		return ok && s.GetTimestamp() == v2
	}, 5*time.Second, 100*time.Millisecond, "watched topic should refresh")

	assert.False(t, c.watching(unwatched), "an undemanded topic must never be read from the store")
}

// countingStore counts reads so a test can assert what a sequence of lookups
// costs in round trips.
type countingStore struct {
	*kvcache.MemoryStore
	gets atomic.Int64
}

func (s *countingStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	s.gets.Add(1)
	return s.MemoryStore.Get(ctx, key)
}

// Callers loop over every RT feed associated with a feed version, and most of
// them carry nothing for a given trip. Re-reading each absent topic on every
// lookup is what turned one query into thousands of round trips.
func TestStoreCache_RemembersMissingTopics(t *testing.T) {
	store := &countingStore{MemoryStore: kvcache.NewMemoryStore()}
	c := newStoreCache(store)
	defer c.Close()

	ctx := context.Background()
	const topic = "rtabsent"
	for i := 0; i < 50; i++ {
		if _, ok := c.GetSource(ctx, topic); ok {
			t.Fatal("an absent topic must not resolve")
		}
	}
	assert.EqualValues(t, 1, store.gets.Load(), "an absent topic should be read once, not once per lookup")

	// A topic that arrives is served straight away rather than waiting out the
	// TTL: the local snapshot is consulted before the record of absence.
	require.NoError(t, c.AddData(ctx, topic, mkRTData(1234)))
	s, ok := c.GetSource(ctx, topic)
	require.True(t, ok)
	assert.EqualValues(t, uint64(1234), s.GetTimestamp())
}

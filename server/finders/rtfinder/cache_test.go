package rtfinder

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
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

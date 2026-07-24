package rtfinder

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/testutil"
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

func testCache(t *testing.T, rtCache Cache) {
	ctx := context.Background()
	var topics []string
	for _, feed := range feeds {
		topics = append(topics, fmt.Sprintf("%s-%d", feed, time.Now().UnixNano()))
	}
	for _, topic := range topics {
		v := "2.0"
		ts := uint64(time.Now().UnixNano())
		rtdata, _ := proto.Marshal(&pb.FeedMessage{Header: &pb.FeedHeader{GtfsRealtimeVersion: &v, Timestamp: &ts}})
		if err := rtCache.AddData(ctx, topic, rtdata); err != nil {
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

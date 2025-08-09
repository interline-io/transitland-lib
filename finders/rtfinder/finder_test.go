package rtfinder

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/rt/pb"
	"google.golang.org/protobuf/proto"
)

var (
	feeds = []string{"BA", "SF", "AC", "CT"}
)

func testCache(t *testing.T, rtCache Cache) {
	ctx := context.Background()
	var topics []string
	for _, feed := range feeds {
		topic := fmt.Sprintf("%s-%d", feed, time.Now().UnixNano())
		topics = append(topics, topic)
	}
	// Add data
	for _, topic := range topics {
		v := "2.0"
		ts := uint64(time.Now().UnixNano())
		rtdata, _ := proto.Marshal(&pb.FeedMessage{Header: &pb.FeedHeader{GtfsRealtimeVersion: &v, Timestamp: &ts}})
		rtCache.AddData(ctx, topic, rtdata)
	}
	found := []uint64{}
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

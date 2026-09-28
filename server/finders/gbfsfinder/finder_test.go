package gbfsfinder

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
)

func TestGbfsFinder(t *testing.T) {
	gbf := NewFinder(kvcache.NewMemoryStore())
	testSetupGbfs(gbf)

	tcs := []struct {
		p           tlxy.Point
		r           float64
		expectBikes int
		expectDocks int
	}{
		{tlxy.Point{Lon: -122.396185, Lat: 37.793412}, 1000, 60, 30},
		{tlxy.Point{Lon: -122.396185, Lat: 37.793412}, 500, 20, 10},
		{tlxy.Point{Lon: -122.41926403193607, Lat: 37.77508791392819}, 1000, 34, 27},
		{tlxy.Point{Lon: -120.99515, Lat: 37.640}, 1000, 0, 0},
	}

	for _, tc := range tcs {
		t.Run("FindBikes", func(t *testing.T) {
			where := model.GbfsBikeRequest{Near: &model.PointRadius{Lon: tc.p.Lon, Lat: tc.p.Lat, Radius: tc.r}}
			bikes, err := gbf.FindBikes(context.Background(), nil, &where)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.expectBikes, len(bikes), "bike count")
		})
	}

	for _, tc := range tcs {
		t.Run("FindBikes", func(t *testing.T) {
			where := model.GbfsDockRequest{Near: &model.PointRadius{Lon: tc.p.Lon, Lat: tc.p.Lat, Radius: tc.r}}
			docks, err := gbf.FindDocks(context.Background(), nil, &where)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.expectDocks, len(docks), "dock count")
		})
	}

}

func TestGbfsFinder_GetFeed(t *testing.T) {
	ctx := context.Background()
	gbf := NewFinder(kvcache.NewMemoryStore())
	defer gbf.Close()
	if err := testSetupGbfs(gbf); err != nil {
		t.Fatal(err)
	}

	feed, ok := gbf.GetFeed(ctx, "gbfs-test")
	assert.True(t, ok)
	assert.NotEmpty(t, feed.GbfsFeed.StationInformation)
	assert.NotEmpty(t, feed.Bikes)

	_, ok = gbf.GetFeed(ctx, "gbfs-other")
	assert.False(t, ok)
}

// A process that did not run a fetch has to see it anyway. Without the
// announcement a second finder serves its first read until the 24 hour expiry.
func TestGbfsFinder_FollowsOtherProcesses(t *testing.T) {
	if a, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip(a)
		return
	}
	ctx := context.Background()
	client := testutil.MustOpenTestRedisClient(t)
	writer := NewFinder(kvcache.NewRedisStore(client))
	defer writer.Close()
	reader := NewFinder(kvcache.NewRedisStore(client))
	defer reader.Close()

	topic := fmt.Sprintf("gbfs-follow-%d", time.Now().UnixNano())
	named := func(name string) gbfs.GbfsFeed {
		return gbfs.GbfsFeed{SystemInformation: &gbfs.SystemInformation{Name: tt.NewString(name)}}
	}
	// Rewritten on each tick, so an announcement that races the reader's
	// subscription setup is retried.
	converges := func(name string) {
		t.Helper()
		assert.EventuallyWithT(t, func(c *assert.CollectT) {
			assert.NoError(c, writer.AddData(ctx, topic, named(name)))
			feed, ok := reader.GetFeed(ctx, topic)
			if assert.True(c, ok) {
				assert.Equal(c, name, feed.GbfsFeed.SystemInformation.Name.Val)
			}
		}, 5*time.Second, 100*time.Millisecond, "the reader should adopt %q", name)
	}

	if err := writer.AddData(ctx, topic, named("first")); err != nil {
		t.Fatal(err)
	}
	feed, ok := reader.GetFeed(ctx, topic)
	assert.True(t, ok)
	assert.Equal(t, "first", feed.GbfsFeed.SystemInformation.Name.Val)
	converges("second")

	// Every fetch in the fleet is announced, and a finder re-reads only what it
	// holds. Announcements arrive in order, so once a later one is adopted the
	// unheld topic's has been seen and passed over.
	unheld := topic + "-unheld"
	if err := writer.AddData(ctx, unheld, named("unheld")); err != nil {
		t.Fatal(err)
	}
	converges("third")
	assert.False(t, reader.feeds.Contains(unheld), "an announcement does not load a topic this finder never read")
}

// orderStore records the writes a finder makes. It announces nothing.
type orderStore struct {
	*kvcache.MemoryStore
	ops []string
}

func (s *orderStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	s.ops = append(s.ops, "set")
	return s.MemoryStore.Set(ctx, key, value, ttl)
}

func (s *orderStore) HSet(ctx context.Context, key string, field string, value string) error {
	s.ops = append(s.ops, "hset")
	return s.MemoryStore.HSet(ctx, key, field, value)
}

func (s *orderStore) Publish(ctx context.Context, channel string, payload []byte) error {
	s.ops = append(s.ops, "publish")
	return nil
}

func (s *orderStore) Subscribe(ctx context.Context, channel string) (kvcache.Subscription, error) {
	return idleSubscription{}, nil
}

type idleSubscription struct{}

func (idleSubscription) Messages() <-chan []byte { return nil }

func (idleSubscription) Close() error { return nil }

// The announcement goes last. A finder re-reading on it has to find the new
// data, and a failed one should cost other processes their freshness, not the
// data or its index.
func TestGbfsFinder_AnnouncesLast(t *testing.T) {
	store := &orderStore{MemoryStore: kvcache.NewMemoryStore()}
	gbf := NewFinder(store)
	defer gbf.Close()
	if err := gbf.AddData(context.Background(), "gbfs-test", gbfs.GbfsFeed{}); err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, []string{"set", "hset", "hset", "publish"}, store.ops)
}

// With pub/sub a write goes to the store only. A process that fetches but
// never reads — a fetch worker — holds nothing, so announcements of its own
// writes pass it by, and it reads a system back from the store when asked.
func TestGbfsFinder_WriterHoldsNothing(t *testing.T) {
	ctx := context.Background()
	store := &orderStore{MemoryStore: kvcache.NewMemoryStore()}
	gbf := NewFinder(store)
	defer gbf.Close()
	sf := gbfs.GbfsFeed{SystemInformation: &gbfs.SystemInformation{Name: tt.NewString("Bay Wheels")}}
	if err := gbf.AddData(ctx, "gbfs-test", sf); err != nil {
		t.Fatal(err)
	}
	assert.False(t, gbf.feeds.Contains("gbfs-test"))

	feed, ok := gbf.GetFeed(ctx, "gbfs-test")
	if assert.True(t, ok) {
		assert.Equal(t, "Bay Wheels", feed.GbfsFeed.SystemInformation.Name.Val)
	}
}

func testSetupGbfs(gbf model.GbfsFinder) error {
	// Setup
	sourceFeedId := "gbfs-test"
	ts := httptest.NewServer(gbfs.NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer ts.Close()
	opts := gbfs.Options{}
	opts.FeedURL = fmt.Sprintf("%s/%s", ts.URL, "gbfs.json")
	opts.AllowHTTPFetchUnfiltered = true
	feeds, _, err := gbfs.Fetch(context.Background(), nil, opts)
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		if err := gbf.AddData(context.Background(), sourceFeedId, feed); err != nil {
			return err
		}
	}
	return nil
}

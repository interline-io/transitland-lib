package gbfsfinder

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/transitland-lib/internal/gbfs"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
)

func TestGbfsFinder(t *testing.T) {
	if a, ok := testutil.CheckTestRedisClient(); !ok {
		t.Skip(a)
		return
	}
	client := testutil.MustOpenTestRedisClient(t)
	gbf := NewFinder(client)
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

func testSetupGbfs(gbf model.GbfsFinder) error {
	// Setup
	sourceFeedId := "gbfs-test"
	ts := httptest.NewServer(gbfs.NewTestGbfsServer("en", testdata.Path("server/gbfs")))
	defer ts.Close()
	opts := gbfs.Options{}
	opts.FeedURL = fmt.Sprintf("%s/%s", ts.URL, "gbfs.json")
	feeds, _, err := gbfs.Fetch(context.Background(), nil, opts)
	if err != nil {
		return err
	}
	for _, feed := range feeds {
		key := fmt.Sprintf("%s:%s", sourceFeedId, feed.SystemInformation.Language.Val)
		gbf.AddData(context.Background(), key, feed)
	}
	return nil
}

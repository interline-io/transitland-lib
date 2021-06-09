package rest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/resolvers"
)

func TestFeedRequest(t *testing.T) {
	cfg := restConfig{srv: resolvers.NewServer()}
	// fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	testcases := []testRest{
		{"basic", &FeedRequest{}, "", "feeds.#.onestop_id", []string{"CT", "BA"}, 0},
		{"onestop_id", &FeedRequest{OnestopID: "BA"}, "", "feeds.#.onestop_id", []string{"BA"}, 0},
		// {"lat,lon,r:100,limit:100", "feeds", &FeedRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 100.0}, "", 2, bartosid},
		// {"lat,lon,r:1000,limit:100", "feeds", &FeedRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 1000.0}, "", 5, bartosid},
	}
	for _, tc := range testcases {
		testquery(t, cfg, tc)
	}
}

package rest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/resolvers"
)

func TestFeedRequest(t *testing.T) {
	cfg := restConfig{srv: resolvers.NewServer()}
	// fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	testcases := []testRest{
		{"basic", &FeedRequest{}, "", "feeds.#.onestop_id", []string{"CT", "BA", "BA~rt", "test"}, 0},
		{"onestop_id", &FeedRequest{OnestopID: "BA"}, "", "feeds.#.onestop_id", []string{"BA"}, 0},
		{"spec", &FeedRequest{Spec: "gtfs-rt"}, "", "feeds.#.onestop_id", []string{"BA~rt"}, 0},
		{"search", &FeedRequest{Search: "ba"}, "", "feeds.#.onestop_id", []string{"BA~rt", "BA"}, 0},
		{"fetch_error true", &FeedRequest{FetchError: "true"}, "", "feeds.#.onestop_id", []string{"test"}, 0},
		{"fetch_error false", &FeedRequest{FetchError: "false"}, "", "feeds.#.onestop_id", []string{"BA", "CT"}, 0},
		// {"lat,lon,r:100,limit:100", "feeds", &FeedRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 100.0}, "", 2, bartosid},
		// {"lat,lon,r:1000,limit:100", "feeds", &FeedRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 1000.0}, "", 5, bartosid},
	}
	for _, tc := range testcases {
		testquery(t, cfg, tc)
	}
}

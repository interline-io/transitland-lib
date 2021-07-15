package rest

import (
	"testing"
)

func TestFeedVersionRequest(t *testing.T) {
	cfg := testRestConfig()
	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	testcases := []testRest{
		{"basic", &FeedVersionRequest{}, "", "feed_versions.#.sha1", []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6"}, 0},
		{"limit:1", &FeedVersionRequest{Limit: 1}, "", "feed_versions.#.sha1", []string{fv}, 0},
		{"sha1", &FeedVersionRequest{FeedVersionKey: fv}, "", "feed_versions.#.sha1", []string{fv}, 0},
		{"feed_onestop_id,limit:100", &FeedVersionRequest{Limit: 100, FeedOnestopID: "BA"}, "", "feed_versions.#.sha1", []string{fv}, 0},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, cfg, tc)
		})
	}
}

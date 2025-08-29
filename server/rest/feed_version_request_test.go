package rest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func TestFeedVersionRequest(t *testing.T) {
	fv := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	testcases := []testCase{
		{
			name:         "basic",
			h:            FeedVersionRequest{},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6", "c969427f56d3a645195dd8365cde6d7feae7e99b", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "43e2278aa272879c79460582152b04e7487f0493", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "limit:1",
			h:            FeedVersionRequest{WithCursor: WithCursor{Limit: 1}},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{},
			expectLength: 1,
		},
		{
			name:         "sha1",
			h:            FeedVersionRequest{FeedVersionKey: fv},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{fv},
		},
		{
			name:         "feed_onestop_id,limit:100",
			h:            FeedVersionRequest{WithCursor: WithCursor{Limit: 100}, FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "fetched_after",
			h:            FeedVersionRequest{FetchedAfter: "2009-08-07T06:05:04.3Z", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907", "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "fetched_after 2",
			h:            FeedVersionRequest{FetchedAfter: "2123-04-05T06:07:08.9Z", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{},
		},
		{
			name:         "fetched_before",
			h:            FeedVersionRequest{FetchedBefore: "2123-04-05T06:07:08.9Z", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907", "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "fetched_before 2",
			h:            FeedVersionRequest{FetchedBefore: "2009-08-07T06:05:04.3Z", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{},
		},
		{
			name:         "covers_start_date",
			h:            FeedVersionRequest{CoversStartDate: "2016-12-31", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			name:         "covers_start_date 2",
			h:            FeedVersionRequest{CoversStartDate: "2012-01-01", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{},
		},
		{
			name:         "covers_end_date",
			h:            FeedVersionRequest{CoversEndDate: "2016-12-31", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			name:         "covers_end_date 2",
			h:            FeedVersionRequest{CoversEndDate: "2040-01-01", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{},
		},
		{
			name:         "covers_start_date and covers_end_date",
			h:            FeedVersionRequest{CoversStartDate: "2016-12-01", CoversEndDate: "2016-12-31", FeedOnestopID: "BA"},
			format:       "",
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		// spatial
		{
			name:         "lat,lon,radius 100m",
			h:            FeedVersionRequest{Lon: -122.407974, Lat: 37.784471, Radius: 100},
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "lat,lon,radius 2000m",
			h:            FeedVersionRequest{Lon: -122.407974, Lat: 37.784471, Radius: 2000},
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "bbox",
			h:            FeedVersionRequest{Bbox: &restBbox{model.BoundingBox{MinLon: -122.2698781543005, MinLat: 37.80700393130445, MaxLon: -122.2677640139239, MaxLat: 37.8088734037938}}},
			selector:     "feed_versions.#.sha1",
			expectSelect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

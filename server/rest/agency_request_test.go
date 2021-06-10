package rest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/config"
	"github.com/interline-io/transitland-lib/server/resolvers"
)

func TestAgencyRequest(t *testing.T) {
	cfg := restConfig{srv: resolvers.NewServer(config.Config{})}
	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	testcases := []testRest{
		{"basic", AgencyRequest{}, "", "agencies.#.agency_id", []string{"caltrain-ca-us", "BART"}, 0},
		{"limit:1", AgencyRequest{Limit: 1}, "", "agencies.#.agency_id", []string{"caltrain-ca-us"}, 0},
		{"feed_version_sha1", AgencyRequest{FeedVersionSHA1: fv}, "", "agencies.#.agency_id", []string{"BART"}, 0},
		{"feed_onestop_id", AgencyRequest{FeedOnestopID: "BA"}, "", "agencies.#.agency_id", []string{"BART"}, 0},
		{"feed_onestop_id,agency_id", AgencyRequest{FeedOnestopID: "BA", AgencyID: "BART"}, "", "agencies.#.agency_id", []string{"BART"}, 0},
		{"agency_id", AgencyRequest{AgencyID: "BART"}, "", "agencies.#.agency_id", []string{"BART"}, 0},
		{"agency_name", AgencyRequest{AgencyName: "Bay Area Rapid Transit"}, "", "agencies.#.agency_name", []string{"Bay Area Rapid Transit"}, 0},
		{"onestop_id", AgencyRequest{OnestopID: "o-9q9-bayarearapidtransit"}, "", "agencies.#.onestop_id", []string{"o-9q9-bayarearapidtransit"}, 0},
		{"onestop_id,feed_version_sha1", AgencyRequest{OnestopID: "o-9q9-bayarearapidtransit", FeedVersionSHA1: fv}, "", "agencies.#.feed_version.sha1", []string{fv}, 0},
		{"lat,lon,radius 10m", AgencyRequest{Lat: -122.407974, Lon: 37.784471, Radius: 10}, "", "agencies.#.agency_id", []string{"BART"}, 0},
		{"lat,lon,radius 2000m", AgencyRequest{Lat: -122.407974, Lon: 37.784471, Radius: 2000}, "", "agencies.#.agency_id", []string{"caltrain-ca-us", "BART"}, 0},
		{"search", AgencyRequest{Search: "caltrain"}, "", "agencies.#.agency_id", []string{"caltrain-ca-us"}, 0},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, cfg, tc)
		})
	}
}

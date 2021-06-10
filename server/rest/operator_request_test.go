package rest

import (
	"testing"
)

func TestOperatorRequest(t *testing.T) {
	cfg := testRestConfig()
	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	testcases := []testRest{
		{"basic", OperatorRequest{}, "", "operators.#.onestop_id", []string{"o-9q9-caltrain", "o-9q9-bayarearapidtransit"}, 0},
		{"feed_version_sha1", OperatorRequest{FeedVersionSHA1: fv}, "", "operators.#.onestop_id", []string{"o-9q9-bayarearapidtransit"}, 0},
		{"feed_onestop_id", OperatorRequest{FeedOnestopID: "BA"}, "", "operators.#.onestop_id", []string{"o-9q9-bayarearapidtransit"}, 0},
		{"onestop_id", OperatorRequest{OnestopID: "o-9q9-bayarearapidtransit"}, "", "operators.#.onestop_id", []string{"o-9q9-bayarearapidtransit"}, 0},
		{"onestop_id,feed_version_sha1", OperatorRequest{OnestopID: "o-9q9-bayarearapidtransit", FeedVersionSHA1: fv}, "", "operators.#.onestop_id", []string{"o-9q9-bayarearapidtransit"}, 0},
		{"search", OperatorRequest{Search: "bay area"}, "", "operators.#.onestop_id", []string{"o-9q9-bayarearapidtransit"}, 0},
		// {"lat,lon,radius 10m", OperatorRequest{Lat: -122.407974, Lon: 37.784471, Radius: 10}, "", "operators.#.onestop_id", []string{"BART"}, 0},
		// {"lat,lon,radius 2000m", OperatorRequest{Lat: -122.407974, Lon: 37.784471, Radius: 2000}, "", "operators.#.onestop_id", []string{"caltrain-ca-us", "BART"}, 0},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, cfg, tc)
		})
	}
}

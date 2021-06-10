package rest

import (
	"testing"
)

func TestRouteRequest(t *testing.T) {
	cfg := testRestConfig()
	routeIds := []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130", "01", "03", "05", "07", "11", "19"}
	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	testcases := []testRest{
		{"none", RouteRequest{}, "", "routes.#.route_id", routeIds, 0},
		{"limit:1", RouteRequest{Limit: 1}, "", "routes.#.route_id", nil, 1},
		{"limit:100", RouteRequest{Limit: 100}, "", "routes.#.route_id", nil, 12},
		{"search", RouteRequest{Search: "bullet"}, "", "routes.#.route_id", []string{"Bu-130"}, 0},
		{"feed_onestop_id", RouteRequest{FeedOnestopID: "CT"}, "", "routes.#.route_id", []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"}, 0},
		{"route_type:2", RouteRequest{RouteType: "2"}, "", "routes.#.route_id", []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130"}, 0},
		{"route_type:1", RouteRequest{RouteType: "1"}, "", "routes.#.route_id", []string{"01", "03", "05", "07", "11", "19"}, 0},
		{"feed_onestop_id,route_id", RouteRequest{FeedOnestopID: "BA", RouteID: "19"}, "", "routes.#.route_id", []string{"19"}, 0},
		{"feed_version_sha1", RouteRequest{FeedVersionSHA1: fv}, "", "routes.#.feed_version.sha1", []string{fv, fv, fv, fv, fv, fv}, 0},
		{"operator_onestop_id", RouteRequest{OperatorOnestopID: "o-9q9-bayarearapidtransit"}, "", "routes.#.route_id", []string{"01", "03", "05", "07", "11", "19"}, 0},
		{"lat,lon,radius 100m", RouteRequest{Lat: -122.407974, Lon: 37.784471, Radius: 100}, "", "routes.#.route_id", []string{"01", "05", "07", "11"}, 0},
		{"lat,lon,radius 2000m", RouteRequest{Lat: -122.407974, Lon: 37.784471, Radius: 2000}, "", "routes.#.route_id", []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "01", "05", "07", "11"}, 0},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, cfg, tc)
		})
	}
}

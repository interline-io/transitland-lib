package rest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/resolvers"
	"github.com/tidwall/gjson"
)

func TestTripRequest(t *testing.T) {
	cfg := restConfig{srv: resolvers.NewServer()}
	d, err := makeGraphQLRequest(cfg.srv, `query{routes(where:{feed_onestop_id:"BA",route_id:"11"}) {id}}`, nil)
	if err != nil {
		t.Error("failed to get route id for tests")
	}
	routeId := int(gjson.Get(toJson(d), "routes.0.id").Int())

	d2, err := makeGraphQLRequest(cfg.srv, `query{trips(where:{trip_id:"5132248WKDY"}){id}}`, nil)
	if err != nil {
		t.Error("failed to get route id for tests")
	}
	tripId := int(gjson.Get(toJson(d2), "trips.0.id").Int())

	fv := "e535eb2b3b9ac3ef15d82c56575e914575e732e0"
	ctfv := "d2813c293bcfd7a97dde599527ae6c62c98e66c6"
	testcases := []testRest{
		{"none", TripRequest{}, "", "trips.#.trip_id", nil, 20},
		{"limit:1", TripRequest{Limit: 1}, "", "trips.#.trip_id", nil, 1},
		{"limit:100", TripRequest{Limit: 100}, "", "trips.#.trip_id", nil, 100},
		{"limit:1000", TripRequest{Limit: 1000}, "", "trips.#.trip_id", nil, 1000},
		{"limit:10000", TripRequest{Limit: 10000}, "", "trips.#.trip_id", nil, 1000},
		{"feed_onestop_id", TripRequest{FeedOnestopID: "BA"}, "", "trips.#.trip_id", nil, 20},
		{"feed_onestop_id ct", TripRequest{FeedOnestopID: "CT", Limit: 1000}, "", "trips.#.trip_id", nil, 185},
		{"feed_version_sha1", TripRequest{FeedVersionSHA1: ctfv, Limit: 1000}, "", "trips.#.trip_id", nil, 185},
		{"feed_version_sha1 ba", TripRequest{FeedVersionSHA1: fv, Limit: 1000}, "", "trips.#.trip_id", nil, 1000}, // over 1000
		{"trip_id", TripRequest{TripID: "5132248WKDY"}, "", "trips.#.trip_id", []string{"5132248WKDY"}, 0},
		{"trip_id,feed_version_id", TripRequest{TripID: "5132248WKDY", FeedVersionSHA1: fv}, "", "trips.#.trip_id", []string{"5132248WKDY"}, 0},
		{"route_id", TripRequest{Limit: 1000, RouteID: routeId}, "", "trips.#.trip_id", nil, 364},
		{"route_id,service_date 1", TripRequest{Limit: 1000, RouteID: routeId, ServiceDate: "2018-01-01"}, "", "trips.#.trip_id", nil, 0},
		{"route_id,service_date 2", TripRequest{Limit: 1000, RouteID: routeId, ServiceDate: "2019-01-01"}, "", "trips.#.trip_id", nil, 100},
		{"route_id,service_date 3", TripRequest{Limit: 1000, RouteID: routeId, ServiceDate: "2019-01-02"}, "", "trips.#.trip_id", nil, 152},
		{"route_id,service_date 4", TripRequest{Limit: 1000, RouteID: routeId, ServiceDate: "2020-05-18"}, "", "trips.#.trip_id", nil, 0},
		{"route_id,trip_id", TripRequest{Limit: 1000, RouteID: routeId, TripID: "5132248WKDY"}, "", "trips.#.trip_id", []string{"5132248WKDY"}, 0},
		{"include_geometry=true", TripRequest{TripID: "5132248WKDY", IncludeGeometry: "true"}, "", "trips.0.shape.geometry.type", []string{"LineString"}, 0},
		{"include_geometry=false", TripRequest{TripID: "5132248WKDY", IncludeGeometry: "false"}, "", "trips.0.shape.geometry.type", []string{}, 0},
		{"does not include stop_times without id", TripRequest{TripID: "5132248WKDY"}, "", "trips.0.stop_times.#.stop_sequence", nil, 0},
		{"id includes stop_times", TripRequest{ID: tripId}, "", "trips.0.stop_times.#.stop_sequence", nil, 18},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, cfg, tc)
		})
	}
}

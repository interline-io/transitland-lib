package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
)

func TestRouteResolver(t *testing.T) {
	vars := hw{"route_id": "03"}
	testcases := []testcase{
		{
			"basic",
			`query {  routes { route_id } }`,
			hw{},
			``,
			"routes.#.route_id",
			[]string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130", "01", "03", "05", "07", "11", "19"},
		},
		{
			"basic fields",
			`query($route_id: String!) {  routes(where:{route_id:$route_id}) {onestop_id route_id route_short_name route_long_name route_type route_color route_text_color route_sort_order route_url route_desc feed_version_sha1 feed_onestop_id} }`,
			vars,
			`{"routes":[{"feed_onestop_id":"BA","feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","onestop_id":"r-9q9n-warmsprings~southfremont~richmond","route_color":"ff9933","route_desc":"","route_id":"03","route_long_name":"Warm Springs/South Fremont - Richmond","route_short_name":"","route_sort_order":0,"route_text_color":"","route_type":1,"route_url":"http://www.bart.gov/schedules/bylineresults?route=3"}]}`,
			"",
			nil,
		},
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			"geometry",
			`query($route_id: String!) {  routes(where:{route_id:$route_id}) {geometry} }`,
			vars,
			``,
			"routes.0.geometry.type",
			[]string{"LineString"},
		},
		{
			"feed_version",
			`query($route_id: String!) {  routes(where:{route_id:$route_id}) {feed_version{sha1}} }`,
			vars,
			`{"routes":[{"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}`,
			"",
			nil,
		},
		{
			"trips",
			`query($route_id: String!) {  routes(where:{route_id:$route_id}) {trips{trip_id trip_headsign}} }`,
			hw{"route_id": "Bu-130"}, // use baby bullet
			``,
			"routes.0.trips.#.trip_id",
			[]string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386", "801", "803", "802", "804"},
		},
		{
			"route_stops",
			`query($route_id: String!) {  routes(where:{route_id:$route_id}) {route_stops{stop{stop_id stop_name}}} }`,
			vars,
			``,
			"routes.0.route_stops.#.stop.stop_id",
			[]string{"12TH", "19TH", "19TH_N", "ASHB", "BAYF", "COLS", "DBRK", "DELN", "PLZA", "FRMT", "FTVL", "HAYW", "LAKE", "MCAR", "MCAR_S", "NBRK", "RICH", "SANL", "SHAY", "UCTY", "WARM"},
		},
		{
			// computations are not stable so just check success
			"geometries",
			`query($route_id: String!) {  routes(where:{route_id:$route_id}) {geometries {direction_id}} }`,
			vars,
			``,
			"routes.0.geometries.#.direction_id",
			[]string{"0"},
		},
		{
			// only check dow_category explicitly it's not a stable computation
			"headways",
			`query($route_id: String!) {  routes(where:{route_id:$route_id}) {headways{dow_category direction_id headway_secs service_date service_seconds stop_trip_count headway_seconds_morning_mid stop{stop_id}}} }`,
			vars,
			``,
			"routes.0.headways.#.dow_category",
			[]string{"1", "6", "7"},
		},

		// TODO: census_geographies
		// TODO: route_stop_buffer
	}
	c := client.New(newServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}

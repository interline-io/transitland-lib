package gql

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestRouteResolver(t *testing.T) {
	vars := hw{"route_id": "03"}
	testcases := []testcase{
		{
			name:         "basic",
			query:        `query {  routes { route_id } }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"1", "12", "14", "15", "16", "17", "19", "20", "24", "25", "275", "30", "31", "32", "33", "34", "35", "36", "360", "37", "38", "39", "400", "42", "45", "46", "48", "5", "51", "6", "60", "7", "75", "8", "9", "96", "97", "570", "571", "572", "573", "574", "800", "PWT", "SKY", "01", "03", "05", "07", "11", "19", "Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130", "2bc6804f-9e24-4b91-8947-c73a2363e7b6", "68456f6e-2a04-4fcb-971b-fd57348e2ed7", "3dce5414-260d-4cdb-b3d8-b256802d35c5", "0553af3e-53b8-4f98-ba47-0fc03d2404de", "fb93d53e-bf9a-426b-adb2-c913e4d5ecfd", "424421e5-c7c4-4307-8893-5ab9c913cecf"},
		},
		{
			name:   "basic fields",
			query:  `query($route_id: String!) {  routes(where:{route_id:$route_id}) {onestop_id route_id route_short_name route_long_name route_type route_color route_text_color route_sort_order route_url route_desc feed_version_sha1 feed_onestop_id} }`,
			vars:   vars,
			expect: `{"routes":[{"feed_onestop_id":"BA","feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","onestop_id":"r-9q9n-warmsprings~southfremont~richmond","route_color":"ff9933","route_desc":null,"route_id":"03","route_long_name":"Warm Springs/South Fremont - Richmond","route_short_name":null,"route_sort_order":null,"route_text_color":null,"route_type":1,"route_url":"http://www.bart.gov/schedules/bylineresults?route=3"}]}`,
		},
		{
			name:         "geometry",
			query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {geometry} }`,
			vars:         vars,
			selector:     "routes.0.geometry.type",
			selectExpect: []string{"MultiLineString"},
		},
		{
			name:   "feed_version",
			query:  `query($route_id: String!) {  routes(where:{route_id:$route_id}) {feed_version{sha1}} }`,
			vars:   vars,
			expect: `{"routes":[{"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}`,
		},
		{
			name:         "trips",
			query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {trips{trip_id trip_headsign}} }`,
			vars:         hw{"route_id": "Bu-130"}, // use baby bullet
			selector:     "routes.0.trips.#.trip_id",
			selectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386", "801", "803", "802", "804"},
		},
		{
			name:         "route_stops",
			query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {route_stops{stop{stop_id stop_name}}} }`,
			vars:         vars,
			selector:     "routes.0.route_stops.#.stop.stop_id",
			selectExpect: []string{"12TH", "19TH", "19TH_N", "ASHB", "BAYF", "COLS", "DBRK", "DELN", "PLZA", "FRMT", "FTVL", "HAYW", "LAKE", "MCAR", "MCAR_S", "NBRK", "RICH", "SANL", "SHAY", "UCTY", "WARM"},
		},
		{
			name:         "stops",
			query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {stops{stop_id stop_name}} }`,
			vars:         vars,
			selector:     "routes.0.stops.#.stop_id",
			selectExpect: []string{"12TH", "19TH", "19TH_N", "ASHB", "BAYF", "COLS", "DBRK", "DELN", "PLZA", "FRMT", "FTVL", "HAYW", "LAKE", "MCAR", "MCAR_S", "NBRK", "RICH", "SANL", "SHAY", "UCTY", "WARM"},
		},
		{
			// computations are not stable so just check success
			name:         "geometries",
			query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {geometries {generated}} }`,
			vars:         vars,
			selector:     "routes.0.geometries.#.generated",
			selectExpect: []string{"false"},
		},
		{
			name:         "route_stop_buffer stop_points 10m",
			query:        `query($route_id: String!) { routes(where:{route_id:$route_id}) {route_stop_buffer(radius: 100.0) {stop_points	stop_buffer	stop_convexhull}}}`,
			vars:         vars,
			selector:     "routes.0.route_stop_buffer.stop_points.type",
			selectExpect: []string{"MultiPoint"},
		},
		{
			name:         "route_stop_buffer stop_buffer 10m",
			query:        `query($route_id: String!) { routes(where:{route_id:$route_id}) {route_stop_buffer(radius: 100.0) {stop_points	stop_buffer	stop_convexhull}}}`,
			vars:         vars,
			selector:     "routes.0.route_stop_buffer.stop_buffer.type",
			selectExpect: []string{"MultiPolygon"},
		},
		{
			name:         "route_stop_buffer stop_convexhull 10m",
			query:        `query($route_id: String!) { routes(where:{route_id:$route_id}) {route_stop_buffer(radius: 100.0) {stop_points	stop_buffer	stop_convexhull}}}`,
			vars:         vars,
			selector:     "routes.0.route_stop_buffer.stop_convexhull.type",
			selectExpect: []string{"Polygon"},
		},
		{
			// only check dow_category explicitly it's not a stable computation
			name:         "headways",
			query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {headways{dow_category departures service_date stop_trip_count stop{stop_id}}} }`,
			vars:         vars,
			selector:     "routes.0.headways.#.dow_category",
			selectExpect: []string{"1", "6", "7", "1", "6", "7"}, // now includes one for each direction and dow category
		},
		{
			name:         "where onestop_id",
			query:        `query {routes(where:{onestop_id:"r-9q9j-bullet"}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130"},
		},
		{
			name:  "where feed_version_sha1",
			query: `query {routes(where:{feed_version_sha1:"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}) {route_id} }`,

			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			name:         "where feed_onestop_id",
			query:        `query {routes(where:{feed_onestop_id:"CT"}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			name:         "where route_id",
			query:        `query {routes(where:{route_id:"Lo-130"}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Lo-130"},
		},
		{
			name:         "where route_type=2",
			query:        `query {routes(where:{route_type:2}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130"},
		},
		{
			name:         "where route_types=[2]",
			query:        `query {routes(where:{route_types:[2]}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130"},
		},
		{
			name:         "where route_types=[0,2]",
			query:        `query {routes(where:{route_types:[0,2]}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "800"},
		},
		{
			name:         "where route_type=0 route_types=[2]",
			query:        `query {routes(where:{route_type:0, route_types:[2]}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "800"},
		},
		{
			name:         "where search",
			query:        `query {routes(where:{search:"warm"}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"03", "05"},
		},
		{
			name:         "where search 2",
			query:        `query {routes(where:{search:"bullet"}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130"},
		},

		// route patterns
		{
			name: "route patterns",
			query: `{
				routes(where: {feed_onestop_id: "BA", route_id: "03"}) {
				  route_id
				  patterns {
					count
					direction_id
					stop_pattern_id
					trips(limit: 1) {
					  trip_id
					}
				  }
				}
			  }`,

			selector:     "routes.0.patterns.#.count",
			selectExpect: []string{"132", "124", "56", "50", "2"},
		},
		{
			name: "route patterns inactive fv",
			query: `{
				routes(where: {feed_onestop_id: "EX", feed_version_sha1: "43e2278aa272879c79460582152b04e7487f0493", route_id: "AAMV"}) {
				  route_id
				  patterns {
					count
					direction_id
					stop_pattern_id
					trips(limit: 1) {
					  trip_id
					}
				  }
				}
			  }`,

			selector:     "routes.0.patterns.#.count",
			selectExpect: []string{"2", "2"},
		},
		// route attributes
		{
			name: "route attributes",
			query: `{
						routes(where: {feed_onestop_id: "BA", route_id: "01"}) {
						  route_id
						  route_attribute {
							category
							subcategory
							running_way
						  }
						}
					  }`,
			f: func(t *testing.T, jj string) {
				assert.EqualValues(t, 2, gjson.Get(jj, "routes.0.route_attribute.category").Int())
				assert.EqualValues(t, 201, gjson.Get(jj, "routes.0.route_attribute.subcategory").Int())
				assert.EqualValues(t, 1, gjson.Get(jj, "routes.0.route_attribute.running_way").Int())
			},
		},
		// route serviced
		{
			name: "route serviced=true",
			query: `{
				routes(where: {feed_onestop_id: "EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:true}) {
				  route_id
				}
			  }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"AB", "BFC", "STBA", "CITY", "AAMV"},
		},
		{
			name: "route serviced=false",
			query: `{
				routes(where: {feed_onestop_id: "EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:false}) {
				  route_id
				}
			  }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"NOTRIPS"},
		},
		// TODO: census_geographies
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestRouteResolver_Location(t *testing.T) {
	c, cfg := newTestClient(t)

	// Florida coordinates: approximately in the center of Tampa Bay area
	// This should put HA stops (Tampa area) much closer than BA/CT stops (San Francisco Bay area)
	floridaFocus := tlxy.Point{Lat: 27.9506, Lon: -82.4572}

	// San Jose coordinates: approximately in downtown San Jose
	// This should put CT stops (Caltrain San Jose area) much closer than HA stops (Florida)
	sanJoseFocus := tlxy.Point{Lat: 37.3382, Lon: -121.8863}
	var testRouteId int
	if err := cfg.Finder.DBX().
		QueryRowx(`select gtfs_routes.id from gtfs_routes join feed_states using(feed_version_id) join current_feeds cf on cf.id = feed_states.feed_id where cf.onestop_id = $1 and route_id = $2`, "HA", "96").
		Scan(&testRouteId); err != nil {
		t.Errorf("could not get route ID for test: %s", err.Error())
	}

	testcases := []testcase{
		// just ensure geometry queries complete successfully; checking coordinates is a pain and flaky.
		{
			name:         "where near 100m",
			query:        `query {routes(where:{near:{lon:-122.407974,lat:37.784471,radius:100.0}}) {route_id route_long_name}}`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"01", "05", "07", "11"},
		},
		{
			name:         "where near 10000m",
			query:        `query {routes(where:{near:{lon:-122.407974,lat:37.784471,radius:10000.0}}) {route_id route_long_name}}`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "01", "05", "07", "11"},
		},
		{
			name:         "where within polygon",
			query:        `query{routes(where:{within:{type:"Polygon",coordinates:[[[-122.396,37.8],[-122.408,37.79],[-122.393,37.778],[-122.38,37.787],[-122.396,37.8]]]}}){id route_id}}`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"01", "05", "07", "11"},
		},
		{
			name:         "where within polygon big",
			query:        `query{routes(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id route_id}}`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "01", "05", "07", "11"},
		},
		{
			name:         "where bbox 1",
			query:        `query($bbox:BoundingBox) {routes(where:{bbox:$bbox}) {route_id route_long_name}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "routes.#.route_id",
			selectExpect: []string{"01", "03", "07"},
		},
		{
			name:         "where bbox 2",
			query:        `query($bbox:BoundingBox) {routes(where:{bbox:$bbox}) {route_id route_long_name}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "routes.#.route_id",
			selectExpect: []string{},
		},
		{
			name:        "where bbox too large",
			query:       `query($bbox:BoundingBox) {routes(where:{bbox:$bbox}) {route_id route_long_name}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			expectError: true,
			f: func(t *testing.T, jj string) {
			},
		},
		// Focus test cases
		{
			name: "focus basic: Florida focus point returns HA routes first",
			query: `query($lat:Float!, $lon:Float!) {
				routes(limit: 5, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					feed_version { feed { onestop_id } }
				}
			}`,
			vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			selector:     "routes.#.feed_version.feed.onestop_id",
			selectExpect: []string{"HA", "HA", "HA", "HA", "HA"},
		},
		{
			name: "focus basic: San Jose focus point returns West Coast routes first",
			query: `query($lat:Float!, $lon:Float!) {
				routes(limit: 5, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					feed_version { feed { onestop_id } }
				}
			}`,
			vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon},
			selector:     "routes.#.feed_version.feed.onestop_id",
			selectExpect: []string{"CT", "CT", "CT", "CT", "CT"},
		},
		{
			name: "focus with feed filter: HA routes only, ordered by distance",
			query: `query($lat:Float!, $lon:Float!) {
				routes(limit: 10, where: {feed_onestop_id: "HA", location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					geometry
				}
			}`,
			vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			selector:     "routes.#.route_id",
			selectExpect: []string{"20", "51", "8", "400", "96", "97", "12", "9", "19", "30"},
		},
		{
			// Should start after "96" in above test
			name: "focus with pagination",
			query: `query($lat:Float!, $lon:Float!, $after: Int!) {
				routes(after:$after,limit: 10, where: {feed_onestop_id: "HA", location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					geometry
				}
			}`,
			vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon, "after": testRouteId},
			selector:     "routes.#.route_id",
			selectExpect: []string{"97", "12", "9", "19", "30", "60", "7", "24", "25", "275"},
		},
	}
	queryTestcases(t, c, testcases)
}

func TestRouteResolver_Date(t *testing.T) {
	testcases := []testcaseWithClock{
		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				name:         "trips service date",
				query:        `query($route_id: String!, $service_date:Date) {  routes(where:{route_id:$route_id}) {trips(where:{service_date:$service_date}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "service_date": "2018-06-18"}, // use baby bullet
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		{
			whenUtc: "2018-06-19T22:00:00Z",
			testcase: testcase{
				name:         "trips relative date today (tuesday)",
				query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "relative_date": "TODAY"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		{
			whenUtc: "2018-06-17T22:00:00Z",
			testcase: testcase{
				name:         "trips relative date today (sunday)",
				query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "relative_date": "TODAY"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{"801", "803", "802", "804"},
			},
		},
		{
			whenUtc: "2018-06-17T22:00:00Z",
			testcase: testcase{
				name:         "trips relative date next-monday (today is sunday)",
				query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_MONDAY"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		{
			whenUtc: "2018-06-18T22:00:00Z",
			testcase: testcase{
				name:         "trips relative date next-sunday (today is monday)",
				query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_SUNDAY"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{"801", "803", "802", "804"},
			},
		},
		// Window with fallback
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				name:         "trips service_date out of window, service date is monday",
				query:        `query($route_id: String!, $service_date:Date) {  routes(where:{route_id:$route_id}) {trips(where:{service_date:$service_date}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "service_date": "2024-07-22"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{},
			},
		},
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				name:         "trips service_date out of window, service date is monday, use fallback",
				query:        `query($route_id: String!, $service_date:Date) {  routes(where:{route_id:$route_id}) {trips(where:{service_date:$service_date, use_service_window:true}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "service_date": "2024-07-22"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		// Window with relative date and fallback
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				name:         "trips relative date next-monday (today is monday), outside of window",
				query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_SUNDAY"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{},
			},
		},
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				name:         "trips relative date next-sunday (today is monday), outside of window, use fallback",
				query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date, use_service_window: true}) {trip_id trip_headsign}} }`,
				vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_SUNDAY"},
				selector:     "routes.0.trips.#.trip_id",
				selectExpect: []string{"801", "803", "802", "804"},
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestClientWithOpts(t, testconfig.Options{
				WhenUtc: tc.whenUtc,
			})
			queryTestcase(t, c, tc.testcase)
		})
	}
}

func TestRouteResolver_PreviousOnestopID(t *testing.T) {
	testcases := []testcase{
		{
			name:         "default",
			query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { route_id onestop_id }}`,
			vars:         hw{"osid": "r-9q9-antioch~sfia~millbrae", "previous": false},
			selector:     "routes.#.onestop_id",
			selectExpect: []string{"r-9q9-antioch~sfia~millbrae"},
		},
		{
			name:         "old id no result",
			query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { route_id onestop_id }}`,
			vars:         hw{"osid": "r-9q9-pittsburg~baypoint~sfia~millbrae", "previous": false},
			selector:     "routes.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "old id specify fv",
			query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous, feed_version_sha1:"dd7aca4a8e4c90908fd3603c097fabee75fea907"}) { route_id onestop_id }}`,
			vars:         hw{"osid": "r-9q9-pittsburg~baypoint~sfia~millbrae", "previous": false},
			selector:     "routes.#.onestop_id",
			selectExpect: []string{"r-9q9-pittsburg~baypoint~sfia~millbrae"},
		},
		{
			name:         "use previous",
			query:        `query($osid:String!, $previous:Boolean!) { routes(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { route_id onestop_id }}`,
			vars:         hw{"osid": "r-9q9-pittsburg~baypoint~sfia~millbrae", "previous": true},
			selector:     "routes.#.onestop_id",
			selectExpect: []string{"r-9q9-pittsburg~baypoint~sfia~millbrae"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestRouteResolver_Segments(t *testing.T) {
	testcases := []testcase{
		{
			name:         "two segments",
			query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id } }}`,
			vars:         hw{"fsid": "HA", "route_id": "12"},
			selector:     "routes.0.segments.#.way_id",
			selectExpect: []string{"645693994", "90865590"},
		},
		{
			name:         "single segment",
			query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id } }}`,
			vars:         hw{"fsid": "HA", "route_id": "19"},
			selector:     "routes.0.segments.#.way_id",
			selectExpect: []string{"645693994"},
		},
		{
			name:     "geometry",
			query:    `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id geometry } }}`,
			vars:     hw{"fsid": "HA", "route_id": "12"},
			selector: "routes.0.segments.#.geometry",
			selectExpect: []string{
				`{"coordinates":[[-82.458062,27.954493],[-82.458044,27.954442]],"type":"LineString"}`,
				`{"coordinates":[[-82.437785,28.058093],[-82.438163,28.058095],[-82.438219,28.058091],[-82.438277,28.058079],[-82.43833,28.058061],[-82.438375,28.058036],[-82.438415,28.058002],[-82.438449,28.057959],[-82.438476,28.057911],[-82.438493,28.05786],[-82.438501,28.057806],[-82.438504,28.057755],[-82.438504,28.057713],[-82.438504,28.057535]],"type":"LineString"}`,
			},
		},
		{
			name:         "segments to patterns multiple",
			query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id segment_patterns { stop_pattern_id }} }}`,
			vars:         hw{"fsid": "HA", "route_id": "12"},
			selector:     "routes.0.segments.#.segment_patterns.#.stop_pattern_id",
			selectExpect: []string{"[42,40]", "[42]"}, // multiple lookup results look like this
		},
		{
			name:         "segments to patterns single",
			query:        `query($fsid:String!, $route_id:String!) { routes(where:{route_id:$route_id, feed_onestop_id:$fsid}) { route_id segments { id way_id segment_patterns { stop_pattern_id }} }}`,
			vars:         hw{"fsid": "HA", "route_id": "19"},
			selector:     "routes.0.segments.#.segment_patterns.#.stop_pattern_id",
			selectExpect: []string{"[42,40]"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestRouteResolver_Cursor(t *testing.T) {
	c, cfg := newTestClient(t)
	allEnts, err := cfg.Finder.FindRoutes(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.RouteID.Val)
	}
	testcases := []testcase{
		{
			name:         "no cursor",
			query:        "query{routes(limit:10){feed_version{id} id route_id}}",
			selector:     "routes.#.route_id",
			selectExpect: allIds[:10],
		},
		{
			name:         "after 0",
			query:        "query{routes(after: 0, limit:10){feed_version{id} id route_id}}",
			selector:     "routes.#.route_id",
			selectExpect: allIds[:10],
		},
		{
			name:         "after 10th",
			query:        "query($after: Int!){routes(after: $after, limit:10){feed_version{id} id route_id}}",
			vars:         hw{"after": allEnts[10].ID},
			selector:     "routes.#.route_id",
			selectExpect: allIds[11:21],
		},
		{
			name:         "after last",
			query:        "query($after: Int!){routes(after: $after, limit:10){feed_version{id} id route_id}}",
			vars:         hw{"after": allEnts[len(allEnts)-1].ID},
			selector:     "routes.#.route_id",
			selectExpect: []string{},
		},
		{
			name:         "after invalid id returns no results",
			query:        "query($after: Int!){routes(after: $after, limit:10){feed_version{id} id route_id}}",
			vars:         hw{"after": 10_000_000},
			selector:     "routes.#.route_id",
			selectExpect: []string{},
		},
	}
	queryTestcases(t, c, testcases)
}

func TestRouteResolver_License(t *testing.T) {
	q := `
	query ($lic: LicenseFilter) {
		routes(limit: 10000, where: {license: $lic}) {
		  route_id
		  feed_version {
			feed {
			  onestop_id
			  license {
				share_alike_optional
				create_derived_product
				commercial_use_allowed
				redistribution_allowed
			  }
			}
		  }
		}
	  }	  
	`
	testcases := []testcase{
		// license: share_alike_optional
		{
			name:               "license filter: share_alike_optional = yes",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  45,
		},
		{
			name:               "license filter: share_alike_optional = no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  6,
		},
		{
			name:               "license filter: share_alike_optional = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "f-c20-ctran"},
			selectExpectCount:  57,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  45,
		},
		{
			name:               "license filter: create_derived_product = no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  6,
		},
		{
			name:               "license filter: create_derived_product = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "f-c20-ctran"},
			selectExpectCount:  57,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  45,
		},
		{
			name:               "license filter: commercial_use_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  6,
		},
		{
			name:               "license filter: commercial_use_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "f-c20-ctran"},
			selectExpectCount:  57,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  45,
		},
		{
			name:               "license filter: redistribution_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  6,
		},
		{
			name:               "license filter: redistribution_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "f-c20-ctran"},
			selectExpectCount:  57,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  45,
		},
		{
			name:               "license filter: use_without_attribution = no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  6,
		},
		{
			name:               "license filter: use_without_attribution = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "f-c20-ctran"},
			selectExpectCount:  57,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

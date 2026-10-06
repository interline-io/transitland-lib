package gql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"github.com/twpayne/go-polyline"
)

func TestRouteResolver(t *testing.T) {
	vars := hw{"route_id": "03"}
	testcases := []testcase{
		{
			name:         "basic",
			query:        `query {  routes { route_id } }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"1", "12", "14", "15", "16", "17", "19", "20", "24", "25", "275", "30", "31", "32", "33", "34", "35", "36", "360", "37", "38", "39", "400", "42", "45", "46", "48", "5", "51", "6", "60", "7", "75", "8", "9", "96", "97", "570", "571", "572", "573", "574", "800", "PWT", "SKY", "01", "03", "05", "07", "11", "19", "Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130", "2bc6804f-9e24-4b91-8947-c73a2363e7b6", "68456f6e-2a04-4fcb-971b-fd57348e2ed7", "3dce5414-260d-4cdb-b3d8-b256802d35c5", "0553af3e-53b8-4f98-ba47-0fc03d2404de", "fb93d53e-bf9a-426b-adb2-c913e4d5ecfd", "424421e5-c7c4-4307-8893-5ab9c913cecf", "RED", "BLUE", "GREEN", "YELLOW", "ORANGE", "SILVER", "SHUTTLE"},
		},
		{
			name:   "basic fields",
			query:  `query($route_id: String!) {  routes(where:{route_id:$route_id}) {onestop_id route_id route_short_name route_long_name route_type route_color route_text_color route_sort_order route_url route_desc cemv_support feed_version_sha1 feed_onestop_id} }`,
			vars:   vars,
			expect: `{"routes":[{"cemv_support":null,"feed_onestop_id":"BA","feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","onestop_id":"r-9q9n-warmsprings~southfremont~richmond","route_color":"ff9933","route_desc":null,"route_id":"03","route_long_name":"Warm Springs/South Fremont - Richmond","route_short_name":null,"route_sort_order":null,"route_text_color":null,"route_type":1,"route_url":"http://www.bart.gov/schedules/bylineresults?route=3"}]}`,
		},
		{
			name:   "route_type_basic",
			query:  `query($route_id: String!) {  routes(where:{route_id:$route_id}) {route_type route_type_basic} }`,
			vars:   vars,
			expect: `{"routes":[{"route_type":1,"route_type_basic":1}]}`,
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
			// Migration guard: geometry is reconstructed from tl_route_representative_shapes,
			// so pin the structural invariants on a reviewed shape (Caltrain Bu-130). These
			// hold regardless of the source table and outlive the tl_route_geometries drop.
			name:  "geometry invariants (Bu-130)",
			query: `query { routes(where:{route_id:"Bu-130"}) { geometries { generated geometry combined_geometry } } }`,
			f: func(t *testing.T, jj string) {
				g := gjson.Get(jj, "routes.0.geometries.0")
				// combined_geometry is a MultiLineString with one linestring per representative shape.
				assert.Equal(t, "MultiLineString", g.Get("combined_geometry.type").String())
				assert.Len(t, g.Get("combined_geometry.coordinates").Array(), 4, "combined_geometry linestring count")
				// primary geometry is the rank-0 line: a LineString equal to combined_geometry[0].
				assert.Equal(t, "LineString", g.Get("geometry.type").String())
				assert.Equal(t, g.Get("combined_geometry.coordinates.0").Raw, g.Get("geometry.coordinates").Raw, "primary geometry == combined_geometry[0]")
				assert.False(t, g.Get("generated").Bool())
				// shape_id isn't exposed via the resolver, so pin the rank-0 line's point
				// count and first vertex to lock the actual shape it resolves to.
				pts := g.Get("geometry.coordinates").Array()
				assert.Len(t, pts, 382, "primary geometry point count")
				assert.InDelta(t, -121.903170, pts[0].Array()[0].Float(), 1e-5, "first vertex lon")
				assert.InDelta(t, 37.330157, pts[0].Array()[1].Float(), 1e-5, "first vertex lat")
			},
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
			name: "route patterns representative_trip",
			query: `{
				routes(where: {feed_onestop_id: "BA", route_id: "03"}) {
				  patterns {
					count
					representative_trip { trip_id }
					trips(limit: 1) { trip_id }
				  }
				}
			  }`,
			f: func(t *testing.T, jj string) {
				// Walked per pattern rather than as two `#.` arrays: gjson omits the
				// elements whose path is missing, so a null representative_trip would
				// shift one array against the other instead of failing.
				//
				// min(id) in the aggregate is the trip trips(limit:1) returns, which
				// orders by (feed_version_id, id). They agree only where a stop pattern
				// runs in one direction — trips() does not filter on direction, so a
				// pattern operated both ways returns the same trip for both rows.
				pats := gjson.Get(jj, "routes.0.patterns").Array()
				assert.NotEmpty(t, pats, "patterns returned")
				for i, pat := range pats {
					rep := pat.Get("representative_trip.trip_id")
					assert.True(t, rep.Exists(), "pattern %d resolves a representative trip", i)
					assert.Equal(t, pat.Get("trips.0.trip_id").String(), rep.String(), "pattern %d", i)
				}
			},
		},
		{
			name:  "route patterns timetable only the date's trips",
			query: `{ routes(where:{feed_onestop_id:"BA", route_id:"03"}) { patterns(where:{service_date:"2018-05-30"}) { stop_pattern_id count trips(limit:1000) { trip_id } timetable { stop_ids trips { trip_id } departure_times { values delta polyline } arrival_times { values } pickup_types { values delta polyline } drop_off_types { values } timepoints { values delta polyline } } } } }`,
			f: func(t *testing.T, jj string) {
				checkPatternTimetable(t, jj)
				// Pattern 31 runs 132 trips across the weekday, Saturday and Sunday
				// calendars, and 26 on this Wednesday.
				pat := gjson.Get(jj, "routes.0.patterns.#(stop_pattern_id==31)")
				assert.Equal(t, int64(26), pat.Get("trips.#").Int(), "the pattern's trips on the date")
				table := pat.Get("timetable")
				assert.Equal(t, int64(26), table.Get("trips.#").Int())
				assert.Equal(t, "2221650WKDY", table.Get("trips.0.trip_id").String(), "the day's first trip")
				assert.Equal(t, "16:50:00", table.Get("departure_times.values.0.0").String())
				assert.Equal(t, int64(16*3600+50*60), table.Get("departure_times.delta.0.0").Int(), "delta in seconds")
				assert.Equal(t, "18:00:00", table.Get("arrival_times.values.18.0").String())
				assert.Equal(t, int64(1), table.Get("timepoints.values.0.0").Int())
			},
		},
		{
			name:  "route patterns timetable without a date",
			query: `{ routes(where:{feed_onestop_id:"BA", route_id:"03"}) { patterns { service_date timetable { stop_ids } } } }`,
			f: func(t *testing.T, jj string) {
				pats := gjson.Get(jj, "routes.0.patterns").Array()
				assert.NotEmpty(t, pats, "patterns returned")
				for i, pat := range pats {
					assert.Equal(t, gjson.Null, pat.Get("service_date").Type, "pattern %d", i)
					assert.Equal(t, gjson.Null, pat.Get("timetable").Type, "pattern %d", i)
				}
			},
		},
		{
			name:  "route patterns timetable under aliases",
			query: `{ routes(where:{feed_onestop_id:"BA", route_id:"03"}) { weekday: patterns(where:{service_date:"2018-05-30"}) { service_date timetable { trips { trip_id } } } again: patterns(where:{service_date:"2018-05-30"}) { service_date timetable { trips { trip_id } } } sunday: patterns(where:{service_date:"2018-06-03"}) { service_date timetable { trips { trip_id } } } } }`,
			f: func(t *testing.T, jj string) {
				// Each alias gets its own day's trips, and the same day twice is the
				// same answer twice.
				route := gjson.Get(jj, "routes.0")
				assert.Equal(t, route.Get("weekday").Raw, route.Get("again").Raw)
				for alias, day := range map[string]struct{ date, service string }{
					"weekday": {"2018-05-30", "WKDY"},
					"sunday":  {"2018-06-03", "SUN"},
				} {
					pats := route.Get(alias).Array()
					assert.NotEmpty(t, pats, "%s patterns returned", alias)
					for i, pat := range pats {
						assert.Equal(t, day.date, pat.Get("service_date").String(), "%s pattern %d", alias, i)
						trips := pat.Get("timetable.trips.#.trip_id").Array()
						assert.NotEmpty(t, trips, "%s pattern %d has trips", alias, i)
						for _, trip := range trips {
							assert.True(t, strings.HasSuffix(trip.String(), day.service), "%s pattern %d trip %s", alias, i, trip.String())
						}
					}
				}
			},
		},
		{
			name:  "route patterns timetable pickup and drop-off",
			query: `{ routes(where:{feed_onestop_id:"WMATA", route_id:"GREEN"}) { patterns(where:{service_date:"2026-04-29"}) { timetable { trips { trip_id } pickup_types { values } drop_off_types { values } } } } }`,
			f: func(t *testing.T, jj string) {
				// This late trip lets nobody off at the stop it starts from.
				found := false
				for _, pat := range gjson.Get(jj, "routes.0.patterns").Array() {
					for j, trip := range pat.Get("timetable.trips").Array() {
						if trip.Get("trip_id").String() != "11687110_20571" {
							continue
						}
						found = true
						assert.Equal(t, int64(0), pat.Get(fmt.Sprintf("timetable.pickup_types.values.0.%d", j)).Int())
						assert.Equal(t, int64(1), pat.Get(fmt.Sprintf("timetable.drop_off_types.values.0.%d", j)).Int())
					}
				}
				assert.True(t, found, "trip found in a pattern's timetable")
			},
		},
		{
			name:  "route patterns timetable frequency trip",
			query: `{ feed_versions(where:{feed_onestop_id:"EX"}) { routes(where:{route_id:"STBA"}) { patterns(where:{service_date:"2007-01-02"}) { timetable { trips { trip_id frequencies { headway_secs } } departure_times { values } } } } } }`,
			f: func(t *testing.T, jj string) {
				// A trip run from frequencies.txt is one column, its template, and says
				// when it repeats.
				table := gjson.Get(jj, "feed_versions.0.routes.0.patterns.0.timetable")
				assert.Equal(t, int64(1), table.Get("trips.#").Int())
				assert.Equal(t, int64(1800), table.Get("trips.0.frequencies.0.headway_secs").Int())
				assert.Equal(t, "06:20:00", table.Get("departure_times.values.1.0").String(), "the template's second stop")
			},
		},
		{
			name:  "route patterns timetable flex",
			query: `{ routes(where:{feed_onestop_id:"ctran-flex", route_id:"2bc6804f-9e24-4b91-8947-c73a2363e7b6"}) { patterns(where:{service_date:"2026-01-06"}) { timetable { stop_ids } } } }`,
			f: func(t *testing.T, jj string) {
				pats := gjson.Get(jj, "routes.0.patterns").Array()
				assert.NotEmpty(t, pats, "patterns returned")
				for i, pat := range pats {
					assert.Equal(t, gjson.Null, pat.Get("timetable").Type, "pattern %d", i)
				}
			},
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
		// serves_stop_onestop_id
		{
			name: "serves_stop_onestop_id",
			query: `{
				routes(where: {serves_stop_onestop_id: "s-9q8yyufxmv-sanfranciscocaltrain"}) {
					route_id
				}
			}`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "Gi-130", "Li-130", "Lo-130", "Sp-130"},
		},
		{
			name: "serves_stop_onestop_id:none",
			query: `{
				routes(where: {serves_stop_onestop_id: "s-invalid-stop"}) {
					route_id
				}
			}`,
			selector:     "routes.#.route_id",
			selectExpect: []string{},
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

// As for stops: both ways of matching an onestop_id must agree.
func TestRouteResolver_OnestopID_Materialized(t *testing.T) {
	testcases := []testcase{
		{
			name:         "where onestop_id",
			query:        `query {routes(where:{onestop_id:"r-9q9j-bullet"}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130"},
		},
		{
			name:         "where onestop_ids",
			query:        `query {routes(where:{onestop_ids:["r-9q9j-bullet","r-9q9n-warmsprings~southfremont~richmond"]}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "03"},
		},
		{
			name:         "selected onestop_id",
			query:        `query {routes(where:{onestop_id:"r-9q9j-bullet"}) {onestop_id} }`,
			selector:     "routes.#.onestop_id",
			selectExpect: []string{"r-9q9j-bullet"},
		},
		{
			name:         "unknown onestop_id",
			query:        `query {routes(where:{onestop_id:"r-0000-nowhere"}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{},
		},
		// As for stops. This route's onestop_id changed between versions.
		{
			name:         "previous allowed, current id",
			query:        `query {routes(where:{onestop_id:"r-9q9-antioch~sfia~millbrae", allow_previous_onestop_ids:true}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"01"},
		},
		{
			name:  "previous allowed, previous id",
			query: `query {routes(where:{onestop_id:"r-9q9-pittsburg~baypoint~sfia~millbrae", allow_previous_onestop_ids:true}) {route_id onestop_id} }`,
			sel: []testcaseSelector{
				{selector: "routes.#.route_id", expect: []string{"01"}},
				{selector: "routes.#.onestop_id", expect: []string{"r-9q9-antioch~sfia~millbrae"}},
			},
		},
		{
			name:         "previous allowed, current and previous ids",
			query:        `query {routes(where:{onestop_ids:["r-9q9j-bullet","r-9q9-pittsburg~baypoint~sfia~millbrae"], allow_previous_onestop_ids:true}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130", "01"},
		},
		{
			name:         "previous allowed, unknown id",
			query:        `query {routes(where:{onestop_id:"r-0000-nowhere", allow_previous_onestop_ids:true}) {route_id} }`,
			selector:     "routes.#.route_id",
			selectExpect: []string{},
		},
	}
	for _, materialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseMaterialized=%t", materialized), func(t *testing.T) {
			c, _ := newTestClientWithOpts(t, testconfig.Options{UseMaterialized: materialized})
			queryTestcases(t, c, testcases)
		})
	}
}

// As for stops, with the operator filter the Explore route page sends. Here the
// old BART version gives route 01 the onestop ID of Caltrain's Baby Bullet.
func TestRouteResolver_OnestopID_PreviousFiltered(t *testing.T) {
	bullet := "r-9q9j-bullet"
	testcases := []testcase{
		{
			name:         "current match",
			query:        `query($osid:String!){routes(where:{onestop_id:$osid, allow_previous_onestop_ids:true}) {route_id} }`,
			vars:         hw{"osid": bullet},
			selector:     "routes.#.route_id",
			selectExpect: []string{"Bu-130"},
		},
		{
			name:         "current match filtered out",
			query:        `query($osid:String!){routes(where:{onestop_id:$osid, allow_previous_onestop_ids:true, operator_onestop_id:"o-9q9-bayarearapidtransit"}) {route_id} }`,
			vars:         hw{"osid": bullet},
			selector:     "routes.#.route_id",
			selectExpect: []string{"01"},
		},
	}
	for _, materialized := range []bool{false, true} {
		t.Run(fmt.Sprintf("UseMaterialized=%t", materialized), func(t *testing.T) {
			testconfig.ConfigTxRollback(t, testconfig.Options{UseMaterialized: materialized}, func(cfg model.Config) {
				ctx := context.Background()
				sha1 := "dd7aca4a8e4c90908fd3603c097fabee75fea907"
				fvs, err := cfg.Finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &sha1})
				if err != nil || len(fvs) != 1 {
					t.Fatalf("feed version %s: %v", sha1, err)
				}
				previous := stats.FeedVersionStats{RouteOnestopIDs: []dmfr.FeedVersionRouteOnestopID{{EntityID: "01", OnestopID: bullet}}}
				if err := stats.WriteFeedVersionStats(ctx, cfg.Adapter, previous, fvs[0].ID, stats.WriteOptions{Stats: []string{stats.StatOnestopIDs}}); err != nil {
					t.Fatal(err)
				}
				srv := model.AddConfigAndPerms(cfg, NewDefaultHandler())
				queryTestcases(t, client.New(usercheck.UserDefaultMiddleware("test")(srv)), testcases)
			})
		})
	}
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
	if err := cfg.Adapter.DBX().
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
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				name:  "patterns service date",
				query: `query($route_id: String!, $service_date:Date) { routes(where:{route_id:$route_id}) { patterns(where:{service_date:$service_date}) { count representative_trip { trip_id } } } }`,
				vars:  hw{"route_id": "Bu-130", "service_date": "2018-06-18"},
				f: func(t *testing.T, jj string) {
					// The same 22 trips the trips() query returns for this date,
					// redistributed across the patterns that run it. Walked per pattern
					// so a null representative_trip fails rather than being skipped.
					pats := gjson.Get(jj, "routes.0.patterns").Array()
					assert.NotEmpty(t, pats, "patterns returned for the date")
					total := 0
					for i, pat := range pats {
						total += int(pat.Get("count").Int())
						assert.True(t, pat.Get("representative_trip.trip_id").Exists(), "pattern %d resolves a representative trip", i)
					}
					assert.Equal(t, 22, total, "day-scoped counts sum to that day's trips")
				},
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
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				name:  "patterns timetable relative date next-sunday, outside of window, use fallback",
				query: `{ routes(where:{route_id:"Bu-130"}) { patterns(where:{relative_date:NEXT_SUNDAY, use_service_window:true}) { service_date timetable { trips { trip_id } } } } }`,
				f: func(t *testing.T, jj string) {
					// The same trips the trips() query falls back to, on the date it fell back to.
					var tripIDs []string
					for _, pat := range gjson.Get(jj, "routes.0.patterns").Array() {
						assert.Equal(t, "2018-06-24", pat.Get("service_date").String(), "the Sunday the window falls back to")
						for _, trip := range pat.Get("timetable.trips").Array() {
							tripIDs = append(tripIDs, trip.Get("trip_id").String())
						}
					}
					assert.ElementsMatch(t, []string{"801", "803", "802", "804"}, tripIDs)
				},
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
			selectExpect: []string{"r-9q9-antioch~sfia~millbrae"},
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
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  52,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  64,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  52,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  64,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  52,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  64,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  52,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  64,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "routes.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  52,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  64,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestRoutePatternGridEncodings(t *testing.T) {
	// Each trip has a gap, the last trip's at the first stop, and three values,
	// so a difference from the first value can't pass for one from the previous.
	cell, gap := tt.NewInt, tt.Int{}
	grid := &model.RouteStopPatternTimetableGrid{Values: [][]tt.Int{
		{cell(100), cell(400), gap},
		{cell(160), gap, cell(700)},
		{gap, cell(520), cell(790)},
		{cell(250), cell(610), cell(880)},
	}}
	r := &routePatternGridResolver{}
	delta, err := r.Delta(context.Background(), grid)
	if assert.NoError(t, err) {
		assert.Equal(t, [][]*int{{ptr(100), ptr(400), nil}, {ptr(60), nil, ptr(700)}, {nil, ptr(120), ptr(90)}, {ptr(150), ptr(210), ptr(180)}}, delta)
	}
	lines, err := r.Polyline(context.Background(), grid)
	if assert.NoError(t, err) {
		// The wire format itself, since the encoder and the decoder below share a library.
		assert.Equal(t, []string{"gEwQ.", "wB._g@", ".oFz@", "kHwBz@"}, lines)
		var rows [][]*int64
		for _, line := range lines {
			rows = append(rows, decodePolylineRow(t, line))
		}
		want := [][]*int64{
			{ptr(int64(100)), ptr(int64(400)), nil},
			{ptr(int64(160)), nil, ptr(int64(700))},
			{nil, ptr(int64(520)), ptr(int64(790))},
			{ptr(int64(250)), ptr(int64(610)), ptr(int64(880))},
		}
		assert.Equal(t, want, undoDelta(rows))
	}

	// The same numbers as times encode the same way, in seconds.
	times := &model.RouteStopPatternTimetableTimeGrid{}
	for _, row := range grid.Values {
		var cells []tt.Seconds
		for _, v := range row {
			cells = append(cells, tt.Seconds{Option: v.Option})
		}
		times.Values = append(times.Values, cells)
	}
	tr := &routePatternTimeGridResolver{}
	timeDelta, err := tr.Delta(context.Background(), times)
	if assert.NoError(t, err) {
		assert.Equal(t, delta, timeDelta)
	}
	timeLines, err := tr.Polyline(context.Background(), times)
	if assert.NoError(t, err) {
		assert.Equal(t, lines, timeLines)
	}
}

// checkPatternTimetable checks each timetable's grids are stops by trips and
// decode back from their encodings, and its trips run in order of first time.
func checkPatternTimetable(t *testing.T, jj string) {
	pats := gjson.Get(jj, "routes.0.patterns").Array()
	assert.NotEmpty(t, pats, "patterns returned")
	for i, pat := range pats {
		count := int(pat.Get("count").Int())
		table := pat.Get("timetable")
		stops := len(table.Get("stop_ids").Array())
		assert.NotZero(t, stops, "pattern %d has stops", i)
		assert.Len(t, table.Get("trips").Array(), count, "pattern %d: a column per trip", i)
		for _, name := range []string{"departure_times", "arrival_times", "pickup_types", "drop_off_types", "timepoints"} {
			grid := table.Get(name)
			if !grid.Exists() {
				continue
			}
			rows := grid.Get("values").Array()
			assert.Len(t, rows, stops, "pattern %d %s: a row per stop", i, name)
			for k, row := range rows {
				assert.Len(t, row.Array(), count, "pattern %d %s stop %d: a cell per trip", i, name, k)
			}
			checkGridEncodings(t, grid, fmt.Sprintf("pattern %d %s", i, name))
		}
		prev := int64(-1)
		for _, v := range gridRow(t, table.Get("departure_times.values.0")) {
			if v == nil {
				continue
			}
			assert.LessOrEqual(t, prev, *v, "pattern %d: trips by time at the first stop", i)
			prev = *v
		}
	}
}

// checkGridEncodings checks a grid's delta and polyline, where asked for, decode
// back to its values.
func checkGridEncodings(t *testing.T, grid gjson.Result, label string) {
	var values [][]*int64
	for _, row := range grid.Get("values").Array() {
		values = append(values, gridRow(t, row))
	}
	if delta := grid.Get("delta"); delta.Exists() {
		var rows [][]*int64
		for _, row := range delta.Array() {
			rows = append(rows, gridRow(t, row))
		}
		assert.Equal(t, values, undoDelta(rows), "%s delta", label)
	}
	if lines := grid.Get("polyline"); lines.Exists() {
		var rows [][]*int64
		for _, line := range lines.Array() {
			rows = append(rows, decodePolylineRow(t, line.String()))
		}
		assert.Equal(t, values, undoDelta(rows), "%s polyline", label)
	}
}

// gridRow reads one row of a grid, times as seconds, nil where null.
func gridRow(t *testing.T, row gjson.Result) []*int64 {
	cells := row.Array()
	ret := make([]*int64, 0, len(cells))
	for _, cell := range cells {
		switch cell.Type {
		case gjson.Null:
			ret = append(ret, nil)
		case gjson.String:
			v, err := tt.StringToSeconds(cell.String())
			assert.NoError(t, err)
			ret = append(ret, &v)
		default:
			v := cell.Int()
			ret = append(ret, &v)
		}
	}
	return ret
}

// undoDelta adds each column's first non-null value back to the later ones.
func undoDelta(rows [][]*int64) [][]*int64 {
	ret := make([][]*int64, len(rows))
	for i, row := range rows {
		ret[i] = make([]*int64, len(row))
	}
	if len(rows) == 0 {
		return ret
	}
	for j := range rows[0] {
		var first *int64
		for i, row := range rows {
			if row[j] == nil {
				continue
			}
			v := *row[j]
			if first == nil {
				first = &v
			} else {
				v += *first
			}
			ret[i][j] = &v
		}
	}
	return ret
}

// decodePolylineRow reads one row of a grid's polyline, nil where '.'.
func decodePolylineRow(t *testing.T, s string) []*int64 {
	ret := make([]*int64, 0, len(s))
	buf, prev := []byte(s), int64(0)
	for len(buf) > 0 {
		if buf[0] == '.' {
			ret = append(ret, nil)
			buf = buf[1:]
			continue
		}
		d, rest, err := polyline.DecodeInt(buf)
		if !assert.NoError(t, err) {
			return ret
		}
		prev += int64(d)
		v := prev
		ret = append(ret, &v)
		buf = rest
	}
	return ret
}

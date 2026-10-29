package gqltest

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"github.com/twpayne/go-polyline"
)

func routeResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase {
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
			Name:               "license filter: share_alike_optional = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  52,
		},
		{
			Name:               "license filter: share_alike_optional = no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  6,
		},
		{
			Name:               "license filter: share_alike_optional = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  64,
		},
		// license: create_derived_product
		{
			Name:               "license filter: create_derived_product = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  52,
		},
		{
			Name:               "license filter: create_derived_product = no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  6,
		},
		{
			Name:               "license filter: create_derived_product = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  64,
		},
		// license: commercial_use_allowed
		{
			Name:               "license filter: commercial_use_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  52,
		},
		{
			Name:               "license filter: commercial_use_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  6,
		},
		{
			Name:               "license filter: commercial_use_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  64,
		},
		// license: redistribution_allowed
		{
			Name:               "license filter: redistribution_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  52,
		},
		{
			Name:               "license filter: redistribution_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  6,
		},
		{
			Name:               "license filter: redistribution_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  64,
		},
		// license: use_without_attribution
		{
			Name:               "license filter: use_without_attribution = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  52,
		},
		{
			Name:               "license filter: use_without_attribution = no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  6,
		},
		{
			Name:               "license filter: use_without_attribution = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			Selector:           "routes.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  64,
		},
	}
	return testcases
}

func routeResolverDateTestcases() []testcaseWithClock {
	testcases := []testcaseWithClock{
		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				Name:         "trips service date",
				Query:        `query($route_id: String!, $service_date:Date) {  routes(where:{route_id:$route_id}) {trips(where:{service_date:$service_date}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "service_date": "2018-06-18"}, // use baby bullet
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		{
			whenUtc: "2018-05-30T22:00:00Z",
			testcase: testcase{
				Name:  "patterns service date",
				Query: `query($route_id: String!, $service_date:Date) { routes(where:{route_id:$route_id}) { patterns(where:{service_date:$service_date}) { count representative_trip { trip_id } } } }`,
				Vars:  hw{"route_id": "Bu-130", "service_date": "2018-06-18"},
				F: func(t *testing.T, jj string) {
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
				Name:         "trips relative date today (tuesday)",
				Query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "relative_date": "TODAY"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		{
			whenUtc: "2018-06-17T22:00:00Z",
			testcase: testcase{
				Name:         "trips relative date today (sunday)",
				Query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "relative_date": "TODAY"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{"801", "803", "802", "804"},
			},
		},
		{
			whenUtc: "2018-06-17T22:00:00Z",
			testcase: testcase{
				Name:         "trips relative date next-monday (today is sunday)",
				Query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_MONDAY"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		{
			whenUtc: "2018-06-18T22:00:00Z",
			testcase: testcase{
				Name:         "trips relative date next-sunday (today is monday)",
				Query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_SUNDAY"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{"801", "803", "802", "804"},
			},
		},
		// Window with fallback
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				Name:         "trips service_date out of window, service date is monday",
				Query:        `query($route_id: String!, $service_date:Date) {  routes(where:{route_id:$route_id}) {trips(where:{service_date:$service_date}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "service_date": "2024-07-22"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{},
			},
		},
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				Name:         "trips service_date out of window, service date is monday, use fallback",
				Query:        `query($route_id: String!, $service_date:Date) {  routes(where:{route_id:$route_id}) {trips(where:{service_date:$service_date, use_service_window:true}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "service_date": "2024-07-22"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386"},
			},
		},
		// Window with relative date and fallback
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				Name:         "trips relative date next-monday (today is monday), outside of window",
				Query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_SUNDAY"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{},
			},
		},
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				Name:         "trips relative date next-sunday (today is monday), outside of window, use fallback",
				Query:        `query($route_id: String!, $relative_date:RelativeDate) {  routes(where:{route_id:$route_id}) {trips(where:{relative_date:$relative_date, use_service_window: true}) {trip_id trip_headsign}} }`,
				Vars:         hw{"route_id": "Bu-130", "relative_date": "NEXT_SUNDAY"},
				Selector:     "routes.0.trips.#.trip_id",
				SelectExpect: []string{"801", "803", "802", "804"},
			},
		},
		{
			whenUtc: "2024-07-22T22:00:00Z",
			testcase: testcase{
				Name:  "patterns timetable relative date next-sunday, outside of window, use fallback",
				Query: `{ routes(where:{route_id:"Bu-130"}) { patterns(where:{relative_date:NEXT_SUNDAY, use_service_window:true}) { service_date timetable { trips { trip_id } } } } }`,
				F: func(t *testing.T, jj string) {
					// The same trips the trips() query falls back to, reported on the
					// requested Sunday.
					var tripIDs []string
					for _, pat := range gjson.Get(jj, "routes.0.patterns").Array() {
						assert.Equal(t, "2024-07-28", pat.Get("service_date").String(), "the requested Sunday")
						for _, trip := range pat.Get("timetable.trips").Array() {
							tripIDs = append(tripIDs, trip.Get("trip_id").String())
						}
					}
					assert.ElementsMatch(t, []string{"801", "803", "802", "804"}, tripIDs)
				},
			},
		},
	}
	return testcases
}

func routeResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	vars := hw{"route_id": "03"}
	testcases := []testcase{
		{
			Name:         "basic",
			Query:        `query {  routes { route_id } }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"1", "12", "14", "15", "16", "17", "19", "20", "24", "25", "275", "30", "31", "32", "33", "34", "35", "36", "360", "37", "38", "39", "400", "42", "45", "46", "48", "5", "51", "6", "60", "7", "75", "8", "9", "96", "97", "570", "571", "572", "573", "574", "800", "PWT", "SKY", "01", "03", "05", "07", "11", "19", "Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130", "2bc6804f-9e24-4b91-8947-c73a2363e7b6", "68456f6e-2a04-4fcb-971b-fd57348e2ed7", "3dce5414-260d-4cdb-b3d8-b256802d35c5", "0553af3e-53b8-4f98-ba47-0fc03d2404de", "fb93d53e-bf9a-426b-adb2-c913e4d5ecfd", "424421e5-c7c4-4307-8893-5ab9c913cecf", "RED", "BLUE", "GREEN", "YELLOW", "ORANGE", "SILVER", "SHUTTLE"},
		},
		{
			Name:   "basic fields",
			Query:  `query($route_id: String!) {  routes(where:{route_id:$route_id}) {onestop_id route_id route_short_name route_long_name route_type route_color route_text_color route_sort_order route_url route_desc cemv_support feed_version_sha1 feed_onestop_id} }`,
			Vars:   vars,
			Expect: `{"routes":[{"cemv_support":null,"feed_onestop_id":"BA","feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","onestop_id":"r-9q9n-warmsprings~southfremont~richmond","route_color":"ff9933","route_desc":null,"route_id":"03","route_long_name":"Warm Springs/South Fremont - Richmond","route_short_name":null,"route_sort_order":null,"route_text_color":null,"route_type":1,"route_url":"http://www.bart.gov/schedules/bylineresults?route=3"}]}`,
		},
		{
			Name:   "route_type_basic",
			Query:  `query($route_id: String!) {  routes(where:{route_id:$route_id}) {route_type route_type_basic} }`,
			Vars:   vars,
			Expect: `{"routes":[{"route_type":1,"route_type_basic":1}]}`,
		},
		{
			Name:         "geometry",
			Query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {geometry} }`,
			Vars:         vars,
			Selector:     "routes.0.geometry.type",
			SelectExpect: []string{"MultiLineString"},
		},
		{
			Name:   "feed_version",
			Query:  `query($route_id: String!) {  routes(where:{route_id:$route_id}) {feed_version{sha1}} }`,
			Vars:   vars,
			Expect: `{"routes":[{"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}]}`,
		},
		{
			Name:         "trips",
			Query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {trips{trip_id trip_headsign}} }`,
			Vars:         hw{"route_id": "Bu-130"}, // use baby bullet
			Selector:     "routes.0.trips.#.trip_id",
			SelectExpect: []string{"305", "309", "313", "319", "323", "329", "365", "371", "375", "381", "385", "310", "314", "320", "324", "330", "360", "366", "370", "376", "380", "386", "801", "803", "802", "804"},
		},
		{
			Name:         "route_stops",
			Query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {route_stops{stop{stop_id stop_name}}} }`,
			Vars:         vars,
			Selector:     "routes.0.route_stops.#.stop.stop_id",
			SelectExpect: []string{"12TH", "19TH", "19TH_N", "ASHB", "BAYF", "COLS", "DBRK", "DELN", "PLZA", "FRMT", "FTVL", "HAYW", "LAKE", "MCAR", "MCAR_S", "NBRK", "RICH", "SANL", "SHAY", "UCTY", "WARM"},
		},
		{
			Name:         "stops",
			Query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {stops{stop_id stop_name}} }`,
			Vars:         vars,
			Selector:     "routes.0.stops.#.stop_id",
			SelectExpect: []string{"12TH", "19TH", "19TH_N", "ASHB", "BAYF", "COLS", "DBRK", "DELN", "PLZA", "FRMT", "FTVL", "HAYW", "LAKE", "MCAR", "MCAR_S", "NBRK", "RICH", "SANL", "SHAY", "UCTY", "WARM"},
		},
		{
			// computations are not stable so just check success
			Name:         "geometries",
			Query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {geometries {generated}} }`,
			Vars:         vars,
			Selector:     "routes.0.geometries.#.generated",
			SelectExpect: []string{"false"},
		},
		{
			// Migration guard: geometry is reconstructed from tl_route_representative_shapes,
			// so pin the structural invariants on a reviewed shape (Caltrain Bu-130). These
			// hold regardless of the source table and outlive the tl_route_geometries drop.
			Name:  "geometry invariants (Bu-130)",
			Query: `query { routes(where:{route_id:"Bu-130"}) { geometries { generated geometry combined_geometry } } }`,
			F: func(t *testing.T, jj string) {
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
			// Each representative shape on its own, and the summary geometry: its parts are
			// the shapes' lines in rank order, and its length is the longest shape's.
			Name:  "representative shapes (Bu-130)",
			Query: `query { routes(where:{route_id:"Bu-130"}) { representative_shapes { rank direction_id generated length shape { shape_id geometry } } geometries { length combined_geometry } } }`,
			F: func(t *testing.T, jj string) {
				r := gjson.Get(jj, "routes.0")
				shapes := r.Get("representative_shapes").Array()
				var ids []string
				var ranks, dirs []int64
				maxLength := 0.0
				for _, s := range shapes {
					ids = append(ids, s.Get("shape.shape_id").String())
					ranks = append(ranks, s.Get("rank").Int())
					dirs = append(dirs, s.Get("direction_id").Int())
					assert.False(t, s.Get("generated").Bool())
					maxLength = max(maxLength, s.Get("length").Float())
				}
				assert.Equal(t, []string{"cal_sj_sf", "cal_sf_sj", "cal_sf_tam", "cal_tam_sf"}, ids)
				assert.Equal(t, []int64{0, 1, 2, 3}, ranks)
				assert.Equal(t, []int64{0, 1, 1, 0}, dirs)
				parts := r.Get("geometries.0.combined_geometry.coordinates").Array()
				if assert.Len(t, parts, len(shapes), "one combined part per representative shape") {
					for i, s := range shapes {
						assert.Len(t, parts[i].Array(), len(s.Get("shape.geometry.coordinates").Array()), "combined part %d is shape %d", i, i)
					}
				}
				assert.InDelta(t, maxLength, r.Get("geometries.0.length").Float(), 1e-6, "summary length is the longest shape's")
				assert.Greater(t, maxLength, shapes[0].Get("length").Float(), "the longest shape is not the rank 0 one")
			},
		},
		{
			Name:         "representative shapes limit",
			Query:        `query { routes(where:{route_id:"Bu-130"}) { representative_shapes(limit: 1) { rank shape { shape_id } } } }`,
			Selector:     "routes.0.representative_shapes.#.shape.shape_id",
			SelectExpect: []string{"cal_sj_sf"},
		},
		{
			// only check dow_category explicitly it's not a stable computation
			Name:         "headways",
			Query:        `query($route_id: String!) {  routes(where:{route_id:$route_id}) {headways{dow_category departures service_date stop_trip_count stop{stop_id}}} }`,
			Vars:         vars,
			Selector:     "routes.0.headways.#.dow_category",
			SelectExpect: []string{"1", "6", "7", "1", "6", "7"}, // now includes one for each direction and dow category
		},
		{
			Name:         "where onestop_id",
			Query:        `query {routes(where:{onestop_id:"r-9q9j-bullet"}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130"},
		},
		{
			Name:  "where feed_version_sha1",
			Query: `query {routes(where:{feed_version_sha1:"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}) {route_id} }`,

			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			Name:         "where feed_onestop_id",
			Query:        `query {routes(where:{feed_onestop_id:"CT"}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			Name:         "where route_id",
			Query:        `query {routes(where:{route_id:"Lo-130"}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Lo-130"},
		},
		{
			Name:         "where route_type=2",
			Query:        `query {routes(where:{route_type:2}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130"},
		},
		{
			Name:         "where route_types=[2]",
			Query:        `query {routes(where:{route_types:[2]}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130"},
		},
		{
			Name:         "where route_types=[0,2]",
			Query:        `query {routes(where:{route_types:[0,2]}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "800"},
		},
		{
			Name:         "where route_type=0 route_types=[2]",
			Query:        `query {routes(where:{route_type:0, route_types:[2]}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "800"},
		},
		{
			Name:         "where search",
			Query:        `query {routes(where:{search:"warm"}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"03", "05"},
		},
		{
			Name:         "where search 2",
			Query:        `query {routes(where:{search:"bullet"}) {route_id} }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130"},
		},
		// route patterns
		{
			Name: "route patterns",
			Query: `{
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

			Selector:     "routes.0.patterns.#.count",
			SelectExpect: []string{"132", "124", "56", "50", "2"},
		},
		{
			Name: "route patterns representative_trip",
			Query: `{
				routes(where: {feed_onestop_id: "BA", route_id: "03"}) {
				  patterns {
					count
					representative_trip { trip_id }
					trips(limit: 1) { trip_id }
				  }
				}
			  }`,
			F: func(t *testing.T, jj string) {
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
			Name:  "route patterns timetable only the date's trips",
			Query: `{ routes(where:{feed_onestop_id:"BA", route_id:"03"}) { patterns(where:{service_date:"2018-05-30"}) { stop_pattern_id count trips(limit:1000) { trip_id } timetable { stop_ids trips { trip_id } departure_times { values delta polyline } arrival_times { values } pickup_types { values delta polyline } drop_off_types { values } timepoints { values delta polyline } } } } }`,
			F: func(t *testing.T, jj string) {
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
			Name:  "route patterns timetable shape distances",
			Query: `{ routes(where:{feed_onestop_id:"BA", route_id:"03"}) { patterns(where:{service_date:"2018-05-30"}) { stop_pattern_id count timetable { stop_ids trips { trip_id shape { distances } stop_times { shape_distance } } shape_distances { values delta polyline } } } } }`,
			F: func(t *testing.T, jj string) {
				checkPatternTimetable(t, jj)
				for i, pat := range gjson.Get(jj, "routes.0.patterns").Array() {
					table := pat.Get("timetable")
					rows := table.Get("shape_distances.values").Array()
					for j, trip := range table.Get("trips").Array() {
						// A trip's own stop times say the same as its column.
						stopTimes := trip.Get("stop_times.#.shape_distance").Array()
						assert.Len(t, stopTimes, len(rows), "pattern %d trip %d", i, j)
						for k, st := range stopTimes {
							assert.Equal(t, rows[k].Array()[j].Raw, st.Raw, "pattern %d trip %d stop %d", i, j, k)
						}
						dists := trip.Get("shape.distances").Array()
						assert.NotEmpty(t, dists, "pattern %d trip %d has a measured shape", i, j)
						end := dists[len(dists)-1].Float()
						// Each stop is at or after the one before, and on the shape.
						prev := 0.0
						for k, row := range rows {
							cell := row.Array()[j]
							assert.Equal(t, gjson.Number, cell.Type, "pattern %d trip %d stop %d", i, j, k)
							assert.GreaterOrEqual(t, cell.Float(), prev, "pattern %d trip %d stop %d", i, j, k)
							assert.LessOrEqual(t, cell.Float(), end+1, "pattern %d trip %d stop %d", i, j, k)
							prev = cell.Float()
						}
						assert.Greater(t, prev, 0.0, "pattern %d trip %d goes somewhere", i, j)
					}
				}
			},
		},
		{
			Name:  "route patterns timetable without a date",
			Query: `{ routes(where:{feed_onestop_id:"BA", route_id:"03"}) { patterns { service_date timetable { stop_ids } } } }`,
			F: func(t *testing.T, jj string) {
				pats := gjson.Get(jj, "routes.0.patterns").Array()
				assert.NotEmpty(t, pats, "patterns returned")
				for i, pat := range pats {
					assert.Equal(t, gjson.Null, pat.Get("service_date").Type, "pattern %d", i)
					assert.Equal(t, gjson.Null, pat.Get("timetable").Type, "pattern %d", i)
				}
			},
		},
		{
			Name:  "route patterns timetable under aliases",
			Query: `{ routes(where:{feed_onestop_id:"BA", route_id:"03"}) { weekday: patterns(where:{service_date:"2018-05-30"}) { service_date timetable { trips { trip_id } } } again: patterns(where:{service_date:"2018-05-30"}) { service_date timetable { trips { trip_id } } } sunday: patterns(where:{service_date:"2018-06-03"}) { service_date timetable { trips { trip_id } } } } }`,
			F: func(t *testing.T, jj string) {
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
			Name:  "route patterns timetable pickup and drop-off",
			Query: `{ routes(where:{feed_onestop_id:"WMATA", route_id:"GREEN"}) { patterns(where:{service_date:"2026-04-29"}) { timetable { trips { trip_id } pickup_types { values } drop_off_types { values } } } } }`,
			F: func(t *testing.T, jj string) {
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
			Name:  "route patterns timetable frequency trip",
			Query: `{ feed_versions(where:{feed_onestop_id:"EX"}) { routes(where:{route_id:"STBA"}) { patterns(where:{service_date:"2007-01-02"}) { timetable { trips { trip_id frequencies { headway_secs } } departure_times { values } } } } } }`,
			F: func(t *testing.T, jj string) {
				// A trip run from frequencies.txt is one column, its template, and says
				// when it repeats.
				table := gjson.Get(jj, "feed_versions.0.routes.0.patterns.0.timetable")
				assert.Equal(t, int64(1), table.Get("trips.#").Int())
				assert.Equal(t, int64(1800), table.Get("trips.0.frequencies.0.headway_secs").Int())
				assert.Equal(t, "06:20:00", table.Get("departure_times.values.1.0").String(), "the template's second stop")
			},
		},
		{
			Name:  "route patterns timetable flex",
			Query: `{ routes(where:{feed_onestop_id:"ctran-flex", route_id:"2bc6804f-9e24-4b91-8947-c73a2363e7b6"}) { patterns(where:{service_date:"2026-01-06"}) { timetable { stop_ids } } } }`,
			F: func(t *testing.T, jj string) {
				pats := gjson.Get(jj, "routes.0.patterns").Array()
				assert.NotEmpty(t, pats, "patterns returned")
				for i, pat := range pats {
					assert.Equal(t, gjson.Null, pat.Get("timetable").Type, "pattern %d", i)
				}
			},
		},
		{
			Name: "route patterns inactive fv",
			Query: `{
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

			Selector:     "routes.0.patterns.#.count",
			SelectExpect: []string{"2", "2"},
		},
		// serves_stop_onestop_id
		{
			Name: "serves_stop_onestop_id",
			Query: `{
				routes(where: {serves_stop_onestop_id: "s-9q8yyufxmv-sanfranciscocaltrain"}) {
					route_id
				}
			}`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Gi-130", "Li-130", "Lo-130", "Sp-130"},
		},
		{
			Name: "serves_stop_onestop_id:none",
			Query: `{
				routes(where: {serves_stop_onestop_id: "s-invalid-stop"}) {
					route_id
				}
			}`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{},
		},
		// route serviced
		{
			Name: "route serviced=true",
			Query: `{
				routes(where: {feed_onestop_id: "EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:true}) {
				  route_id
				}
			  }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"AB", "BFC", "STBA", "CITY", "AAMV"},
		},
		{
			Name: "route serviced=false",
			Query: `{
				routes(where: {feed_onestop_id: "EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:false}) {
				  route_id
				}
			  }`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"NOTRIPS"},
		},
	}
	return testcases
}

func routeResolverCursorTestcases(t testing.TB, cfg model.Config) []testcase {
	allEnts, err := cfg.Finder.FindRoutes(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 21, "routes", allEnts)
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.RouteID.Val)
	}
	testcases := []testcase{
		{
			Name:         "no cursor",
			Query:        "query{routes(limit:10){feed_version{id} id route_id}}",
			Selector:     "routes.#.route_id",
			SelectExpect: allIds[:10],
		},
		{
			Name:         "after 0",
			Query:        "query{routes(after: 0, limit:10){feed_version{id} id route_id}}",
			Selector:     "routes.#.route_id",
			SelectExpect: allIds[:10],
		},
		{
			Name:         "after 10th",
			Query:        "query($after: Int!){routes(after: $after, limit:10){feed_version{id} id route_id}}",
			Vars:         hw{"after": allEnts[10].ID},
			Selector:     "routes.#.route_id",
			SelectExpect: allIds[11:21],
		},
		{
			Name:         "after last",
			Query:        "query($after: Int!){routes(after: $after, limit:10){feed_version{id} id route_id}}",
			Vars:         hw{"after": allEnts[len(allEnts)-1].ID},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{},
		},
		{
			Name:         "after invalid id returns no results",
			Query:        "query($after: Int!){routes(after: $after, limit:10){feed_version{id} id route_id}}",
			Vars:         hw{"after": 10_000_000},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func routeResolverLocationTestcases(t testing.TB, cfg model.Config) []testcase {

	// Florida coordinates: approximately in the center of Tampa Bay area
	// This should put HA stops (Tampa area) much closer than BA/CT stops (San Francisco Bay area)
	floridaFocus := tlxy.Point{Lat: 27.9506, Lon: -82.4572}

	// San Jose coordinates: approximately in downtown San Jose
	// This should put CT stops (Caltrain San Jose area) much closer than HA stops (Florida)
	sanJoseFocus := tlxy.Point{Lat: 37.3382, Lon: -121.8863}
	testFeed, testRouteCode := "HA", "96"
	testRoutes, err := cfg.Finder.FindRoutes(model.WithConfig(context.Background(), cfg), nil, nil, nil, &model.RouteFilter{FeedOnestopID: &testFeed, RouteID: &testRouteCode})
	if err != nil {
		t.Fatal(err)
	}
	if len(testRoutes) != 1 {
		t.Fatalf("found %d active routes with route_id %s in %s, expected 1", len(testRoutes), testRouteCode, testFeed)
	}
	testRouteId := testRoutes[0].ID

	testcases := []testcase{
		// just ensure geometry queries complete successfully; checking coordinates is a pain and flaky.
		{
			Name:         "where near 100m",
			Query:        `query {routes(where:{near:{lon:-122.407974,lat:37.784471,radius:100.0}}) {route_id route_long_name}}`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"01", "05", "07", "11"},
		},
		{
			Name:         "where near 10000m",
			Query:        `query {routes(where:{near:{lon:-122.407974,lat:37.784471,radius:10000.0}}) {route_id route_long_name}}`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "01", "05", "07", "11"},
		},
		{
			Name:         "where within polygon",
			Query:        `query{routes(where:{within:{type:"Polygon",coordinates:[[[-122.396,37.8],[-122.408,37.79],[-122.393,37.778],[-122.38,37.787],[-122.396,37.8]]]}}){id route_id}}`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"01", "05", "07", "11"},
		},
		{
			Name:         "where within polygon big",
			Query:        `query{routes(where:{within:{type:"Polygon",coordinates:[[[-122.39481925964355,37.80151060070086],[-122.41653442382812,37.78652126637423],[-122.39662170410156,37.76847577247014],[-122.37301826477051,37.784757615348575],[-122.39481925964355,37.80151060070086]]]}}){id route_id}}`,
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "Gi-130", "Sp-130", "01", "05", "07", "11"},
		},
		{
			Name:         "where bbox 1",
			Query:        `query($bbox:BoundingBox) {routes(where:{bbox:$bbox}) {route_id route_long_name}}`,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"01", "03", "07"},
		},
		{
			Name:         "where bbox 2",
			Query:        `query($bbox:BoundingBox) {routes(where:{bbox:$bbox}) {route_id route_long_name}}`,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{},
		},
		{
			Name:        "where bbox too large",
			Query:       `query($bbox:BoundingBox) {routes(where:{bbox:$bbox}) {route_id route_long_name}}`,
			Vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			ExpectError: true,
		},
		// Focus test cases
		{
			Name: "focus basic: Florida focus point returns HA routes first",
			Query: `query($lat:Float!, $lon:Float!) {
				routes(limit: 5, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					feed_version { feed { onestop_id } }
				}
			}`,
			Vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			Selector:     "routes.#.feed_version.feed.onestop_id",
			SelectExpect: []string{"HA", "HA", "HA", "HA", "HA"},
		},
		{
			Name: "focus basic: San Jose focus point returns West Coast routes first",
			Query: `query($lat:Float!, $lon:Float!) {
				routes(limit: 5, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					feed_version { feed { onestop_id } }
				}
			}`,
			Vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon},
			Selector:     "routes.#.feed_version.feed.onestop_id",
			SelectExpect: []string{"CT", "CT", "CT", "CT", "CT"},
		},
		{
			Name: "focus with feed filter: HA routes only, ordered by distance",
			Query: `query($lat:Float!, $lon:Float!) {
				routes(limit: 10, where: {feed_onestop_id: "HA", location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					geometry
				}
			}`,
			Vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"20", "51", "8", "400", "96", "97", "12", "9", "19", "30"},
		},
		{
			// Should start after "96" in above test
			Name: "focus with pagination",
			Query: `query($lat:Float!, $lon:Float!, $after: Int!) {
				routes(after:$after,limit: 10, where: {feed_onestop_id: "HA", location: {focus: {lat: $lat, lon: $lon}}}) {
					route_id
					geometry
				}
			}`,
			Vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon, "after": testRouteId},
			Selector:     "routes.#.route_id",
			SelectExpect: []string{"97", "12", "9", "19", "30", "60", "7", "24", "25", "275"},
		},
	}
	return testcases
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
		for _, name := range []string{"departure_times", "arrival_times", "pickup_types", "drop_off_types", "timepoints", "shape_distances"} {
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

// routeResolverTripsAcrossFeedVersionsTest is the trip counterpart of
// stopResolverServiceWindowIsPerFeedVersionTest.
//
// A `routes` selection's trips can span feed versions, and each feed version
// resolves the requested date against its own service window. Sharing the
// resolved filter, or the rows, from one feed version to the next loses
// results. Which results it loses depends on the order in which the finder
// visits the feed versions, so the same request answers differently from one
// call to the next.
//
// 2018-05-29 falls inside BA's service window; CT has none covering it and
// relocates the date into its fallback week.
func routeResolverTripsAcrossFeedVersionsTest(t *testing.T, c *client.Client, cfg model.Config) {
	query := func(q string) string {
		var resp map[string]interface{}
		if err := c.Post(q, &resp); err != nil {
			t.Fatal(err)
		}
		return gqlcase.ToJSON(resp)
	}
	// Resolved rather than hardcoded: these are unstable database ids.
	routeID := func(feed string, gtfsRouteID string) string {
		j := query(`query{routes(where:{feed_onestop_id:"` + feed + `", route_id:"` + gtfsRouteID + `"}){id}}`)
		id := gjson.Get(j, "routes.0.id")
		if !id.Exists() {
			t.Fatalf("fixture route %s:%s not found", feed, gtfsRouteID)
		}
		return id.String()
	}
	inWindow := routeID("BA", "01")
	relocates := routeID("CT", "Bu-130")

	// The trips of one route out of a request covering all the given routes.
	trips := func(want string, ids ...string) string {
		j := query(`query{routes(ids:[` + strings.Join(ids, ",") + `]){
			id trips(limit:5, where:{service_date:"` + serviceWindowDate + `", use_service_window:true}){ trip_id }
		}}`)
		for _, r := range gjson.Get(j, "routes").Array() {
			if r.Get("id").String() == want {
				return r.Get("trips.#.trip_id").Raw
			}
		}
		t.Fatalf("route %s missing from response %s", want, j)
		return ""
	}

	// The answer when nothing else shares the request.
	alone := trips(inWindow, inWindow)
	assert.NotEqual(t, "[]", alone, "fixture route should run on the requested date")

	// The premise: the other route's feed version really does resolve the date
	// differently. Without this the test can keep passing while asserting
	// nothing, if the fixtures' service windows ever drift.
	assert.NotEqual(t, "[]", trips(relocates, relocates),
		"fixture should include a second feed version that returns trips for this date")

	// Asking for both must not change either one's answer, and must not change
	// it differently from one call to the next.
	for i := 0; i < 10; i++ {
		if !assert.JSONEq(t, alone, trips(inWindow, inWindow, relocates), "run %d differs from the single-route query", i) {
			t.FailNow()
		}
	}
}

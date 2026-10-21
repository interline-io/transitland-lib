package findertest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Cases duplicated from server/gql/route_resolver_test.go.

func routeResolverDateTestcases() []testcaseWithClock {
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

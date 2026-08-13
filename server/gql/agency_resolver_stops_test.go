package gql

import (
	"testing"
)

// The WMATA feed is the only fixture with a station hierarchy: 125 served
// platforms hanging off 98 stations, plus entrances and generic nodes that no
// route serves.
func TestAgencyResolver_Stops(t *testing.T) {
	q := `query($osid:String!, $where:StopFilter, $limit:Int) { agencies(where:{onestop_id:$osid}) { stops(limit:$limit, where:$where) { stop_id location_type } } }`
	wmata := hw{"osid": "o-dqcj-wmata", "limit": 1000}
	testcases := []testcase{
		{
			name:              "served platforms and their stations",
			query:             q,
			vars:              wmata,
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 223,
		},
		{
			name:              "location_type 1 returns stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 98,
		},
		{
			name:              "location_type 0 returns platforms",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 0}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 125,
		},
		{
			// Entrances and nodes belong to the same stations but are not served
			// and are not reached from a served stop, so they stay out.
			name:         "location_type 2 returns no entrances",
			query:        q,
			vars:         hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 2}},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: []string{},
		},
		{
			name:              "limit applies per agency",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 10, "where": hw{"location_type": 1}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 10,
		},
		{
			name:         "agency without stations",
			query:        q,
			vars:         hw{"osid": "o-9q9-caltrain", "limit": 1000, "where": hw{"location_type": 1}},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: []string{},
		},
		{
			name:               "scoped to the agency, not the feed version",
			query:              q,
			vars:               hw{"osid": "o-9q9-caltrain", "limit": 1000},
			selector:           "agencies.0.stops.#.location_type",
			selectExpectUnique: []string{"0"},
			selectExpectCount:  64,
		},
		{
			name:         "other StopFilter options apply",
			query:        q,
			vars:         hw{"osid": "o-9q9-caltrain", "limit": 1000, "where": hw{"stop_id": "70011"}},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: []string{"70011"},
		},
		{
			// Service filters apply to the served platform, so they select
			// stations too. Applied to the returned row they would match no
			// station at all, since a station is never served directly.
			name:               "served_by_route_types selects the stations of those routes",
			query:              q,
			vars:               hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_types": []int{3}}},
			selector:           "agencies.0.stops.#.location_type",
			selectExpectUnique: []string{"1"},
			selectExpectCount:  10,
		},
		{
			name:              "served_by_route_types narrows to the subway stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_types": []int{1}}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 98,
		},
		{
			name:              "served_by_onestop_ids selects stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_onestop_ids": []string{"o-dqcj-wmata"}}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 98,
		},
		{
			// serviced picks which end of the platform-to-station walk to keep.
			name:               "serviced false returns the stations",
			query:              q,
			vars:               hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"serviced": false}},
			selector:           "agencies.0.stops.#.location_type",
			selectExpectUnique: []string{"1"},
			selectExpectCount:  98,
		},
		{
			name:               "serviced true returns the platforms",
			query:              q,
			vars:               hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"serviced": true}},
			selector:           "agencies.0.stops.#.location_type",
			selectExpectUnique: []string{"0"},
			selectExpectCount:  125,
		},
		{
			// The relation walks platform to station and never back down: HART
			// serves 2,349 platforms and has no stations, so nothing is added.
			name:               "unserved siblings are not pulled in",
			query:              q,
			vars:               hw{"osid": "o-dhv-hillsborougharearegionaltransit", "limit": 3000},
			selector:           "agencies.0.stops.#.location_type",
			selectExpectUnique: []string{"0"},
			selectExpectCount:  2349,
		},
		{
			// Operators reach stops through their agencies; there is no
			// Operator.stops, just as there is no Operator.routes.
			name:  "reached through an operator's agencies",
			query: `query { operators(where:{onestop_id:"o-dqcj-wmata"}) { agencies { stations: stops(limit:1000, where:{location_type:1}) { stop_id } platforms: stops(limit:100, where:{location_type:0}) { stop_id } } } }`,
			sel: []testcaseSelector{
				{selector: "operators.0.agencies.0.stations.#.stop_id", expectCount: 98},
				{selector: "operators.0.agencies.0.platforms.#.stop_id", expectCount: 100},
			},
		},
		{
			// Two agencies in one request must not share a limit.
			name:  "batched agencies keep separate limits",
			query: `query { agencies(where:{adm0_iso:"US"}) { onestop_id stops(limit:3) { stop_id } } }`,
			sel: []testcaseSelector{
				{selector: "agencies.#.onestop_id", expect: []string{"o-9q9-bayarearapidtransit", "o-9q9-caltrain", "o-dhv-hillsborougharearegionaltransit", "o-dqcj-wmata"}},
				{selector: "agencies.0.stops.#.stop_id", expectCount: 3},
				{selector: "agencies.3.stops.#.stop_id", expectCount: 3},
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

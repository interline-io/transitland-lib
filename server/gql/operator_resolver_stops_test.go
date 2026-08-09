package gql

import (
	"testing"
)

func TestOperatorResolver_Stops(t *testing.T) {
	q := `query($osid:String!, $where:StopFilter, $limit:Int) { operators(where:{onestop_id:$osid}) { stops(limit:$limit, where:$where) { stop_id location_type } } }`
	testcases := []testcase{
		{
			name:              "served platforms and their stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000},
			selector:          "operators.0.stops.#.stop_id",
			selectExpectCount: 223,
		},
		{
			name:              "location_type 1 returns stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1}},
			selector:          "operators.0.stops.#.stop_id",
			selectExpectCount: 98,
		},
		{
			// The operator page shape: every station, plus a capped page of platforms.
			name:  "stations plus a page of platforms",
			query: `query { operators(where:{onestop_id:"o-dqcj-wmata"}) { stations: stops(limit:1000, where:{location_type:1}) { stop_id } platforms: stops(limit:100, where:{location_type:0}) { stop_id } } }`,
			sel: []testcaseSelector{
				{selector: "operators.0.stations.#.stop_id", expectCount: 98},
				{selector: "operators.0.platforms.#.stop_id", expectCount: 100},
			},
		},
		{
			name:               "operator without stations",
			query:              q,
			vars:               hw{"osid": "o-9q9-caltrain", "limit": 1000},
			selector:           "operators.0.stops.#.location_type",
			selectExpectUnique: []string{"0"},
			selectExpectCount:  64,
		},
		{
			name:         "other StopFilter options apply",
			query:        q,
			vars:         hw{"osid": "o-9q9-caltrain", "limit": 1000, "where": hw{"stop_id": "70011"}},
			selector:     "operators.0.stops.#.stop_id",
			selectExpect: []string{"70011"},
		},
		{
			name:  "batched operators keep separate limits",
			query: `query { operators(where:{adm0_iso:"US"}) { onestop_id stops(limit:3) { stop_id } } }`,
			sel: []testcaseSelector{
				{selector: "operators.0.stops.#.stop_id", expectCount: 3},
				{selector: "operators.1.stops.#.stop_id", expectCount: 3},
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

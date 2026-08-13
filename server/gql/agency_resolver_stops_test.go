package gql

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

// The WMATA feed is the only fixture with a station hierarchy: 125 served
// platforms hanging off 98 stations, plus entrances and generic nodes that no
// route serves.
func TestAgencyResolver_Stops(t *testing.T) {
	q := `query($osid:String!, $where:AgencyStopFilter, $limit:Int) { agencies(where:{onestop_id:$osid}) { stops(limit:$limit, where:$where) { stop_id location_type } } }`
	wmata := hw{"osid": "o-dqcj-wmata", "limit": 1000}
	testcases := []testcase{
		{
			name:               "default returns the served platforms",
			query:              q,
			vars:               wmata,
			selector:           "agencies.0.stops.#.location_type",
			selectExpectUnique: []string{"0"},
			selectExpectCount:  125,
		},
		{
			name:              "location_type 1 returns stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 98,
		},
		{
			name:              "explicit location_type 0 matches the default",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 0}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 125,
		},
		{
			name:              "location_type null is treated as 0",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": nil}},
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
			name:         "other stop options apply",
			query:        q,
			vars:         hw{"osid": "o-9q9-caltrain", "limit": 1000, "where": hw{"stop_id": "70011"}},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: []string{"70011"},
		},
		{
			// Served-by filters apply to the served platform, so they select
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
			// The subway serves every station, so this matches the unfiltered
			// station count.
			name:              "served_by_route_types narrows to the subway stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_types": []int{1}}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 98,
		},
		{
			name:              "served_by_route_onestop_ids narrows to one line's stations",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_onestop_ids": []string{"r-dqcj-r"}}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 27,
		},
		{
			name:              "served_by_route_onestop_ids takes the union of lines",
			query:             q,
			vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_onestop_ids": []string{"r-dqcj-r", "r-dqcm-g"}}},
			selector:          "agencies.0.stops.#.stop_id",
			selectExpectCount: 46,
		},
		{
			// The filter matches against the agency's own routes; another
			// operator's route can never match.
			name:         "another agency's route matches nothing",
			query:        q,
			vars:         hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"served_by_route_onestop_ids": []string{"r-9q9-antioch~sfia~millbrae"}}},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: []string{},
		},
		{
			// Only route Onestop IDs are accepted; anything else matches nothing
			// rather than being ignored.
			name:         "non-route ids match nothing",
			query:        q,
			vars:         hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"served_by_route_onestop_ids": []string{"o-dqcj-wmata"}}},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: []string{},
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

func TestAgencyResolver_StopsCursor(t *testing.T) {
	c, cfg := newTestClient(t)
	ctx := model.WithConfig(context.Background(), cfg)
	osid := "o-dqcj-wmata"
	agencies, err := cfg.Finder.FindAgencies(ctx, nil, nil, nil, &model.AgencyFilter{OnestopID: &osid})
	if err != nil {
		t.Fatal(err)
	}
	if len(agencies) != 1 {
		t.Fatalf("expected 1 agency, got %d", len(agencies))
	}
	locationType := 1
	allGroups, err := cfg.Finder.StopsByAgencyIDs(ctx, nil, nil, &model.AgencyStopFilter{LocationType: &locationType}, []int{agencies[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	allEnts := allGroups[0]
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.StopID.Val)
	}
	q := `query($osid:String!, $after:Int) { agencies(where:{onestop_id:$osid}) { stops(after:$after, limit:1000, where:{location_type:1}) { stop_id } } }`
	testcases := []testcase{
		{
			name:         "no cursor",
			query:        q,
			vars:         hw{"osid": osid},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: allIds,
		},
		{
			name:         "after 0",
			query:        q,
			vars:         hw{"osid": osid, "after": 0},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: allIds,
		},
		{
			name:         "after 1st",
			query:        q,
			vars:         hw{"osid": osid, "after": allEnts[1].ID},
			selector:     "agencies.0.stops.#.stop_id",
			selectExpect: allIds[2:],
		},
	}
	queryTestcases(t, c, testcases)
}

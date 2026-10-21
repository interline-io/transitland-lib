package findertest

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

// Cases duplicated from server/gql/agency_resolver_test.go.

func agencyResolverLicenseTestcases() []testcase {
	q := `
	query ($lic: LicenseFilter) {
		agencies(limit: 10000, where: {license: $lic}) {
		  agency_id
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
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: share_alike_optional = no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: share_alike_optional = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: create_derived_product = no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: create_derived_product = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: commercial_use_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: commercial_use_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: redistribution_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: redistribution_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  2,
		},
		{
			name:               "license filter: use_without_attribution = no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  1,
		},
		{
			name:               "license filter: use_without_attribution = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:           "agencies.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4,
		},
	}
	return testcases
}

func agencyResolverTestcases() []testcase {
	vars := hw{"agency_id": "caltrain-ca-us"}
	testcases := []testcase{
		{
			name:         "basic",
			query:        `query { agencies {agency_id}}`,
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us", "a8b6ef46-7d4d-45f8-8200-cf4f5ce9d5a6", "BART", "", "1"},
		},
		{
			name:   "basic fields",
			query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {onestop_id agency_id agency_name agency_lang agency_phone agency_timezone agency_url agency_email agency_fare_url cemv_support feed_version_sha1 feed_onestop_id}}`,
			vars:   vars,
			expect: `{"agencies":[{"agency_email":null,"agency_fare_url":null,"agency_id":"caltrain-ca-us","agency_lang":"en","agency_name":"Caltrain","agency_phone":"800-660-4287","agency_timezone":"America/Los_Angeles","agency_url":"http://www.caltrain.com","cemv_support":null,"feed_onestop_id":"CT","feed_version_sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","onestop_id":"o-9q9-caltrain"}]}`,
		},
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			name:         "geometry",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {geometry}}`,
			vars:         vars,
			selector:     "agencies.0.geometry.type",
			selectExpect: []string{"Polygon"},
		},
		{
			name:   "feed_version",
			query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {feed_version { sha1 }}}`,
			vars:   vars,
			expect: `{"agencies":[{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}]}`,
		},
		{
			name:         "routes",
			query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {routes { route_id }}}`,
			vars:         vars,
			selector:     "agencies.0.routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			name:   "route_types",
			query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {route_types}}`,
			vars:   vars,
			expect: `{"agencies":[{"route_types":[2,3]}]}`,
		},
		{
			name:   "route_types extended",
			query:  `query{agencies(where:{feed_onestop_id:"HA"}) {route_types}}`,
			expect: `{"agencies":[{"route_types":[0,3,4]}]}`,
		},
		// Every route type in the fixtures is already a basic one, so this says the
		// field is wired and deduplicates; tt.TestBasicRouteType is what covers the
		// fold itself, over codes no fixture carries.
		{
			name:   "route_types_basic",
			query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {route_types route_types_basic}}`,
			vars:   vars,
			expect: `{"agencies":[{"route_types":[2,3],"route_types_basic":[2,3]}]}`,
		},
		{
			name:   "route_types_basic across a feed",
			query:  `query{agencies(where:{feed_onestop_id:"HA"}) {route_types_basic}}`,
			expect: `{"agencies":[{"route_types_basic":[0,3,4]}]}`,
		},
		{
			name:         "route_types through operator",
			query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {agencies{route_types}}}`,
			selector:     "operators.0.agencies.0.route_types",
			selectExpect: []string{"2", "3"},
		},
		// search
		{
			name:         "search",
			query:        `query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			vars:         hw{"search": "Bay Area"},
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"BART"},
		},
		{
			name:         "search",
			query:        `query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			vars:         hw{"search": "caltrain"},
			selector:     "agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us"},
		},
	}
	return testcases
}

// The WMATA feed is the only fixture with a station hierarchy: 125 served
// platforms hanging off 98 stations, plus entrances and generic nodes that no
// route serves.
func agencyResolverStopsTestcases() []testcase {
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
	}
	return testcases
}

func agencyResolverCursorTestcases(t *testing.T, cfg model.Config) []testcase {
	allEnts, err := cfg.Finder.FindAgencies(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.AgencyID.Val)
	}
	testcases := []testcase{
		{
			name:         "no cursor",
			query:        "query{agencies(limit:10){feed_version{id} id agency_id}}",
			selector:     "agencies.#.agency_id",
			selectExpect: allIds,
		},
		{
			name:         "after 0",
			query:        "query{agencies(after: 0, limit:10){feed_version{id} id agency_id}}",
			selector:     "agencies.#.agency_id",
			selectExpect: allIds,
		},
		{
			name:         "after 1st",
			query:        "query($after: Int!){agencies(after: $after, limit:10){feed_version{id} id agency_id}}",
			vars:         hw{"after": allEnts[1].ID},
			selector:     "agencies.#.agency_id",
			selectExpect: allIds[2:],
		},
	}
	return testcases
}

func agencyResolverStopsCursorTestcases(t *testing.T, cfg model.Config) []testcase {
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
	return testcases
}

package gqltest

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
)

func agencyResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase {
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
			Name:               "license filter: share_alike_optional = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: share_alike_optional = no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: share_alike_optional = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4,
		},
		// license: create_derived_product
		{
			Name:               "license filter: create_derived_product = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: create_derived_product = no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: create_derived_product = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4,
		},
		// license: commercial_use_allowed
		{
			Name:               "license filter: commercial_use_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: commercial_use_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: commercial_use_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4,
		},
		// license: redistribution_allowed
		{
			Name:               "license filter: redistribution_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: redistribution_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: redistribution_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4,
		},
		// license: use_without_attribution
		{
			Name:               "license filter: use_without_attribution = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  2,
		},
		{
			Name:               "license filter: use_without_attribution = no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  1,
		},
		{
			Name:               "license filter: use_without_attribution = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			Selector:           "agencies.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4,
		},
	}
	return testcases
}

func agencyResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	vars := hw{"agency_id": "caltrain-ca-us"}
	testcases := []testcase{
		{
			Name:         "basic",
			Query:        `query { agencies {agency_id}}`,
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us", "a8b6ef46-7d4d-45f8-8200-cf4f5ce9d5a6", "BART", "", "1"},
		},
		{
			Name:   "basic fields",
			Query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {onestop_id agency_id agency_name agency_lang agency_phone agency_timezone agency_url agency_email agency_fare_url cemv_support feed_version_sha1 feed_onestop_id}}`,
			Vars:   vars,
			Expect: `{"agencies":[{"agency_email":null,"agency_fare_url":null,"agency_id":"caltrain-ca-us","agency_lang":"en","agency_name":"Caltrain","agency_phone":"800-660-4287","agency_timezone":"America/Los_Angeles","agency_url":"http://www.caltrain.com","cemv_support":null,"feed_onestop_id":"CT","feed_version_sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","onestop_id":"o-9q9-caltrain"}]}`,
		},
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			Name:         "geometry",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {geometry}}`,
			Vars:         vars,
			Selector:     "agencies.0.geometry.type",
			SelectExpect: []string{"Polygon"},
		},
		{
			Name:   "feed_version",
			Query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {feed_version { sha1 }}}`,
			Vars:   vars,
			Expect: `{"agencies":[{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}]}`,
		},
		{
			Name:         "routes",
			Query:        `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {routes { route_id }}}`,
			Vars:         vars,
			Selector:     "agencies.0.routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			Name:   "route_types",
			Query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {route_types}}`,
			Vars:   vars,
			Expect: `{"agencies":[{"route_types":[2,3]}]}`,
		},
		{
			Name:   "route_types extended",
			Query:  `query{agencies(where:{feed_onestop_id:"HA"}) {route_types}}`,
			Expect: `{"agencies":[{"route_types":[0,3,4]}]}`,
		},
		// Every route type in the fixtures is already a basic one, so this says the
		// field is wired and deduplicates; tt.TestBasicRouteType is what covers the
		// fold itself, over codes no fixture carries.
		{
			Name:   "route_types_basic",
			Query:  `query($agency_id:String!) { agencies(where:{agency_id:$agency_id}) {route_types route_types_basic}}`,
			Vars:   vars,
			Expect: `{"agencies":[{"route_types":[2,3],"route_types_basic":[2,3]}]}`,
		},
		{
			Name:   "route_types_basic across a feed",
			Query:  `query{agencies(where:{feed_onestop_id:"HA"}) {route_types_basic}}`,
			Expect: `{"agencies":[{"route_types_basic":[0,3,4]}]}`,
		},
		{
			Name:         "route_types through operator",
			Query:        `query{operators(where:{onestop_id:"o-9q9-caltrain"}) {agencies{route_types}}}`,
			Selector:     "operators.0.agencies.0.route_types",
			SelectExpect: []string{"2", "3"},
		},
		// search
		{
			Name:         "search",
			Query:        `query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			Vars:         hw{"search": "Bay Area"},
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"BART"},
		},
		{
			Name:         "search",
			Query:        `query($search:String!) { agencies(where:{search:$search}) {agency_id}}`,
			Vars:         hw{"search": "caltrain"},
			Selector:     "agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us"},
		},
	}
	return testcases
}

// The WMATA feed is the only fixture with a station hierarchy: 125 served
// platforms hanging off 98 stations, plus entrances and generic nodes that no
// route serves.
func agencyResolverStopsTestcases(t testing.TB, cfg model.Config) []testcase {
	q := `query($osid:String!, $where:AgencyStopFilter, $limit:Int) { agencies(where:{onestop_id:$osid}) { stops(limit:$limit, where:$where) { stop_id location_type } } }`
	wmata := hw{"osid": "o-dqcj-wmata", "limit": 1000}
	testcases := []testcase{
		{
			Name:               "default returns the served platforms",
			Query:              q,
			Vars:               wmata,
			Selector:           "agencies.0.stops.#.location_type",
			SelectExpectUnique: []string{"0"},
			SelectExpectCount:  125,
		},
		{
			Name:              "location_type 1 returns stations",
			Query:             q,
			Vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1}},
			Selector:          "agencies.0.stops.#.stop_id",
			SelectExpectCount: 98,
		},
		{
			Name:              "explicit location_type 0 matches the default",
			Query:             q,
			Vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 0}},
			Selector:          "agencies.0.stops.#.stop_id",
			SelectExpectCount: 125,
		},
		{
			Name:              "location_type null is treated as 0",
			Query:             q,
			Vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": nil}},
			Selector:          "agencies.0.stops.#.stop_id",
			SelectExpectCount: 125,
		},
		{
			// Entrances and nodes belong to the same stations but are not served
			// and are not reached from a served stop, so they stay out.
			Name:         "location_type 2 returns no entrances",
			Query:        q,
			Vars:         hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 2}},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			Name:              "limit applies per agency",
			Query:             q,
			Vars:              hw{"osid": "o-dqcj-wmata", "limit": 10, "where": hw{"location_type": 1}},
			Selector:          "agencies.0.stops.#.stop_id",
			SelectExpectCount: 10,
		},
		{
			Name:         "agency without stations",
			Query:        q,
			Vars:         hw{"osid": "o-9q9-caltrain", "limit": 1000, "where": hw{"location_type": 1}},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			Name:               "scoped to the agency, not the feed version",
			Query:              q,
			Vars:               hw{"osid": "o-9q9-caltrain", "limit": 1000},
			Selector:           "agencies.0.stops.#.location_type",
			SelectExpectUnique: []string{"0"},
			SelectExpectCount:  64,
		},
		{
			Name:         "other stop options apply",
			Query:        q,
			Vars:         hw{"osid": "o-9q9-caltrain", "limit": 1000, "where": hw{"stop_id": "70011"}},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: []string{"70011"},
		},
		{
			// Served-by filters apply to the served platform, so they select
			// stations too. Applied to the returned row they would match no
			// station at all, since a station is never served directly.
			Name:               "served_by_route_types selects the stations of those routes",
			Query:              q,
			Vars:               hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_types": []int{3}}},
			Selector:           "agencies.0.stops.#.location_type",
			SelectExpectUnique: []string{"1"},
			SelectExpectCount:  10,
		},
		{
			// The subway serves every station, so this matches the unfiltered
			// station count.
			Name:              "served_by_route_types narrows to the subway stations",
			Query:             q,
			Vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_types": []int{1}}},
			Selector:          "agencies.0.stops.#.stop_id",
			SelectExpectCount: 98,
		},
		{
			Name:              "served_by_route_onestop_ids narrows to one line's stations",
			Query:             q,
			Vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_onestop_ids": []string{"r-dqcj-r"}}},
			Selector:          "agencies.0.stops.#.stop_id",
			SelectExpectCount: 27,
		},
		{
			Name:              "served_by_route_onestop_ids takes the union of lines",
			Query:             q,
			Vars:              hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"location_type": 1, "served_by_route_onestop_ids": []string{"r-dqcj-r", "r-dqcm-g"}}},
			Selector:          "agencies.0.stops.#.stop_id",
			SelectExpectCount: 46,
		},
		{
			// The filter matches against the agency's own routes; another
			// operator's route can never match.
			Name:         "another agency's route matches nothing",
			Query:        q,
			Vars:         hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"served_by_route_onestop_ids": []string{"r-9q9-antioch~sfia~millbrae"}}},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			// Only route Onestop IDs are accepted; anything else matches nothing
			// rather than being ignored.
			Name:         "non-route ids match nothing",
			Query:        q,
			Vars:         hw{"osid": "o-dqcj-wmata", "limit": 1000, "where": hw{"served_by_route_onestop_ids": []string{"o-dqcj-wmata"}}},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			// The relation walks platform to station and never back down: HART
			// serves 2,349 platforms and has no stations, so nothing is added.
			Name:               "unserved siblings are not pulled in",
			Query:              q,
			Vars:               hw{"osid": "o-dhv-hillsborougharearegionaltransit", "limit": 3000},
			Selector:           "agencies.0.stops.#.location_type",
			SelectExpectUnique: []string{"0"},
			SelectExpectCount:  2349,
		},
		{
			// Operators reach stops through their agencies; there is no
			// Operator.stops, just as there is no Operator.routes.
			Name:  "reached through an operator's agencies",
			Query: `query { operators(where:{onestop_id:"o-dqcj-wmata"}) { agencies { stations: stops(limit:1000, where:{location_type:1}) { stop_id } platforms: stops(limit:100, where:{location_type:0}) { stop_id } } } }`,
			Sel: []testcaseSelector{
				{Selector: "operators.0.agencies.0.stations.#.stop_id", ExpectCount: 98},
				{Selector: "operators.0.agencies.0.platforms.#.stop_id", ExpectCount: 100},
			},
		},
	}
	return testcases
}

func agencyResolverCursorTestcases(t testing.TB, cfg model.Config) []testcase {
	allEnts, err := cfg.Finder.FindAgencies(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 2, "agencies", allEnts)
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.AgencyID.Val)
	}
	testcases := []testcase{
		{
			Name:         "no cursor",
			Query:        "query{agencies(limit:10){feed_version{id} id agency_id}}",
			Selector:     "agencies.#.agency_id",
			SelectExpect: allIds,
		},
		{
			Name:         "after 0",
			Query:        "query{agencies(after: 0, limit:10){feed_version{id} id agency_id}}",
			Selector:     "agencies.#.agency_id",
			SelectExpect: allIds,
		},
		{
			Name:         "after 1st",
			Query:        "query($after: Int!){agencies(after: $after, limit:10){feed_version{id} id agency_id}}",
			Vars:         hw{"after": allEnts[1].ID},
			Selector:     "agencies.#.agency_id",
			SelectExpect: allIds[2:],
		},
	}
	return testcases
}

func agencyResolverStopsCursorTestcases(t testing.TB, cfg model.Config) []testcase {
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
	needAtLeast(t, 1, "agency stop groups", allGroups)
	allEnts := allGroups[0]
	needAtLeast(t, 2, "stations", allEnts)
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.StopID.Val)
	}
	q := `query($osid:String!, $after:Int) { agencies(where:{onestop_id:$osid}) { stops(after:$after, limit:1000, where:{location_type:1}) { stop_id } } }`
	testcases := []testcase{
		{
			Name:         "no cursor",
			Query:        q,
			Vars:         hw{"osid": osid},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: allIds,
		},
		{
			Name:         "after 0",
			Query:        q,
			Vars:         hw{"osid": osid, "after": 0},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: allIds,
		},
		{
			Name:         "after 1st",
			Query:        q,
			Vars:         hw{"osid": osid, "after": allEnts[1].ID},
			Selector:     "agencies.0.stops.#.stop_id",
			SelectExpect: allIds[2:],
		},
	}
	return testcases
}

func agencyResolverLocationTestcases(t testing.TB, cfg model.Config) []testcase {
	testFeed, testAgencyCode := "BA", "BART"
	testAgencies, err := cfg.Finder.FindAgencies(model.WithConfig(context.Background(), cfg), nil, nil, nil, &model.AgencyFilter{FeedOnestopID: &testFeed, AgencyID: &testAgencyCode})
	if err != nil {
		t.Fatal(err)
	}
	if len(testAgencies) != 1 {
		t.Fatalf("found %d active agencies with agency_id %s in %s, expected 1", len(testAgencies), testAgencyCode, testFeed)
	}
	testAgencyID := testAgencies[0].ID
	floridaFocus := tlxy.Point{Lat: 27.9506, Lon: -82.4572}
	sanJoseFocus := tlxy.Point{Lat: 37.3382, Lon: -121.8863}
	testcases := []testcase{
		// Focus tests
		{
			Name: "focus basic: West coast focus returns CA agencies before FL agencies",
			Query: `query($lat:Float!, $lon:Float!) {
				agencies(limit: 10, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					agency_id
					feed_version { feed { onestop_id } }
				}
			}`,
			Vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon},
			Selector:     "agencies.#.feed_version.feed.onestop_id",
			SelectExpect: []string{"CT", "BA", "HA", "WMATA", "ctran-flex"},
		},
		{
			Name: "focus basic: East coast focus returns FL agency before CA agencies",
			Query: `query($lat:Float!, $lon:Float!) {
				agencies(limit: 10, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					agency_id
					feed_version { feed { onestop_id } }
				}
			}`,
			Vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			Selector:     "agencies.#.feed_version.feed.onestop_id",
			SelectExpect: []string{"HA", "WMATA", "BA", "CT", "ctran-flex"},
		},
		{
			Name: "focus with pagination: maintains ordering",
			Query: `query($lat:Float!, $lon:Float!, $after:Int) {
				agencies(limit: 2, after: $after, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					id
					agency_id
					feed_version { feed { onestop_id } }
				}
			}`,
			Vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon, "after": testAgencyID},
			Selector:     "agencies.#.feed_version.feed.onestop_id",
			SelectExpect: []string{"HA", "WMATA"},
		},
	}
	return testcases
}

package gqltest

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func stopResolverLocationGroupsTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	// These stops are part of the Fairgrounds location group
	fairgroundsStop1 := "stop_id__756c0e65-32d2-4e32-a6b7-a15c3c22e6cf"
	fairgroundsStop2 := "stop_id__2e44e463-310b-4069-a709-fa0eb8f73ba9"
	fairgroundsLocationGroupID := "location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"

	testcases := []testcase{
		{
			Name: "stop with location groups - returns location group id",
			Query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups {
							location_group_id
						}
					}
				}
			}`,
			Vars:     hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop1},
			Selector: "feed_versions.0.stops.0.location_groups.#.location_group_id",
			SelectExpect: []string{
				fairgroundsLocationGroupID,
			},
		},
		{
			Name: "stop with location groups - returns location group name",
			Query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups {
							location_group_name
						}
					}
				}
			}`,
			Vars:     hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop2},
			Selector: "feed_versions.0.stops.0.location_groups.0.location_group_name",
			SelectExpect: []string{
				"Clark County Fairgroun...",
			},
		},
		{
			Name: "stop location groups - navigate to feed metadata",
			Query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups {
							location_group_id
							feed_onestop_id
							feed_version_sha1
						}
					}
				}
			}`,
			Vars:   hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop1},
			Expect: `{"feed_versions":[{"stops":[{"location_groups":[{"feed_onestop_id":"ctran-flex","feed_version_sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c","location_group_id":"location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"}],"stop_id":"stop_id__756c0e65-32d2-4e32-a6b7-a15c3c22e6cf"}]}]}`,
		},
		{
			Name: "stop without location groups - returns empty",
			Query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups {
							location_group_id
						}
					}
				}
			}`,
			// BART stop - not part of any location group
			Vars:         hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "MCAR"},
			Selector:     "feed_versions.0.stops.0.location_groups.#.location_group_id",
			SelectExpect: []string{},
		},
		{
			Name: "stop location groups with limit",
			Query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups(limit: 1) {
							location_group_id
						}
					}
				}
			}`,
			Vars:              hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop1},
			Selector:          "feed_versions.0.stops.0.location_groups.#.location_group_id",
			SelectExpectCount: 1,
		},
	}
	return testcases
}

func stopResolverCursorTestcases(t testing.TB, cfg model.Config) []testcase {
	// First 1000 stops...
	dbf := cfg.Finder
	allEnts, err := dbf.FindStops(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 100, "stops", allEnts)
	allIds := []string{}
	for _, st := range allEnts {
		allIds = append(allIds, st.StopID.Val)
	}
	testcases := []testcase{
		{
			Name:         "no cursor",
			Query:        "query{stops(limit:100){feed_version{id} id stop_id}}",
			Selector:     "stops.#.stop_id",
			SelectExpect: allIds[:100],
		},
		{
			Name:         "after 0",
			Query:        "query{stops(after: 0, limit:100){feed_version{id} id stop_id}}",
			Selector:     "stops.#.stop_id",
			SelectExpect: allIds[:100],
		},
		{
			Name:         "after 10th",
			Query:        "query($after: Int!){stops(after: $after, limit:10){feed_version{id} id stop_id}}",
			Vars:         hw{"after": allEnts[10].ID},
			Selector:     "stops.#.stop_id",
			SelectExpect: allIds[11:21],
		},
		{
			Name:         "after invalid id returns no records",
			Query:        "query($after: Int!){stops(after: $after, limit:10){feed_version{id} id stop_id}}",
			Vars:         hw{"after": 10_000_000},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func stopResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase {
	_ = t
	_ = cfg
	q := `
	query ($lic: LicenseFilter) {
		stops(limit: 10000, where: {license: $lic}) {
		  stop_id
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
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  4524,
		},
		{
			Name:               "license filter: share_alike_optional = no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  50,
		},
		{
			Name:               "license filter: share_alike_optional = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4881,
		},
		// license: create_derived_product
		{
			Name:               "license filter: create_derived_product = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  4524,
		},
		{
			Name:               "license filter: create_derived_product = no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  50,
		},
		{
			Name:               "license filter: create_derived_product = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4881,
		},
		// license: commercial_use_allowed
		{
			Name:               "license filter: commercial_use_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  4524,
		},
		{
			Name:               "license filter: commercial_use_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  50,
		},
		{
			Name:               "license filter: commercial_use_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4881,
		},
		// license: redistribution_allowed
		{
			Name:               "license filter: redistribution_allowed = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  4524,
		},
		{
			Name:               "license filter: redistribution_allowed = no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  50,
		},
		{
			Name:               "license filter: redistribution_allowed = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4881,
		},
		// license: use_without_attribution
		{
			Name:               "license filter: use_without_attribution = yes",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"HA", "WMATA"},
			SelectExpectCount:  4524,
		},
		{
			Name:               "license filter: use_without_attribution = no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"BA"},
			SelectExpectCount:  50,
		},
		{
			Name:               "license filter: use_without_attribution = exclude_no",
			Query:              q,
			Vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			Selector:           "stops.#.feed_version.feed.onestop_id",
			SelectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			SelectExpectCount:  4881,
		},
	}
	return testcases
}

func stopResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	bartStops := []string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "BAYF", "CAST", "CIVC", "COLS", "COLM", "CONC", "DALY", "DBRK", "DUBL", "DELN", "PLZA", "EMBR", "FRMT", "FTVL", "GLEN", "HAYW", "LAFY", "LAKE", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "OAKL", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SANL", "SHAY", "SSAN", "UCTY", "WCRK", "WARM", "WDUB", "WOAK"}
	caltrainRailStops := []string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322"}
	caltrainBusStops := []string{"777402", "777403"}
	caltrainStops := []string{}
	caltrainStops = append(caltrainStops, caltrainRailStops...)
	caltrainStops = append(caltrainStops, caltrainBusStops...)
	allStops := []string{}
	allStops = append(allStops, bartStops...)
	allStops = append(allStops, caltrainStops...)
	vars := hw{"stop_id": "MCAR"}

	testcases := []testcase{
		{
			Name:         "basic",
			Query:        `query($feed_version_sha1:String!) { stops(where:{feed_version_sha1:$feed_version_sha1}) { stop_id } }`, // just check BART
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			Selector:     "stops.#.stop_id",
			SelectExpect: bartStops,
		},
		{
			Name:   "basic fields",
			Query:  `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {onestop_id feed_version_sha1 feed_onestop_id location_type stop_code stop_desc stop_id stop_name stop_timezone stop_url wheelchair_boarding zone_id} }`,
			Vars:   vars,
			Expect: `{"stops":[{"feed_onestop_id":"BA","feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","location_type":0,"onestop_id":"s-9q9p1wxf72-macarthur","stop_code":null,"stop_desc":null,"stop_id":"MCAR","stop_name":"MacArthur","stop_timezone":null,"stop_url":"http://www.bart.gov/stations/MCAR/","wheelchair_boarding":1,"zone_id":"MCAR"}]}`,
		},
		{
			Name:   "feed_version",
			Query:  `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {feed_version_sha1} }`,
			Vars:   vars,
			Expect: `{"stops":[{"feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}]}`,
		},
		{
			Name:         "route_stops",
			Query:        `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {route_stops{route{route_id route_short_name}}} }`,
			Vars:         vars,
			Selector:     "stops.0.route_stops.#.route.route_id",
			SelectExpect: []string{"01", "03", "07"},
		},
		{
			Name:  "timestamps",
			Query: `query($stop_id: String!) { stops(where:{stop_id:$stop_id}) { created_at updated_at } }`,
			Vars:  vars,
			F: func(t *testing.T, jj string) {
				createdAt := gjson.Get(jj, "stops.0.created_at").String()
				updatedAt := gjson.Get(jj, "stops.0.updated_at").String()
				assert.NotEmpty(t, createdAt, "created_at should be exposed and non-empty")
				assert.NotEmpty(t, updatedAt, "updated_at should be exposed and non-empty")
			},
		},
		{
			Name:         "where onestop_id",
			Query:        `query{stops(where:{onestop_id:"s-9q9k658fd1-sanjosediridoncaltrain"}) {stop_id} }`,
			Vars:         vars,
			Selector:     "stops.0.stop_id",
			SelectExpect: []string{"70262"},
		},
		{
			Name:         "where feed_version_sha1",
			Query:        `query($feed_version_sha1:String!) { stops(where:{feed_version_sha1:$feed_version_sha1}) { stop_id } }`, // just check BART
			Vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			Selector:     "stops.#.stop_id",
			SelectExpect: bartStops,
		},
		{
			Name:         "where feed_onestop_id",
			Query:        `query{stops(where:{feed_onestop_id:"BA"}) { stop_id } }`, // just check BART
			Selector:     "stops.#.stop_id",
			SelectExpect: bartStops,
		},
		{
			Name:         "where stop_id",
			Query:        `query{stops(where:{stop_id:"12TH"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"12TH"},
		},
		{
			Name:         "where search",
			Query:        `query{stops(where:{search:"macarthur"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"MCAR", "MCAR_S"},
		},
		{
			Name:         "where search 2",
			Query:        `query{stops(where:{search:"ftvl"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"FTVL"},
		},
		{
			Name:         "where search 3",
			Query:        `query{stops(where:{search:"warm springs"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"WARM"},
		},
		// served_by_route_types
		{
			Name:         "served_by_route_types=[2]",
			Query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_types:[0]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713"},
		},
		{
			Name:         "served_by_route_types=[0,4]",
			Query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_types:[0,4]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713", "8030", "8031", "8032", "8033", "8034", "8035", "8038", "8039", "8040", "8041", "8042", "8043"},
		},
		{
			Name:         "served_by_route_type=0",
			Query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_type:0}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713"},
		},
		{
			Name:         "served_by_route_type=4",
			Query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_type:4}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"8030", "8031", "8032", "8033", "8034", "8035", "8038", "8039", "8040", "8041", "8042", "8043"},
		},
		{
			Name:         "served_by_route_type=0 served_by_route_types=[4]",
			Query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_type:0, served_by_route_types:[4]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713", "8030", "8031", "8032", "8033", "8034", "8035", "8038", "8039", "8040", "8041", "8042", "8043"},
		},
		// served_by_onestop_ids
		{
			Name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit",
			Query:        `query{stops(where:{served_by_onestop_ids:["o-9q9-bayarearapidtransit"]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: bartStops,
		},
		{
			Name:         "served_by_onestop_ids=o-9q9-caltrain",
			Query:        `query{stops(where:{served_by_onestop_ids:["o-9q9-caltrain"]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: caltrainStops, // caltrain stops minus a couple non-service stops
		},
		{
			Name:         "served_by_onestop_ids=r-9q9-antioch~sfia~millbrae",
			Query:        `query{stops(where:{served_by_onestop_ids:["r-9q9-antioch~sfia~millbrae"]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "BALB", "CIVC", "COLM", "CONC", "DALY", "EMBR", "GLEN", "LAFY", "MCAR", "MCAR_S", "MLBR", "MONT", "NCON", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "ROCK", "SBRN", "SFIA", "SSAN", "WCRK", "WOAK"}, // yellow line stops
		},
		{
			Name:         "served_by_onestop_ids=r-9q9-antioch~sfia~millbrae,r-9q8y-richmond~dalycity~millbrae",
			Query:        `query{stops(where:{served_by_onestop_ids:["r-9q9-antioch~sfia~millbrae","r-9q8y-richmond~dalycity~millbrae"]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "CIVC", "COLM", "CONC", "DALY", "DBRK", "DELN", "PLZA", "EMBR", "GLEN", "LAFY", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SSAN", "WCRK", "WOAK"}, // combination of yellow and red line stops
		},
		{
			Name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit,r-9q9-antioch~sfia~millbrae",
			Query:        `query{stops(where:{served_by_onestop_ids:["o-9q9-bayarearapidtransit","r-9q9-antioch~sfia~millbrae"]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: bartStops, // all bart stops
		},
		{
			Name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit,o-9q9-caltrain",
			Query:        `query{stops(limit:1000,where:{served_by_onestop_ids:["o-9q9-bayarearapidtransit","o-9q9-caltrain"]}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: allStops, // all stops
		},
		// {
		// 	"served_by_route_types=2,served_by_onestop_ids=o-9q9-bayarearapidtransit,o-9q9-caltrain",
		// 	`query{stops(where:{served_by_onestop_ids:["o-9q9-bayarearapidtransit","o-9q9-caltrain"], served_by_route_types:[2]}) { stop_id } }`,
		// 	hw{},
		// 	``,
		// 	"stops.#.stop_id",
		// 	caltrainRailStops,
		// },
		// TODO: parent, children; test data has no stations.
		// TODO: level, pathways_from_stop, pathways_to_stop: test data has no pathways...
		// stop_times
		{
			Name:         "stop_times",
			Query:        `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {stop_times { trip { trip_id} }} }`,
			Vars:         hw{"stop_id": "70302"}, // Morgan hill
			Selector:     "stops.0.stop_times.#.trip.trip_id",
			SelectExpect: []string{"268", "274", "156"},
		},
		{
			Name:         "stop_times where weekday_morning",
			Query:        `query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:21600, end_time:25200}) { trip { trip_id} }} }`,
			Vars:         hw{"stop_id": "MCAR", "service_date": "2018-05-29"},
			Selector:     "stops.0.stop_times.#.trip.trip_id",
			SelectExpect: []string{"3830503WKDY", "3850526WKDY", "3610541WKDY", "3630556WKDY", "3650611WKDY", "2210533WKDY", "2230548WKDY", "2250603WKDY", "2270618WKDY", "4410518WKDY", "4430533WKDY", "4450548WKDY", "4470603WKDY"},
		},
		{
			Name:         "stop_times where sunday_morning",
			Query:        `query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:21600, end_time:36000}) { trip { trip_id} }} }`,
			Vars:         hw{"stop_id": "MCAR", "service_date": "2018-05-27"},
			Selector:     "stops.0.stop_times.#.trip.trip_id",
			SelectExpect: []string{"3730756SUN", "3750757SUN", "3770801SUN", "3790821SUN", "3610841SUN", "3630901SUN", "2230800SUN", "2250748SUN", "2270808SUN", "2290828SUN", "2310848SUN", "2330908SUN"},
		},
		{
			Name:         "stop_times where saturday_evening",
			Query:        `query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:57600, end_time:72000}) { trip { trip_id} }} }`,
			Vars:         hw{"stop_id": "MCAR", "service_date": "2018-05-26"},
			Selector:     "stops.0.stop_times.#.trip.trip_id",
			SelectExpect: []string{"3611521SAT", "3631541SAT", "3651601SAT", "3671621SAT", "3691641SAT", "3711701SAT", "3731721SAT", "3751741SAT", "3771801SAT", "3791821SAT", "3611841SAT", "3631901SAT", "2231528SAT", "2251548SAT", "2271608SAT", "2291628SAT", "2311648SAT", "2331708SAT", "2351728SAT", "2211748SAT", "2231808SAT", "2251828SAT", "2271848SAT", "2291908SAT", "4471533SAT", "4491553SAT", "4511613SAT", "4531633SAT", "4411653SAT", "4431713SAT", "4451733SAT", "4471753SAT", "4491813SAT", "4511833SAT", "4531853SAT"},
		},
		// serviced
		{
			Name:         "stop serviced=true",
			Query:        `query{stops(where:{feed_onestop_id:"EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:true}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"FUR_CREEK_RES", "BEATTY_AIRPORT", "BULLFROG", "STAGECOACH", "NADAV", "NANAA", "DADAN", "EMSI", "AMV"},
		},
		{
			Name:         "stop serviced=false",
			Query:        `query{stops(where:{feed_onestop_id:"EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:false}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"NOTRIPS"},
		},
		// route_type
		{
			// tampa street car
			Name:         "stop route_type=0",
			Query:        `query{stops(where:{served_by_route_type:0}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713"},
		},
		{
			Name:         "stop route_type=0 empty",
			Query:        `query{stops(where:{served_by_route_type:0, feed_onestop_id:"BA"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			// BART
			Name:         "stop route_type=1",
			Query:        `query{stops(where:{served_by_route_type:1, feed_onestop_id:"BA"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: bartStops,
		},
		{
			Name:         "stop route_type=1 empty",
			Query:        `query{stops(where:{served_by_route_type:1, feed_onestop_id:"CT"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			// Caltrain
			Name:         "stop route_type=2",
			Query:        `query{stops(where:{served_by_route_type:2, feed_onestop_id:"CT"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: caltrainRailStops,
		},
		{
			Name:         "stop route_type=2 empty",
			Query:        `query{stops(where:{served_by_route_type:2, feed_onestop_id:"BA"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			// Caltrain
			Name:         "stop route_type=3",
			Query:        `query{stops(where:{served_by_route_type:3, feed_onestop_id:"CT"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: caltrainBusStops,
		},
		{
			Name:         "stop route_type=3 empty",
			Query:        `query{stops(where:{served_by_route_type:3, feed_onestop_id:"BA"}) { stop_id } }`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func stopResolverLocationTestcases(t testing.TB, cfg model.Config) []testcase {
	vars := hw{"stop_id": "MCAR"}
	featureBig := decodeGeojson(t, `{
		"id": "big",
		"geometry": {
			"type": "Polygon",
			"coordinates": [
			[
				[
				-122.26056968514843,
				37.82764708485912
				],
				[
				-122.27696196387409,
				37.802600070178286
				],
				[
				-122.2689749038766,
				37.800104259637735
				],
				[
				-122.25401406862724,
				37.82458700630413
				],
				[
				-122.26056968514843,
				37.82764708485912
				]
			]
		]
	}}`)

	featureSmall := decodeGeojson(t, `{
		"id": "small",
		"geometry": {
			"coordinates": [
			[
				[
				-122.27340683066507,
				37.806198590527046
				],
				[
				-122.27340683066507,
				37.801023273225354
				],
				[
				-122.26772379009174,
				37.801023273225354
				],
				[
				-122.26772379009174,
				37.806198590527046
				],
				[
				-122.27340683066507,
				37.806198590527046
				]
			]
			],
			"type": "Polygon"
		}
	}`)

	bartStops := []string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "BAYF", "CAST", "CIVC", "COLS", "COLM", "CONC", "DALY", "DBRK", "DUBL", "DELN", "PLZA", "EMBR", "FRMT", "FTVL", "GLEN", "HAYW", "LAFY", "LAKE", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "OAKL", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SANL", "SHAY", "SSAN", "UCTY", "WCRK", "WARM", "WDUB", "WOAK"}

	floridaFocus := tlxy.Point{Lat: 27.9506, Lon: -82.4572}
	sanJoseFocus := tlxy.Point{Lat: 37.3382, Lon: -121.8863}
	sanJoseRadiusMeters := 10_000.0 // 10km

	testStopCode := "70252"
	testStops, err := cfg.Finder.FindStops(model.WithConfig(context.Background(), cfg), nil, nil, nil, &model.StopFilter{StopID: &testStopCode})
	if err != nil {
		t.Fatal(err)
	}
	if len(testStops) != 1 {
		t.Fatalf("found %d active stops with stop_id %s, expected 1", len(testStops), testStopCode)
	}
	testStopID := testStops[0].ID

	testcases := []testcase{
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			Name:         "geometry",
			Query:        `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {geometry} }`,
			Vars:         vars,
			Selector:     "stops.0.geometry.type",
			SelectExpect: []string{"Point"},
		},
		{
			Name:         "where near 10m",
			Query:        `query {stops(where:{near:{lon:-122.407974,lat:37.784471,radius:10.0}}) {stop_id onestop_id geometry}}`,
			Vars:         vars,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"POWL"},
		},
		{
			Name:         "where near 2000m",
			Query:        `query {stops(where:{near:{lon:-122.407974,lat:37.784471,radius:2000.0}}) {stop_id onestop_id geometry}}`,
			Vars:         vars,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"70011", "70012", "CIVC", "EMBR", "MONT", "POWL"},
		},
		// within features
		{
			Name:         "where within polygon",
			Query:        `query{stops(where:{within:{type:"Polygon",coordinates:[[[-122.396,37.8],[-122.408,37.79],[-122.393,37.778],[-122.38,37.787],[-122.396,37.8]]]}}){id stop_id}}`,
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"EMBR", "MONT"},
		},
		{
			Name:   "where locations within_features 1",
			Query:  `query($features: [Feature]) {stops(where:{location:{features:$features}}){stop_id within_features}}`,
			Vars:   hw{"features": []hw{featureSmall}},
			Expect: `{"stops":[{"stop_id":"12TH","within_features":["small"]}]}`,
		},
		{
			Name:   "where locations within_features 2",
			Query:  `query($features: [Feature]) {stops(where:{location:{features:$features}}){stop_id within_features}}`,
			Vars:   hw{"features": []hw{featureSmall, featureBig}},
			Expect: `{"stops":[{"stop_id":"12TH","within_features":["small","big"]},{"stop_id":"19TH","within_features":["big"]},{"stop_id":"19TH_N","within_features":["big"]}]}`,
		},
		// nearby stops
		{
			Name: "nearby stops 1000m",
			Query: `query($stop_id:String!, $radius:Float!) {
				stops(where: {feed_onestop_id: "BA", stop_id: $stop_id}) {
				  stop_id
				  nearby_stops(radius:$radius, limit:10) {
					stop_id
				  }
				}
			  }			  `,
			Vars:         hw{"stop_id": "19TH", "radius": 1000},
			Selector:     "stops.0.nearby_stops.#.stop_id",
			SelectExpect: []string{"19TH", "19TH_N", "12TH"},
		},
		{
			Name: "nearby stops 2000m",
			Query: `query($stop_id:String!, $radius:Float!) {
				stops(where: {feed_onestop_id: "BA", stop_id: $stop_id}) {
				  stop_id
				  nearby_stops(radius:$radius, limit:10) {
					stop_id
				  }
				}
			  }			  `,
			Vars:         hw{"stop_id": "19TH", "radius": 2000},
			Selector:     "stops.0.nearby_stops.#.stop_id",
			SelectExpect: []string{"19TH", "19TH_N", "12TH", "LAKE"},
		},
		// bbox
		{
			Name: "bbox 1",
			Query: `query($bbox:BoundingBox) {
				stops(where: {bbox:$bbox}) {
				  stop_id
				}
			  }			  `,
			Vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"19TH", "19TH_N"},
		},
		{
			Name: "bbox 2",
			Query: `query($bbox:BoundingBox) {
				stops(where: {bbox:$bbox}) {
				  stop_id
				}
			  }			  `,
			Vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{},
		},
		{
			Name: "bbox too large",
			Query: `query($bbox:BoundingBox) {
				stops(where: {bbox:$bbox}) {
				  stop_id
				}
			  }			  `,
			Vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			ExpectError: true,
		},
		// this test is just for debugging purposes
		{
			Name: "nearby stops check n+1 query",
			Query: `query($radius:Float!) {
				stops(where: {feed_onestop_id: "BA"}) {
				  stop_id
				  nearby_stops(radius:$radius, limit:10) {
					stop_id
				  }
				}
			  }			  `,
			Vars:         hw{"radius": 1000},
			Selector:     "stops.#.stop_id",
			SelectExpect: bartStops,
		},
		// Focus test cases
		{
			Name: "focus basic: Florida focus point returns HA stops first",
			Query: `query($lat:Float!, $lon:Float!) {
				stops(limit: 10, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					stop_id
					feed_version { feed { onestop_id } }
				}
			}`,
			Vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			Selector:     "stops.#.feed_version.feed.onestop_id",
			SelectExpect: []string{"HA", "HA", "HA", "HA", "HA", "HA", "HA", "HA", "HA", "HA"},
		},
		{
			Name: "focus basic: San Jose focus point returns CT stops first",
			Query: `query($lat:Float!, $lon:Float!) {
				stops(limit: 10, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					stop_id
					feed_version { feed { onestop_id } }
				}
			}`,
			Vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon},
			Selector:     "stops.#.feed_version.feed.onestop_id",
			SelectExpect: []string{"CT", "CT", "CT", "CT", "CT", "CT", "CT", "CT", "CT", "CT"},
		},
		{
			Name: "focus with feed filter: HA stops only, ordered by distance",
			Query: `query($lat:Float!, $lon:Float!) {
				stops(limit: 10, where: {feed_onestop_id: "HA", location: {focus: {lat: $lat, lon: $lon}}}) {
					stop_id
					geometry
				}
			}`,
			Vars: hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			F: func(t *testing.T, jj string) {
				// Verify the query returns stops
				stopIds := gjson.Get(jj, "stops.#.stop_id").Array()
				assert.Equal(t, 10, len(stopIds), "should return exactly 10 stops")

				// Parse geometries and verify stops are sorted by increasing distance
				var prevDistance float64
				for i, stop := range gjson.Get(jj, "stops").Array() {
					// Parse into tlxy.Point
					coords := stop.Get("geometry.coordinates").Array()
					stopPoint := tlxy.Point{
						Lon: coords[0].Float(),
						Lat: coords[1].Float(),
					}

					// Calculate distance from focus point
					// Verify distance is increasing (or equal for stops at same location)
					distance := tlxy.DistanceHaversine(floridaFocus, stopPoint)
					if i > 0 {
						assert.GreaterOrEqual(t, distance, prevDistance,
							"stop index %d (stop_id=%s) at distance %.2f should be >= previous distance %.2f",
							i, stop.Get("stop_id").String(), distance, prevDistance)
					}
					prevDistance = distance
				}
			},
		},
		{
			Name: "focus with pagination: first page",
			Query: `query($lat:Float!, $lon:Float!) {
				stops: stops(limit: 10, where: {feed_onestop_id: "CT", location: {focus: {lat: $lat, lon: $lon}}}) {
					id
					stop_id
				}
			}`,
			Vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"777402", "70261", "70262", "70251", "70252", "70272", "70271", "777403", "70241", "70242"},
		},
		{
			Name: "focus with pagination: second page maintains ordering",
			Query: `query($lat:Float!, $lon:Float!, $after:Int!) {
				stops: stops(after: $after, limit: 10, where: {feed_onestop_id: "CT", location: {focus: {lat: $lat, lon: $lon}}}) {
					id
					stop_id
				}
			}`,
			Vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon, "after": testStopID},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"70272", "70271", "777403", "70241", "70242", "70282", "70281", "70232", "70231", "70291"},
		},
		{
			Name: "focus with near filter: combined spatial queries",
			Query: `query($lat:Float!, $lon:Float!, $radius:Float!) {
				stops(limit: 100, where: {
					feed_onestop_id: "CT",
					location: {
						focus: {lat: $lat, lon: $lon},
						near: {lat: $lat, lon: $lon, radius: $radius}
					}
				}) {
					stop_id
				}
			}`,
			Vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon, "radius": sanJoseRadiusMeters},
			Selector:     "stops.#.stop_id",
			SelectExpect: []string{"777402", "70261", "70262", "70251", "70252", "70272", "70271", "777403", "70241", "70242", "70282", "70281"},
		},
	}
	return testcases
}

// Fixture stops in feed versions that resolve 2018-05-29 differently: BA's
// service window covers it, so BA answers for the date as asked, while the
// others have no window covering it and relocate it into their own fallback
// week.
const serviceWindowDate = "2018-05-29"

var (
	serviceWindowInWindow  = [2]string{"BA", "MONT"}
	serviceWindowRelocates = [][2]string{{"CT", "70011"}, {"HA", "1014"}, {"WMATA", "ENT_A01_C01_E"}}
)

// stopResolverServiceWindowIsPerFeedVersionTest checks that each feed version
// in one `stops` selection resolves the date against its own service window.
//
// Every stop in the selection shares a single filter, and the date resolved for
// one feed version must not follow that filter into the next. A finder that
// visits feed versions in no fixed order would then make BA's answer depend on
// whether BA came first, so the same request would return different stop times
// from one call to the next.
func stopResolverServiceWindowIsPerFeedVersionTest(t *testing.T, c *client.Client, cfg model.Config) {
	query := func(q string) string {
		var resp map[string]interface{}
		if err := c.Post(q, &resp); err != nil {
			t.Fatal(err)
		}
		return gqlcase.ToJSON(resp)
	}
	// Resolved rather than hardcoded: these are unstable database ids.
	stopID := func(feed string, gtfsStopID string) string {
		j := query(`query{stops(where:{feed_onestop_id:"` + feed + `", stop_id:"` + gtfsStopID + `"}){id}}`)
		id := gjson.Get(j, "stops.0.id")
		if !id.Exists() {
			t.Fatalf("fixture stop %s:%s not found", feed, gtfsStopID)
		}
		return id.String()
	}
	inWindow := stopID(serviceWindowInWindow[0], serviceWindowInWindow[1])
	// Three more feed versions. Only the visit order decides whether a leak
	// shows, so one alone would leave a coin flip; with these the odds of
	// missing it across the runs below are about one in a million.
	var relocating []string
	for _, s := range serviceWindowRelocates {
		relocating = append(relocating, stopID(s[0], s[1]))
	}

	// stop_times for one stop out of a request covering all the given stops.
	departures := func(want string, ids ...string) string {
		j := query(`query{stops(ids:[` + strings.Join(ids, ",") + `]){
			id stop_times(where:{date:"` + serviceWindowDate + `", use_service_window:true}){ service_date departure_time }
		}}`)
		for _, s := range gjson.Get(j, "stops").Array() {
			if s.Get("id").String() == want {
				return s.Get("stop_times").Raw
			}
		}
		t.Fatalf("stop %s missing from response %s", want, j)
		return ""
	}

	// The answer when nothing else shares the request.
	alone := departures(inWindow, inWindow)
	assert.Contains(t, alone, `"service_date":"`+serviceWindowDate+`"`,
		"fixture should answer for the requested date, not a relocated one")

	// The premise: at least one of the others really does relocate the date,
	// its service window not covering the date. Without this the test can keep
	// passing while asserting nothing, if the fixtures ever drift so that no
	// feed version resolves the date differently. Read from the window itself:
	// a relocated stop time reports the requested date.
	relocated := false
	for i, id := range relocating {
		fvid := gjson.Get(query(`query{stops(ids:[`+id+`]){feed_version{id}}}`), "stops.0.feed_version.id").Int()
		w, err := cfg.Finder.FindFeedVersionServiceWindow(model.WithConfig(context.Background(), cfg), int(fvid))
		if err != nil {
			t.Fatal(err)
		}
		day, _ := time.ParseInLocation("2006-01-02", serviceWindowDate, w.StartDate.Location())
		if day.Before(w.StartDate) || day.After(w.EndDate) {
			relocated = true
			break
		}
		t.Logf("%v did not relocate %s (window %s to %s)", serviceWindowRelocates[i], serviceWindowDate, w.StartDate.Format("2006-01-02"), w.EndDate.Format("2006-01-02"))
	}
	assert.True(t, relocated, "fixture should include a feed version that relocates the date")

	// Adding stops whose feed versions relocate the date must not change BA's
	// answer, and must not change it differently from one call to the next.
	all := append([]string{inWindow}, relocating...)
	for i := 0; i < 10; i++ {
		if !assert.JSONEq(t, alone, departures(inWindow, all...), "run %d differs from the single-stop query", i) {
			t.FailNow()
		}
	}
}

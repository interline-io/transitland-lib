package gql

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestStopResolver(t *testing.T) {
	c, cfg := newTestClient(t)
	queryTestcases(t, c, stopResolverTestcases(t, cfg))
}

func TestStopResolverLocation(t *testing.T) {
	c, cfg := newTestClient(t)
	queryTestcases(t, c, stopResolverLocationTestcases(t, cfg))
}

func TestStopResolver_Cursor(t *testing.T) {
	c, cfg := newTestClient(t)
	queryTestcases(t, c, stopResolverCursorTestcases(t, cfg))
}

func TestStopResolver_PreviousOnestopID(t *testing.T) {
	c, cfg := newTestClient(t)
	queryTestcases(t, c, stopResolverPreviousOnestopIDTestcases(t, cfg))
}

func TestStopResolver_License(t *testing.T) {
	c, cfg := newTestClient(t)
	queryTestcases(t, c, stopResolverLicenseTestcases(t, cfg))
}

func TestStopResolver_LocationGroups(t *testing.T) {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	// These stops are part of the Fairgrounds location group
	fairgroundsStop1 := "stop_id__756c0e65-32d2-4e32-a6b7-a15c3c22e6cf"
	fairgroundsStop2 := "stop_id__2e44e463-310b-4069-a709-fa0eb8f73ba9"
	fairgroundsLocationGroupID := "location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"

	testcases := []testcase{
		{
			name: "stop with location groups - returns location group id",
			query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups {
							location_group_id
						}
					}
				}
			}`,
			vars:     hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop1},
			selector: "feed_versions.0.stops.0.location_groups.#.location_group_id",
			selectExpect: []string{
				fairgroundsLocationGroupID,
			},
		},
		{
			name: "stop with location groups - returns location group name",
			query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups {
							location_group_name
						}
					}
				}
			}`,
			vars:     hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop2},
			selector: "feed_versions.0.stops.0.location_groups.0.location_group_name",
			selectExpect: []string{
				"Clark County Fairgroun...",
			},
		},
		{
			name: "stop location groups - navigate to feed metadata",
			query: `query($sha1: String!, $stop_id: String!) {
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
			vars:   hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop1},
			expect: `{"feed_versions":[{"stops":[{"location_groups":[{"feed_onestop_id":"ctran-flex","feed_version_sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c","location_group_id":"location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"}],"stop_id":"stop_id__756c0e65-32d2-4e32-a6b7-a15c3c22e6cf"}]}]}`,
		},
		{
			name: "stop without location groups - returns empty",
			query: `query($sha1: String!, $stop_id: String!) {
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
			vars:         hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "MCAR"},
			selector:     "feed_versions.0.stops.0.location_groups.#.location_group_id",
			selectExpect: []string{},
		},
		{
			name: "stop location groups with limit",
			query: `query($sha1: String!, $stop_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					stops(where: {stop_id: $stop_id}) {
						stop_id
						location_groups(limit: 1) {
							location_group_id
						}
					}
				}
			}`,
			vars:              hw{"sha1": ctranFlexSha1, "stop_id": fairgroundsStop1},
			selector:          "feed_versions.0.stops.0.location_groups.#.location_group_id",
			selectExpectCount: 1,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestStopResolver_AdminCache(t *testing.T) {
	type canLoadAdmins interface {
		LoadAdmins(context.Context) error
	}
	c, cfg := newTestClient(t)
	if v, ok := cfg.Finder.(canLoadAdmins); !ok {
		t.Fatal("finder cant load admins")
	} else {
		if err := v.LoadAdmins(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	q := `query($feed_version_sha1:String!, $stop_id:String!) { stops(where:{stop_id:$stop_id, feed_version_sha1:$feed_version_sha1}) { place { adm0_name adm1_name adm0_iso adm1_iso } } }`
	tcs := []testcase{
		{
			name:         "usa",
			query:        q,
			vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			selector:     "stops.#.place.adm0_name",
			selectExpect: []string{"United States of America"},
		},
		{
			name:         "california",
			query:        q,
			vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			selector:     "stops.#.place.adm1_name",
			selectExpect: []string{"California"},
		},
		{
			name:         "adm0_iso",
			query:        q,
			vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			selector:     "stops.#.place.adm0_iso",
			selectExpect: []string{"US"},
		},
		{
			name:         "adm1_iso",
			query:        q,
			vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "stop_id": "FTVL"},
			selector:     "stops.#.place.adm1_iso",
			selectExpect: []string{"US-CA"},
		},
		{
			name:         "florida",
			query:        q,
			vars:         hw{"feed_version_sha1": "c969427f56d3a645195dd8365cde6d7feae7e99b", "stop_id": "8032"},
			selector:     "stops.#.place.adm1_name",
			selectExpect: []string{"Florida"},
		},
	}
	queryTestcases(t, c, tcs)
}

func BenchmarkStopResolver(b *testing.B) {
	c, cfg := newTestClient(b)
	benchmarkTestcases(b, c, stopResolverTestcases(b, cfg))
}

func decodeGeojson(d string) hw {
	featureAbc := hw{}
	if err := json.Unmarshal([]byte(d), &featureAbc); err != nil {
		panic(err)
	}
	return featureAbc
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

	stopObsFvid := 0
	if err := cfg.Finder.DBX().QueryRowx("select feed_version_id from ext_performance_stop_observations limit 1").Scan(&stopObsFvid); err != nil {
		t.Errorf("could not get fvid for stop observation test: %s", err.Error())
	}
	testcases := []testcase{
		{
			name:         "basic",
			query:        `query($feed_version_sha1:String!) { stops(where:{feed_version_sha1:$feed_version_sha1}) { stop_id } }`, // just check BART
			vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			selector:     "stops.#.stop_id",
			selectExpect: bartStops,
		},
		{
			name:   "basic fields",
			query:  `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {onestop_id feed_version_sha1 feed_onestop_id location_type stop_code stop_desc stop_id stop_name stop_timezone stop_url wheelchair_boarding zone_id} }`,
			vars:   vars,
			expect: `{"stops":[{"feed_onestop_id":"BA","feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","location_type":0,"onestop_id":"s-9q9p1wxf72-macarthur","stop_code":null,"stop_desc":null,"stop_id":"MCAR","stop_name":"MacArthur","stop_timezone":null,"stop_url":"http://www.bart.gov/stations/MCAR/","wheelchair_boarding":1,"zone_id":"MCAR"}]}`,
		},
		{
			name:   "feed_version",
			query:  `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {feed_version_sha1} }`,
			vars:   vars,
			expect: `{"stops":[{"feed_version_sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}]}`,
		},
		{
			name:         "route_stops",
			query:        `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {route_stops{route{route_id route_short_name}}} }`,
			vars:         vars,
			selector:     "stops.0.route_stops.#.route.route_id",
			selectExpect: []string{"01", "03", "07"},
		},

		{
			name:         "where onestop_id",
			query:        `query{stops(where:{onestop_id:"s-9q9k658fd1-sanjosediridoncaltrain"}) {stop_id} }`,
			vars:         vars,
			selector:     "stops.0.stop_id",
			selectExpect: []string{"70262"},
		},
		{
			name:         "where feed_version_sha1",
			query:        `query($feed_version_sha1:String!) { stops(where:{feed_version_sha1:$feed_version_sha1}) { stop_id } }`, // just check BART
			vars:         hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			selector:     "stops.#.stop_id",
			selectExpect: bartStops,
		},
		{
			name:         "where feed_onestop_id",
			query:        `query{stops(where:{feed_onestop_id:"BA"}) { stop_id } }`, // just check BART
			selector:     "stops.#.stop_id",
			selectExpect: bartStops,
		},
		{
			name:         "where stop_id",
			query:        `query{stops(where:{stop_id:"12TH"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"12TH"},
		},
		{
			name:         "where search",
			query:        `query{stops(where:{search:"macarthur"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"MCAR", "MCAR_S"},
		},
		{
			name:         "where search 2",
			query:        `query{stops(where:{search:"ftvl"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"FTVL"},
		},
		{
			name:         "where search 3",
			query:        `query{stops(where:{search:"warm springs"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"WARM"},
		},
		// served_by_route_types
		{
			name:         "served_by_route_types=[2]",
			query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_types:[0]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713"},
		},
		{
			name:         "served_by_route_types=[0,4]",
			query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_types:[0,4]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713", "8030", "8031", "8032", "8033", "8034", "8035", "8038", "8039", "8040", "8041", "8042", "8043"},
		},
		{
			name:         "served_by_route_type=0",
			query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_type:0}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713"},
		},
		{
			name:         "served_by_route_type=4",
			query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_type:4}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"8030", "8031", "8032", "8033", "8034", "8035", "8038", "8039", "8040", "8041", "8042", "8043"},
		},
		{
			name:         "served_by_route_type=0 served_by_route_types=[4]",
			query:        `query{stops(where:{feed_onestop_id: "HA", served_by_route_type:0, served_by_route_types:[4]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713", "8030", "8031", "8032", "8033", "8034", "8035", "8038", "8039", "8040", "8041", "8042", "8043"},
		},
		// served_by_onestop_ids
		{
			name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit",
			query:        `query{stops(where:{served_by_onestop_ids:["o-9q9-bayarearapidtransit"]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: bartStops,
		},
		{
			name:         "served_by_onestop_ids=o-9q9-caltrain",
			query:        `query{stops(where:{served_by_onestop_ids:["o-9q9-caltrain"]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: caltrainStops, // caltrain stops minus a couple non-service stops
		},
		{
			name:         "served_by_onestop_ids=r-9q9-antioch~sfia~millbrae",
			query:        `query{stops(where:{served_by_onestop_ids:["r-9q9-antioch~sfia~millbrae"]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "BALB", "CIVC", "COLM", "CONC", "DALY", "EMBR", "GLEN", "LAFY", "MCAR", "MCAR_S", "MLBR", "MONT", "NCON", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "ROCK", "SBRN", "SFIA", "SSAN", "WCRK", "WOAK"}, // yellow line stops
		},
		{
			name:         "served_by_onestop_ids=r-9q9-antioch~sfia~millbrae,r-9q8y-richmond~dalycity~millbrae",
			query:        `query{stops(where:{served_by_onestop_ids:["r-9q9-antioch~sfia~millbrae","r-9q8y-richmond~dalycity~millbrae"]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"12TH", "16TH", "19TH", "19TH_N", "24TH", "ANTC", "ASHB", "BALB", "CIVC", "COLM", "CONC", "DALY", "DBRK", "DELN", "PLZA", "EMBR", "GLEN", "LAFY", "MCAR", "MCAR_S", "MLBR", "MONT", "NBRK", "NCON", "ORIN", "PITT", "PCTR", "PHIL", "POWL", "RICH", "ROCK", "SBRN", "SFIA", "SSAN", "WCRK", "WOAK"}, // combination of yellow and red line stops
		},
		{
			name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit,r-9q9-antioch~sfia~millbrae",
			query:        `query{stops(where:{served_by_onestop_ids:["o-9q9-bayarearapidtransit","r-9q9-antioch~sfia~millbrae"]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: bartStops, // all bart stops
		},
		{
			name:         "served_by_onestop_ids=o-9q9-bayarearapidtransit,o-9q9-caltrain",
			query:        `query{stops(limit:1000,where:{served_by_onestop_ids:["o-9q9-bayarearapidtransit","o-9q9-caltrain"]}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: allStops, // all stops
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
		// TODO: census_geographies
		// stop_times
		{
			name:         "stop_times",
			query:        `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {stop_times { trip { trip_id} }} }`,
			vars:         hw{"stop_id": "70302"}, // Morgan hill
			selector:     "stops.0.stop_times.#.trip.trip_id",
			selectExpect: []string{"268", "274", "156"},
		},
		{
			name:         "stop_times where weekday_morning",
			query:        `query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:21600, end_time:25200}) { trip { trip_id} }} }`,
			vars:         hw{"stop_id": "MCAR", "service_date": "2018-05-29"},
			selector:     "stops.0.stop_times.#.trip.trip_id",
			selectExpect: []string{"3830503WKDY", "3850526WKDY", "3610541WKDY", "3630556WKDY", "3650611WKDY", "2210533WKDY", "2230548WKDY", "2250603WKDY", "2270618WKDY", "4410518WKDY", "4430533WKDY", "4450548WKDY", "4470603WKDY"},
		},
		{
			name:         "stop_times where sunday_morning",
			query:        `query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:21600, end_time:36000}) { trip { trip_id} }} }`,
			vars:         hw{"stop_id": "MCAR", "service_date": "2018-05-27"},
			selector:     "stops.0.stop_times.#.trip.trip_id",
			selectExpect: []string{"3730756SUN", "3750757SUN", "3770801SUN", "3790821SUN", "3610841SUN", "3630901SUN", "2230800SUN", "2250748SUN", "2270808SUN", "2290828SUN", "2310848SUN", "2330908SUN"},
		},
		{
			name:         "stop_times where saturday_evening",
			query:        `query($stop_id: String!, $service_date:Date!) {  stops(where:{stop_id:$stop_id}) {stop_times(where:{service_date:$service_date, start_time:57600, end_time:72000}) { trip { trip_id} }} }`,
			vars:         hw{"stop_id": "MCAR", "service_date": "2018-05-26"},
			selector:     "stops.0.stop_times.#.trip.trip_id",
			selectExpect: []string{"3611521SAT", "3631541SAT", "3651601SAT", "3671621SAT", "3691641SAT", "3711701SAT", "3731721SAT", "3751741SAT", "3771801SAT", "3791821SAT", "3611841SAT", "3631901SAT", "2231528SAT", "2251548SAT", "2271608SAT", "2291628SAT", "2311648SAT", "2331708SAT", "2351728SAT", "2211748SAT", "2231808SAT", "2251828SAT", "2271848SAT", "2291908SAT", "4471533SAT", "4491553SAT", "4511613SAT", "4531633SAT", "4411653SAT", "4431713SAT", "4451733SAT", "4471753SAT", "4491813SAT", "4511833SAT", "4531853SAT"},
		},

		// stop external references
		{
			name: "external reference: working",
			query: `query {
				stops(where:{feed_onestop_id: "BA", stop_id:"FTVL"}) {
				  external_reference {
					target_stop_id
					target_feed_onestop_id
					target_active_stop {
					  stop_id
					}
				  }
				}
			  }`,
			f: func(t *testing.T, jj string) {
				assert.EqualValues(t, "CT", gjson.Get(jj, "stops.0.external_reference.target_feed_onestop_id").String())
				assert.EqualValues(t, "70041", gjson.Get(jj, "stops.0.external_reference.target_stop_id").String())
				assert.EqualValues(t, "70041", gjson.Get(jj, "stops.0.external_reference.target_active_stop.stop_id").String())
			},
		},
		{
			name: "external reference: broken",
			query: `query {
				stops(where:{feed_onestop_id: "BA", stop_id:"POWL"}) {
				  external_reference {
					target_stop_id
					target_feed_onestop_id
					target_active_stop {
					  stop_id
					}
				  }
				}
			  }`,
			f: func(t *testing.T, jj string) {
				assert.EqualValues(t, "CT", gjson.Get(jj, "stops.0.external_reference.target_feed_onestop_id").String())
				assert.EqualValues(t, "missing", gjson.Get(jj, "stops.0.external_reference.target_stop_id").String())
				assert.EqualValues(t, nil, gjson.Get(jj, "stops.0.external_reference.target_active_stop").Value())
			},
		},
		// stop observations
		{
			name: "observations: found",
			query: `query($fvid:Int!,$day:Date!) {
				stops(where:{feed_onestop_id: "BA", stop_id:"FTVL"}) {
					observations(where:{feed_version_id:$fvid, source:"TripUpdate", trip_start_date:$day}) {
						trip_id
						route_id
						observed_arrival_time
						observed_departure_time
					}
				}
			  }`,
			vars: hw{"fvid": stopObsFvid, "day": "2023-03-09"},
			f: func(t *testing.T, jj string) {
				assert.EqualValues(t, "test", gjson.Get(jj, "stops.0.observations.0.trip_id").String())
				assert.EqualValues(t, "03", gjson.Get(jj, "stops.0.observations.0.route_id").String())
				assert.EqualValues(t, "10:00:00", gjson.Get(jj, "stops.0.observations.0.observed_arrival_time").String())
				assert.EqualValues(t, "10:00:10", gjson.Get(jj, "stops.0.observations.0.observed_departure_time").String())
				assert.EqualValues(t, 1, len(gjson.Get(jj, "stops.0.observations").Array()))
			},
		},
		{
			name: "observations: none",
			query: `query($fvid:Int!,$day:Date!) {
				stops(where:{feed_onestop_id: "BA", stop_id:"FTVL"}) {
					observations(where:{feed_version_id:$fvid, source:"TripUpdate", trip_start_date:$day}) {
						trip_id
						route_id
						observed_arrival_time
						observed_departure_time
					}
				}
			  }`,
			vars:         hw{"fvid": stopObsFvid, "day": "2023-03-08"},
			selector:     "stops.0.observations.#.trip_id",
			selectExpect: []string{},
		},
		// serviced
		{
			name:         "stop serviced=true",
			query:        `query{stops(where:{feed_onestop_id:"EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:true}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"FUR_CREEK_RES", "BEATTY_AIRPORT", "BULLFROG", "STAGECOACH", "NADAV", "NANAA", "DADAN", "EMSI", "AMV"},
		},
		{
			name:         "stop serviced=false",
			query:        `query{stops(where:{feed_onestop_id:"EX", feed_version_sha1:"43e2278aa272879c79460582152b04e7487f0493", serviced:false}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"NOTRIPS"},
		},
		// route_type
		{
			// tampa street car
			name:         "stop route_type=0",
			query:        `query{stops(where:{served_by_route_type:0}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"6690", "6691", "6692", "6693", "6694", "6695", "6698", "6699", "6700", "6701", "7713"},
		},
		{
			name:         "stop route_type=0 empty",
			query:        `query{stops(where:{served_by_route_type:0, feed_onestop_id:"BA"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{},
		},
		{
			// BART
			name:         "stop route_type=1",
			query:        `query{stops(where:{served_by_route_type:1, feed_onestop_id:"BA"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: bartStops,
		},
		{
			name:         "stop route_type=1 empty",
			query:        `query{stops(where:{served_by_route_type:1, feed_onestop_id:"CT"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{},
		},
		{
			// Caltrain
			name:         "stop route_type=2",
			query:        `query{stops(where:{served_by_route_type:2, feed_onestop_id:"CT"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: caltrainRailStops,
		},
		{
			name:         "stop route_type=2 empty",
			query:        `query{stops(where:{served_by_route_type:2, feed_onestop_id:"BA"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{},
		},
		{
			// Caltrain
			name:         "stop route_type=3",
			query:        `query{stops(where:{served_by_route_type:3, feed_onestop_id:"CT"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: caltrainBusStops,
		},
		{
			name:         "stop route_type=3 empty",
			query:        `query{stops(where:{served_by_route_type:3, feed_onestop_id:"BA"}) { stop_id } }`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{},
		},
		// TODO: census_geographies
		// TODO: route_stop_buffer
	}
	return testcases
}

func stopResolverLocationTestcases(t *testing.T, cfg model.Config) []testcase {
	geographyId := 0
	if err := cfg.Finder.DBX().QueryRowx(`select id from tl_census_geographies where geoid = '1400000US06001402900'`).Scan(&geographyId); err != nil {
		t.Errorf("could not get geography id for test: %s", err.Error())
	}
	vars := hw{"stop_id": "MCAR"}
	featureBig := decodeGeojson(`{
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

	featureSmall := decodeGeojson(`{
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

	var testStopID int
	if err := cfg.Finder.DBX().QueryRowx(`select gtfs_stops.id from gtfs_stops join feed_states using(feed_version_id) where stop_id = $1`, "70252").Scan(&testStopID); err != nil {
		t.Errorf("could not get stop ID for test: %s", err.Error())
	}

	testcases := []testcase{
		{
			// just ensure this query completes successfully; checking coordinates is a pain and flaky.
			name:         "geometry",
			query:        `query($stop_id: String!) {  stops(where:{stop_id:$stop_id}) {geometry} }`,
			vars:         vars,
			selector:     "stops.0.geometry.type",
			selectExpect: []string{"Point"},
		},
		{
			name:         "where near 10m",
			query:        `query {stops(where:{near:{lon:-122.407974,lat:37.784471,radius:10.0}}) {stop_id onestop_id geometry}}`,
			vars:         vars,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"POWL"},
		},
		{
			name:         "where near 2000m",
			query:        `query {stops(where:{near:{lon:-122.407974,lat:37.784471,radius:2000.0}}) {stop_id onestop_id geometry}}`,
			vars:         vars,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"70011", "70012", "CIVC", "EMBR", "MONT", "POWL"},
		},
		// within features
		{
			name:         "where within polygon",
			query:        `query{stops(where:{within:{type:"Polygon",coordinates:[[[-122.396,37.8],[-122.408,37.79],[-122.393,37.778],[-122.38,37.787],[-122.396,37.8]]]}}){id stop_id}}`,
			selector:     "stops.#.stop_id",
			selectExpect: []string{"EMBR", "MONT"},
		},
		{
			name:   "where locations within_features 1",
			query:  `query($features: [Feature]) {stops(where:{location:{features:$features}}){stop_id within_features}}`,
			vars:   hw{"features": []hw{featureSmall}},
			expect: `{"stops":[{"stop_id":"12TH","within_features":["small"]}]}`,
		},
		{
			name:   "where locations within_features 2",
			query:  `query($features: [Feature]) {stops(where:{location:{features:$features}}){stop_id within_features}}`,
			vars:   hw{"features": []hw{featureSmall, featureBig}},
			expect: `{"stops":[{"stop_id":"12TH","within_features":["small","big"]},{"stop_id":"19TH","within_features":["big"]},{"stop_id":"19TH_N","within_features":["big"]}]}`,
		},
		// within geography ids
		{
			name:         "where within geography ids 1",
			query:        `query($geographyIds:[Int!]){stops(where:{location:{geography_ids:$geographyIds}}){id stop_id}}`,
			selector:     "stops.#.stop_id",
			vars:         hw{"geographyIds": []int{geographyId}},
			selectExpect: []string{"19TH", "19TH_N"},
		},
		// nearby stops
		{
			name: "nearby stops 1000m",
			query: `query($stop_id:String!, $radius:Float!) {
				stops(where: {feed_onestop_id: "BA", stop_id: $stop_id}) {
				  stop_id
				  nearby_stops(radius:$radius, limit:10) {
					stop_id
				  }
				}
			  }			  `,
			vars:         hw{"stop_id": "19TH", "radius": 1000},
			selector:     "stops.0.nearby_stops.#.stop_id",
			selectExpect: []string{"19TH", "19TH_N", "12TH"},
		},
		{
			name: "nearby stops 2000m",
			query: `query($stop_id:String!, $radius:Float!) {
				stops(where: {feed_onestop_id: "BA", stop_id: $stop_id}) {
				  stop_id
				  nearby_stops(radius:$radius, limit:10) {
					stop_id
				  }
				}
			  }			  `,
			vars:         hw{"stop_id": "19TH", "radius": 2000},
			selector:     "stops.0.nearby_stops.#.stop_id",
			selectExpect: []string{"19TH", "19TH_N", "12TH", "LAKE"},
		},
		// bbox
		{
			name: "bbox 1",
			query: `query($bbox:BoundingBox) {
				stops(where: {bbox:$bbox}) {
				  stop_id
				}
			  }			  `,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "stops.#.stop_id",
			selectExpect: []string{"19TH", "19TH_N"},
		},
		{
			name: "bbox 2",
			query: `query($bbox:BoundingBox) {
				stops(where: {bbox:$bbox}) {
				  stop_id
				}
			  }			  `,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "stops.#.stop_id",
			selectExpect: []string{},
		},
		{
			name: "bbox too large",
			query: `query($bbox:BoundingBox) {
				stops(where: {bbox:$bbox}) {
				  stop_id
				}
			  }			  `,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			expectError: true,
		},

		// this test is just for debugging purposes
		{
			name: "nearby stops check n+1 query",
			query: `query($radius:Float!) {
				stops(where: {feed_onestop_id: "BA"}) {
				  stop_id
				  nearby_stops(radius:$radius, limit:10) {
					stop_id
				  }
				}
			  }			  `,
			vars:         hw{"radius": 1000},
			selector:     "stops.#.stop_id",
			selectExpect: bartStops,
		},
		// Focus test cases
		{
			name: "focus basic: Florida focus point returns HA stops first",
			query: `query($lat:Float!, $lon:Float!) {
				stops(limit: 10, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					stop_id
					feed_version { feed { onestop_id } }
				}
			}`,
			vars:         hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			selector:     "stops.#.feed_version.feed.onestop_id",
			selectExpect: []string{"HA", "HA", "HA", "HA", "HA", "HA", "HA", "HA", "HA", "HA"},
		},
		{
			name: "focus basic: San Jose focus point returns CT stops first",
			query: `query($lat:Float!, $lon:Float!) {
				stops(limit: 10, where: {location: {focus: {lat: $lat, lon: $lon}}}) {
					stop_id
					feed_version { feed { onestop_id } }
				}
			}`,
			vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon},
			selector:     "stops.#.feed_version.feed.onestop_id",
			selectExpect: []string{"CT", "CT", "CT", "CT", "CT", "CT", "CT", "CT", "CT", "CT"},
		},
		{
			name: "focus with feed filter: HA stops only, ordered by distance",
			query: `query($lat:Float!, $lon:Float!) {
				stops(limit: 10, where: {feed_onestop_id: "HA", location: {focus: {lat: $lat, lon: $lon}}}) {
					stop_id
					geometry
				}
			}`,
			vars: hw{"lat": floridaFocus.Lat, "lon": floridaFocus.Lon},
			f: func(t *testing.T, jj string) {
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
			name: "focus with pagination: first page",
			query: `query($lat:Float!, $lon:Float!) {
				stops: stops(limit: 10, where: {feed_onestop_id: "CT", location: {focus: {lat: $lat, lon: $lon}}}) {
					id
					stop_id
				}
			}`,
			vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon},
			selector:     "stops.#.stop_id",
			selectExpect: []string{"777402", "70261", "70262", "70251", "70252", "70272", "70271", "777403", "70241", "70242"},
		},
		{
			name: "focus with pagination: second page maintains ordering",
			query: `query($lat:Float!, $lon:Float!, $after:Int!) {
				stops: stops(after: $after, limit: 10, where: {feed_onestop_id: "CT", location: {focus: {lat: $lat, lon: $lon}}}) {
					id
					stop_id
				}
			}`,
			vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon, "after": testStopID},
			selector:     "stops.#.stop_id",
			selectExpect: []string{"70272", "70271", "777403", "70241", "70242", "70282", "70281", "70232", "70231", "70291"},
		},
		{
			name: "focus with near filter: combined spatial queries",
			query: `query($lat:Float!, $lon:Float!, $radius:Float!) {
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
			vars:         hw{"lat": sanJoseFocus.Lat, "lon": sanJoseFocus.Lon, "radius": sanJoseRadiusMeters},
			selector:     "stops.#.stop_id",
			selectExpect: []string{"777402", "70261", "70262", "70251", "70252", "70272", "70271", "777403", "70241", "70242", "70282", "70281"},
		},
	}
	return testcases

}

func stopResolverCursorTestcases(t *testing.T, cfg model.Config) []testcase {
	// First 1000 stops...
	dbf := cfg.Finder
	allEnts, err := dbf.FindStops(context.Background(), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	allIds := []string{}
	for _, st := range allEnts {
		allIds = append(allIds, st.StopID.Val)
	}
	testcases := []testcase{
		{
			name:         "no cursor",
			query:        "query{stops(limit:100){feed_version{id} id stop_id}}",
			selector:     "stops.#.stop_id",
			selectExpect: allIds[:100],
		},
		{
			name:         "after 0",
			query:        "query{stops(after: 0, limit:100){feed_version{id} id stop_id}}",
			selector:     "stops.#.stop_id",
			selectExpect: allIds[:100],
		},
		{
			name:         "after 10th",
			query:        "query($after: Int!){stops(after: $after, limit:10){feed_version{id} id stop_id}}",
			vars:         hw{"after": allEnts[10].ID},
			selector:     "stops.#.stop_id",
			selectExpect: allIds[11:21],
		},
		{
			name:         "after invalid id returns no records",
			query:        "query($after: Int!){stops(after: $after, limit:10){feed_version{id} id stop_id}}",
			vars:         hw{"after": 10_000_000},
			selector:     "stops.#.stop_id",
			selectExpect: []string{},
		},
		// TODO: uncomment after schema changes
		// {
		// 	"no cursor",
		// 	"query($cursor: Cursor!){stops(after: $cursor, limit:100){feed_version{id} id stop_id}}",
		// 	hw{"cursor": 0},
		// 	``,
		// 	"stops.#.stop_id",
		// 	stopIds[:100],
		// },
	}
	return testcases
}

func stopResolverPreviousOnestopIDTestcases(t testing.TB, cfg model.Config) []testcase {
	_ = t
	_ = cfg
	testcases := []testcase{
		{
			name:         "default",
			query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { stop_id onestop_id }}`,
			vars:         hw{"osid": "s-9q9nfsxn67-fruitvale", "previous": false},
			selector:     "stops.#.onestop_id",
			selectExpect: []string{"s-9q9nfsxn67-fruitvale"},
		},
		{
			name:         "old id no result",
			query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { stop_id onestop_id }}`,
			vars:         hw{"osid": "s-9q9nfswzpg-fruitvale", "previous": false},
			selector:     "stops.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "old id no specify fv",
			query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous, feed_version_sha1:"dd7aca4a8e4c90908fd3603c097fabee75fea907"}) { stop_id onestop_id }}`,
			vars:         hw{"osid": "s-9q9nfswzpg-fruitvale", "previous": false},
			selector:     "stops.#.onestop_id",
			selectExpect: []string{"s-9q9nfswzpg-fruitvale"},
		},
		{
			name:         "use previous",
			query:        `query($osid:String!, $previous:Boolean!) { stops(where:{onestop_id:$osid, allow_previous_onestop_ids:$previous}) { stop_id onestop_id }}`,
			vars:         hw{"osid": "s-9q9nfswzpg-fruitvale", "previous": true},
			selector:     "stops.#.onestop_id",
			selectExpect: []string{"s-9q9nfswzpg-fruitvale"},
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
			name:               "license filter: share_alike_optional = yes",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  2349,
		},
		{
			name:               "license filter: share_alike_optional = no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  50,
		},
		{
			name:               "license filter: share_alike_optional = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  2706,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  2349,
		},
		{
			name:               "license filter: create_derived_product = no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  50,
		},
		{
			name:               "license filter: create_derived_product = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  2706,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  2349,
		},
		{
			name:               "license filter: commercial_use_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  50,
		},
		{
			name:               "license filter: commercial_use_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  2706,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  2349,
		},
		{
			name:               "license filter: redistribution_allowed = no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  50,
		},
		{
			name:               "license filter: redistribution_allowed = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  2706,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA"},
			selectExpectCount:  2349,
		},
		{
			name:               "license filter: use_without_attribution = no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"BA"},
			selectExpectCount:  50,
		},
		{
			name:               "license filter: use_without_attribution = exclude_no",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"CT", "HA", "ctran-flex"},
			selectExpectCount:  2706,
		},
	}
	return testcases
}

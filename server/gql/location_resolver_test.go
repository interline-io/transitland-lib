package gql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestLocationResolver(t *testing.T) {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	roseVillageLocationID := "location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"
	testcases := []testcase{
		{
			name: "locations count and ids",
			query: `query($sha1: String!) {
				feed_versions(where: {sha1: $sha1}) {
					locations {
						location_id
						stop_name
					}
				}
			}`,
			vars:     hw{"sha1": ctranFlexSha1},
			selector: "feed_versions.0.locations.#.location_id",
			selectExpect: []string{
				"location_id__11f830d0-adec-468a-a8d6-513184e476a1",
				"location_id__2a077a44-c1e9-44c6-8b26-6ece58b64db6",
				"location_id__43ca2d5b-a235-4669-a27e-371a7c528cca",
				"location_id__75e0de0d-d90c-4f15-a6cc-001f734e0f13",
				"location_id__8d41f4d3-7760-457e-94e1-6f7980cb3c20",
				"location_id__ac79ba5e-31ae-4879-a455-a053862dbe59",
				"location_id__bb80cf18-9fa7-498a-b22f-1f66eb4214a6",
				"location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59",
			},
		},
		{
			name: "location fields",
			query: `query($sha1: String!, $location_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					locations(where: {location_id: $location_id}) {
						location_id
						stop_name
						stop_desc
						zone_id
						stop_url
						geometry
						feed_version {
							sha1
						}
					}
				}
			}`,
			vars:   hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			expect: `{"feed_versions":[{"locations":[{"feed_version":{"sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c"},"geometry":{"coordinates":[[[-122.67151988787528,45.640239867113465],[-122.66531553935086,45.640196283362634],[-122.66490760228291,45.64013588505739],[-122.66454068611063,45.63997388749707],[-122.66362140070383,45.63920641403817],[-122.66308366082943,45.63894039603798],[-122.6622399397029,45.63872594572448],[-122.66111535962024,45.63880529545077],[-122.65947652977027,45.63933633170179],[-122.65945763749346,45.639370516569116],[-122.65761727409583,45.63978678396147],[-122.65428757108356,45.639080059577985],[-122.65207272399068,45.63869551462473],[-122.6478525477521,45.63848754690533],[-122.64783134700939,45.63851365039364],[-122.64181726752285,45.63803956014843],[-122.6418217505775,45.63586363794634],[-122.64013391715848,45.63583978393064],[-122.64014198381815,45.637893111129735],[-122.62982020087483,45.63733345560127],[-122.62881757914062,45.63748398884993],[-122.62685692195674,45.63817113526683],[-122.62842022396616,45.639372602559504],[-122.63527305728671,45.64300300957819],[-122.63408470248598,45.643005213585],[-122.63374920628864,45.64336927789179],[-122.63350596062409,45.643983173935034],[-122.63420358334933,45.64438849388529],[-122.63489269067372,45.644610924158805],[-122.63645793595622,45.64459415539177],[-122.63771704271298,45.64589658577323],[-122.63959433879144,45.645787752874305],[-122.63983366686479,45.64606241364155],[-122.64200696797151,45.64645619286868],[-122.64288979024167,45.64683159683269],[-122.64714144922497,45.64958971909228],[-122.64831508883982,45.650078048561625],[-122.65022520970011,45.650439272310905],[-122.65197348186929,45.65047369175858],[-122.6575081621618,45.649741330126],[-122.66760435040396,45.649901662815225],[-122.6684979109249,45.64657292165278],[-122.66861076463377,45.64615776413661],[-122.66876230752703,45.64571362197488],[-122.66885041202838,45.64543700468926],[-122.67151988787528,45.640239867113465]],[[-122.64166509361434,45.63748360138959],[-122.64069536600529,45.63748694143604],[-122.64068103505541,45.63721639702986],[-122.640303653375,45.637199696715115],[-122.64028454544183,45.63595717933418],[-122.64164120869788,45.6359905806994],[-122.64166509361434,45.63748360138959]]],"type":"Polygon"},"location_id":"location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59","stop_desc":null,"stop_name":"Rose Village","stop_url":null,"zone_id":null}]}]}`,
		},
		// Test geometry type (simpler verification than exact JSON match)
		{
			name: "location geometry type",
			query: `query($sha1: String!, $location_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					locations(where: {location_id: $location_id}) {
						geometry
					}
				}
			}`,
			vars:         hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			selector:     "feed_versions.0.locations.0.geometry.type",
			selectExpect: []string{"Polygon"},
		},
		// Test feed_onestop_id and feed_version_sha1 fields
		{
			name: "location feed metadata",
			query: `query($sha1: String!, $location_id: String!) {
				feed_versions(where: {sha1: $sha1}) {
					locations(where: {location_id: $location_id}) {
						location_id
						feed_onestop_id
						feed_version_sha1
					}
				}
			}`,
			vars:   hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			expect: `{"feed_versions":[{"locations":[{"feed_onestop_id":"ctran-flex","feed_version_sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c","location_id":"location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"}]}]}`,
		},
		// Test filter returns empty array for non-existent location
		{
			name: "location filter not found",
			query: `query($sha1: String!) {
				feed_versions(where: {sha1: $sha1}) {
					locations(where: {location_id: "nonexistent"}) {
						location_id
					}
				}
			}`,
			vars:         hw{"sha1": ctranFlexSha1},
			selector:     "feed_versions.0.locations.#.location_id",
			selectExpect: []string{},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestLocationResolver_StopTimes(t *testing.T) {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	roseVillageLocationID := "location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"
	testcases := []testcase{
		{
			name: "location stop times with trip",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1) {
							stop_sequence
							trip {
								trip_id
							}
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			f: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				if len(locs) == 0 {
					t.Fatal("expected locations")
				}
				loc := locs[0]
				assert.Equal(t, roseVillageLocationID, loc.Get("location_id").String())
				sts := loc.Get("stop_times").Array()
				if len(sts) > 0 {
					assert.NotEmpty(t, sts[0].Get("trip.trip_id").String(), "expected trip_id on stop time")
				}
			},
		},
		{
			name: "location stop times count for known location",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_name
						stop_times(limit: 200) {
							stop_sequence
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			f: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				if len(locs) == 0 {
					t.Fatal("expected locations")
				}
				loc := locs[0]
				assert.Equal(t, roseVillageLocationID, loc.Get("location_id").String())
				assert.Equal(t, "Rose Village", loc.Get("stop_name").String())
				sts := loc.Get("stop_times").Array()
				// Rose Village location has 150 stop_times in the C-TRAN flex feed
				assert.Equal(t, 150, len(sts), "expected 150 stop_times for Rose Village location")
			},
		},
		{
			name: "location stop times flex fields",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1) {
							stop_sequence
							start_pickup_drop_off_window
							end_pickup_drop_off_window
							pickup_type
							drop_off_type
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			f: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				if len(locs) == 0 {
					t.Fatal("expected locations")
				}
				sts := locs[0].Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				st := sts[0]
				// Flex stop_times have pickup/drop-off windows instead of arrival/departure times
				assert.True(t, st.Get("start_pickup_drop_off_window").Exists(), "expected start_pickup_drop_off_window")
				assert.True(t, st.Get("end_pickup_drop_off_window").Exists(), "expected end_pickup_drop_off_window")
				// Verify pickup/drop_off types are set (2=must coordinate, 1=no pickup/drop-off)
				assert.True(t, st.Get("pickup_type").Exists(), "expected pickup_type")
				assert.True(t, st.Get("drop_off_type").Exists(), "expected drop_off_type")
			},
		},
		{
			name: "location stop times with booking rules",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1) {
							stop_sequence
							pickup_booking_rule {
								booking_rule_id
								booking_type
								message
							}
							drop_off_booking_rule {
								booking_rule_id
							}
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			f: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				if len(locs) == 0 {
					t.Fatal("expected locations")
				}
				sts := locs[0].Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				st := sts[0]
				// Flex stop_times have associated booking rules
				pickupRule := st.Get("pickup_booking_rule")
				assert.True(t, pickupRule.Exists(), "expected pickup_booking_rule")
				assert.NotEmpty(t, pickupRule.Get("booking_rule_id").String(), "expected booking_rule_id")
				assert.True(t, pickupRule.Get("booking_type").Exists(), "expected booking_type")
				assert.NotEmpty(t, pickupRule.Get("message").String(), "expected booking message")
			},
		},
		{
			name: "location stop times navigates back to location",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1) {
							stop_sequence
							location {
								location_id
								stop_name
							}
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			f: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				if len(locs) == 0 {
					t.Fatal("expected locations")
				}
				sts := locs[0].Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				// The stop_time's location should point back to the same location
				stLoc := sts[0].Get("location")
				assert.Equal(t, roseVillageLocationID, stLoc.Get("location_id").String())
				assert.Equal(t, "Rose Village", stLoc.Get("stop_name").String())
			},
		},
		{
			name: "location stop times with trip details",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 5) {
							stop_sequence
							trip {
								trip_id
								trip_short_name
								route {
									route_id
									route_short_name
								}
							}
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			f: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				if len(locs) == 0 {
					t.Fatal("expected locations")
				}
				sts := locs[0].Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				// All stop_times should have valid trip and route references
				for i, st := range sts {
					trip := st.Get("trip")
					assert.NotEmpty(t, trip.Get("trip_id").String(), "stop_time[%d] expected trip_id", i)
					route := trip.Get("route")
					assert.NotEmpty(t, route.Get("route_id").String(), "stop_time[%d] expected route_id", i)
				}
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

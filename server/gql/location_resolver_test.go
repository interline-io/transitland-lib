package gql

import (
	"testing"
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
			name: "location stop times count for known location",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1000) {
							stop_sequence
						}
					}
				}
			}`,
			vars:              hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			selector:          "feed_versions.0.locations.0.stop_times.#.stop_sequence",
			selectExpectCount: 150,
		},
		{
			name: "location stop times flex fields",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1000) {
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
			sel: []testcaseSelector{
				{
					selector:          "feed_versions.0.locations.0.stop_times.#.stop_sequence",
					expectCount:       150,
					expectUniqueCount: 2,
				},
				{
					selector:          "feed_versions.0.locations.0.stop_times.#.start_pickup_drop_off_window",
					expectCount:       150,
					expectUniqueCount: 2,
				},
				{
					selector:          "feed_versions.0.locations.0.stop_times.#.end_pickup_drop_off_window",
					expectCount:       150,
					expectUniqueCount: 2,
				},
				{
					selector:          "feed_versions.0.locations.0.stop_times.#.pickup_type",
					expectCount:       150,
					expectUniqueCount: 2,
				},
				{
					selector:          "feed_versions.0.locations.0.stop_times.#.drop_off_type",
					expectCount:       150,
					expectUniqueCount: 2,
				},
			},
		},
		{
			name: "location stop times navigates back to location",
			query: `query($sha1: String!, $location_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1000) {
							location {
								location_id
							}
						}
					}
				}
			}`,
			vars:               hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID},
			selector:           "feed_versions.0.locations.0.stop_times.#.location.location_id",
			selectExpectUnique: []string{roseVillageLocationID},
		},
		{
			name: "location stop times with service date",
			query: `query($sha1: String!, $location_id: String, $service_date: Date) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1000, where: {service_date: $service_date}) {
							trip {
								trip_id
							}
						}
					}
				}
			}`,
			vars:     hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID, "service_date": "2025-12-08"},
			selector: "feed_versions.0.locations.0.stop_times.#.trip.trip_id",
			selectExpectUnique: []string{
				"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_to-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>",
				"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_to-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>",
				"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_to-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>",
				"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_to-<b5f50364-07f6-46f3-aa8c-f50aefaecb53>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>",
				"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<b5f50364-07f6-46f3-aa8c-f50aefaecb53>_to-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_si-<MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a>",
			},
		},
		{
			name: "location stop times with future service date returns empty",
			query: `query($sha1: String!, $location_id: String, $service_date: Date) {
				feed_versions(where:{sha1:$sha1}) {
					locations(where:{location_id:$location_id}) {
						location_id
						stop_times(limit: 1000, where: {service_date: $service_date}) {
							trip {
								trip_id
							}
						}
					}
				}
			}`,
			vars:         hw{"sha1": ctranFlexSha1, "location_id": roseVillageLocationID, "service_date": "2099-01-01"},
			selector:     "feed_versions.0.locations.0.stop_times.#.trip.trip_id",
			selectExpect: []string{},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

package gql

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestLocationGroupResolver(t *testing.T) {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	fairgroundsLocationGroupID := "location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"
	vaMedicalCenterLocationGroupID := "location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"
	testcases := []testcase{
		{
			name: "location groups basic fields",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where: {sha1: $sha1}) {
					location_groups(where: {location_group_id: $location_group_id}, limit: 1) {
						location_group_id
						location_group_name
						feed_version {
							sha1
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": fairgroundsLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				lg := lgs[0]
				assert.Equal(t, fairgroundsLocationGroupID, lg.Get("location_group_id").String())
				assert.Equal(t, "Clark County Fairgroun...", lg.Get("location_group_name").String())
				assert.Equal(t, ctranFlexSha1, lg.Get("feed_version.sha1").String())
			},
		},
		{
			name: "location group with stops",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where: {sha1: $sha1}) {
					location_groups(where: {location_group_id: $location_group_id}) {
						location_group_id
						stops {
							stop_id
							stop_name
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": fairgroundsLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				lg := lgs[0]
				stops := lg.Get("stops").Array()
				// Fairgrounds location group has 12 stops
				assert.Equal(t, 12, len(stops), "expected 12 stops for Fairgrounds location group")
				// Verify stops have expected fields
				for _, stop := range stops {
					assert.NotEmpty(t, stop.Get("stop_id").String(), "expected stop_id")
				}
			},
		},
		{
			name: "location group stops count for VA Medical Center",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where: {sha1: $sha1}) {
					location_groups(where: {location_group_id: $location_group_id}) {
						location_group_id
						location_group_name
						stops {
							stop_id
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				lg := lgs[0]
				assert.Equal(t, vaMedicalCenterLocationGroupID, lg.Get("location_group_id").String())
				assert.Equal(t, "VA Medical Center (fixed route stop)", lg.Get("location_group_name").String())
				stops := lg.Get("stops").Array()
				// VA Medical Center location group has 7 stops
				assert.Equal(t, 7, len(stops), "expected 7 stops for VA Medical Center location group")
			},
		},
	}
	c, _ := newTestClient(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			queryTestcase(t, c, tc)
		})
	}
}

func TestLocationGroupResolver_StopTimes(t *testing.T) {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	vaMedicalCenterLocationGroupID := "location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"
	testcases := []testcase{
		{
			name: "location group stop times basic",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
						location_group_name
						stop_times(limit: 1) {
							stop_sequence
							trip {
								trip_id
							}
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				lg := lgs[0]
				assert.Equal(t, vaMedicalCenterLocationGroupID, lg.Get("location_group_id").String())
				assert.Equal(t, "VA Medical Center (fixed route stop)", lg.Get("location_group_name").String())
				sts := lg.Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				assert.NotEmpty(t, sts[0].Get("trip.trip_id").String(), "expected trip_id on stop time")
			},
		},
		{
			name: "location group stop times count",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
						stop_times(limit: 200) {
							stop_sequence
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				sts := lgs[0].Get("stop_times").Array()
				// VA Medical Center location group has 150 stop_times in the C-TRAN flex feed
				assert.Equal(t, 150, len(sts), "expected 150 stop_times for VA Medical Center location group")
			},
		},
		{
			name: "location group stop times flex fields",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
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
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				sts := lgs[0].Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				st := sts[0]
				// Flex stop_times have pickup/drop-off windows instead of arrival/departure times
				assert.True(t, st.Get("start_pickup_drop_off_window").Exists(), "expected start_pickup_drop_off_window")
				assert.True(t, st.Get("end_pickup_drop_off_window").Exists(), "expected end_pickup_drop_off_window")
				assert.True(t, st.Get("pickup_type").Exists(), "expected pickup_type")
				assert.True(t, st.Get("drop_off_type").Exists(), "expected drop_off_type")
			},
		},
		{
			name: "location group stop times with booking rule",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
						stop_times(limit: 1) {
							stop_sequence
							pickup_booking_rule {
								booking_rule_id
								booking_type
							}
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				sts := lgs[0].Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				pickupRule := sts[0].Get("pickup_booking_rule")
				assert.True(t, pickupRule.Exists(), "expected pickup_booking_rule")
				assert.NotEmpty(t, pickupRule.Get("booking_rule_id").String(), "expected booking_rule_id")
			},
		},
		{
			name: "location group stop times navigates back to location_group",
			query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
						stop_times(limit: 1) {
							stop_sequence
							location_group {
								location_group_id
								location_group_name
							}
						}
					}
				}
			}`,
			vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			f: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				if len(lgs) == 0 {
					t.Fatal("expected location_groups")
				}
				sts := lgs[0].Get("stop_times").Array()
				if len(sts) == 0 {
					t.Fatal("expected stop_times")
				}
				// The stop_time's location_group should point back to the same location_group
				stLg := sts[0].Get("location_group")
				assert.Equal(t, vaMedicalCenterLocationGroupID, stLg.Get("location_group_id").String())
				assert.Equal(t, "VA Medical Center (fixed route stop)", stLg.Get("location_group_name").String())
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func locationGroupResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	fairgroundsLocationGroupID := "location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"
	testcases := []testcase{
		{
			Name: "location groups - returns multiple location groups",
			Query: `query($sha1: String!) {
				feed_versions(where: {sha1: $sha1}) {
					location_groups(limit: 1000) {
						location_group_id
					}
				}
			}`,
			Vars:     hw{"sha1": ctranFlexSha1},
			Selector: "feed_versions.0.location_groups.#.location_group_id",
			SelectExpect: []string{
				"location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a",
				"location_group_id__331e6ae2-66b0-4d01-aef9-f9ef35328981",
				"location_group_id__58bcb950-3baa-41ce-a36c-ae6a1a36f97a",
				"location_group_id__941932c8-20bb-4f79-b167-d9bec90cff9f",
				"location_group_id__af8ed7b5-0db2-4872-b8e1-c9ca6922d39b",
				"location_group_id__b5f50364-07f6-46f3-aa8c-f50aefaecb53",
				"location_group_id__b781e1a2-5f6b-4955-8c97-f91e938589b4",
				"location_group_id__bfe1744d-c541-4b12-9a0a-5daa8b0524ba",
				"location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e",
				"location_group_id__f772298b-6506-4272-b032-1d13296ec3bd",
			},
		},
		// Test feed metadata fields
		{
			Name: "location group feed metadata",
			Query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where: {sha1: $sha1}) {
					location_groups(where: {location_group_id: $location_group_id}) {
						location_group_id
						feed_onestop_id
						feed_version_sha1
					}
				}
			}`,
			Vars:   hw{"sha1": ctranFlexSha1, "location_group_id": fairgroundsLocationGroupID},
			Expect: `{"feed_versions":[{"location_groups":[{"feed_onestop_id":"ctran-flex","feed_version_sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c","location_group_id":"location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"}]}]}`,
		},
		// Test empty result for non-existent location_group_id
		{
			Name: "location group filter not found",
			Query: `query($sha1: String!) {
				feed_versions(where: {sha1: $sha1}) {
					location_groups(where: {location_group_id: "nonexistent"}) {
						location_group_id
					}
				}
			}`,
			Vars:         hw{"sha1": ctranFlexSha1},
			Selector:     "feed_versions.0.location_groups.#.location_group_id",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func locationGroupResolverStopTimesTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	vaMedicalCenterLocationGroupID := "location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"
	testcases := []testcase{
		{
			Name: "location group stop times basic",
			Query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
						location_group_name
						stop_times(limit: 1000) {
							trip {
								trip_id
							}
						}
					}
				}
			}`,
			Vars:                    hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			Selector:                "feed_versions.0.location_groups.0.stop_times.#.trip.trip_id",
			SelectExpectCount:       150,
			SelectExpectUniqueCount: 125,
			SelectExpectContains: []string{
				"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_to-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_si-<xxxxxxx_20220107_20320522__080000_180000__080000_180000__p_20250901>",
				"trip_id__ri-<2bc6804f-9e24-4b91-8947-c73a2363e7b6>_from-<db7489d3-7478-4d3b-a47f-60c58e3fed6e>_to-<c7400cc8-959c-42c8-991f-8f601ec9ea59>_si-<xxxxxxx_20220107_20320522__080000_180000__080000_180000__p_20240704>",
			},
		},
		{
			Name: "location group stop times flex fields",
			Query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
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
			Vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			Sel: []testcaseSelector{
				{
					Selector:          "feed_versions.0.location_groups.0.stop_times.#.stop_sequence",
					ExpectCount:       150,
					ExpectUniqueCount: 2,
				},
				{
					Selector:          "feed_versions.0.location_groups.0.stop_times.#.start_pickup_drop_off_window",
					ExpectCount:       150,
					ExpectUniqueCount: 5,
				},
				{
					Selector:          "feed_versions.0.location_groups.0.stop_times.#.end_pickup_drop_off_window",
					ExpectCount:       150,
					ExpectUniqueCount: 4,
				},
				{
					Selector:          "feed_versions.0.location_groups.0.stop_times.#.pickup_type",
					ExpectCount:       150,
					ExpectUniqueCount: 2,
				},
				{
					Selector:          "feed_versions.0.location_groups.0.stop_times.#.drop_off_type",
					ExpectCount:       150,
					ExpectUniqueCount: 2,
				},
			},
		},
		{
			Name: "location group stop times with booking rule",
			Query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
						stop_times(limit: 1000) {
							pickup_booking_rule {
								booking_rule_id
								booking_type
							}
						}
					}
				}
			}`,
			Vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			Sel: []testcaseSelector{
				{
					Selector:          "feed_versions.0.location_groups.0.stop_times.#.pickup_booking_rule.booking_rule_id",
					ExpectCount:       150,
					ExpectUniqueCount: 25,
				},
				{
					Selector:          "feed_versions.0.location_groups.0.stop_times.#.pickup_booking_rule.booking_type",
					ExpectCount:       150,
					ExpectUniqueCount: 1,
				},
			},
		},
		{
			Name: "location group stop times navigates back to location_group",
			Query: `query($sha1: String!, $location_group_id: String) {
				feed_versions(where:{sha1:$sha1}) {
					location_groups(where:{location_group_id:$location_group_id}) {
						location_group_id
						stop_times(limit: 1000) {
							location_group {
								location_group_id
								location_group_name
							}
						}
					}
				}
			}`,
			Vars: hw{"sha1": ctranFlexSha1, "location_group_id": vaMedicalCenterLocationGroupID},
			Sel: []testcaseSelector{
				{
					Selector:     "feed_versions.0.location_groups.0.stop_times.#.location_group.location_group_id",
					ExpectCount:  150,
					ExpectUnique: []string{vaMedicalCenterLocationGroupID},
				},
				{
					Selector:     "feed_versions.0.location_groups.0.stop_times.#.location_group.location_group_name",
					ExpectCount:  150,
					ExpectUnique: []string{"VA Medical Center (fixed route stop)"},
				},
			},
		},
	}
	return testcases
}

func locationGroupResolverStopsTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	fairgroundsLocationGroupID := "location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"
	testcases := []testcase{
		{
			Name: "location group with stops",
			Query: `query($sha1: String!, $location_group_id: String) {
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
			Vars:     hw{"sha1": ctranFlexSha1, "location_group_id": fairgroundsLocationGroupID},
			Selector: "feed_versions.0.location_groups.0.stops.#.stop_id",
			SelectExpect: []string{
				"stop_id__756c0e65-32d2-4e32-a6b7-a15c3c22e6cf",
				"stop_id__2e44e463-310b-4069-a709-fa0eb8f73ba9",
				"stop_id__b9138aff-d962-4de4-aa6e-1a2d0fd11dcb",
				"stop_id__f236b4c8-6655-4232-9482-22d8aea3b2cc",
				"stop_id__a3462850-8cc7-4b2a-8d82-90fc45ec64d9",
				"stop_id__3e7a8119-c692-4fb3-9c3f-53d56168a4f1",
				"stop_id__57b40253-b681-472b-9acb-755244b182d2",
				"stop_id__0962bc45-0653-49b3-9937-793284c88ce9",
				"stop_id__5e838e67-ba88-4338-b2cb-72504e2a8830",
				"stop_id__3ef5ab4b-bdb8-4d8b-b4d1-4c8b10388e8d",
				"stop_id__31934462-e8f1-41b5-8275-c8313c96b1e3",
				"stop_id__bd5a1d72-c3f6-4024-9ee3-c08779870647",
			},
		},
	}
	return testcases
}

func locationGroupResolverFeedVersionTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	fairgroundsLocationGroupID := "location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a"
	testcases := []testcase{
		{
			Name: "location groups basic fields",
			Query: `query($sha1: String!, $location_group_id: String) {
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
			Vars:   hw{"sha1": ctranFlexSha1, "location_group_id": fairgroundsLocationGroupID},
			Expect: `{"feed_versions":[{"location_groups":[{"feed_version":{"sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c"},"location_group_id":"location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a","location_group_name":"Clark County Fairgroun..."}]}]}`,
		},
	}
	return testcases
}

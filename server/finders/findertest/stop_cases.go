package findertest

// Cases duplicated from server/gql/stop_resolver_test.go.

func stopResolverLocationGroupsTestcases() []testcase {
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
	return testcases
}

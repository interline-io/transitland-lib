package findertest

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

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
	}
	return testcases
}

func stopResolverLicenseTestcases(t *testing.T, cfg model.Config) []testcase {
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
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  4524,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4881,
		},
		// license: create_derived_product
		{
			name:               "license filter: create_derived_product = yes",
			query:              q,
			vars:               hw{"lic": hw{"create_derived_product": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  4524,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4881,
		},
		// license: commercial_use_allowed
		{
			name:               "license filter: commercial_use_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  4524,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4881,
		},
		// license: redistribution_allowed
		{
			name:               "license filter: redistribution_allowed = yes",
			query:              q,
			vars:               hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  4524,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4881,
		},
		// license: use_without_attribution
		{
			name:               "license filter: use_without_attribution = yes",
			query:              q,
			vars:               hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:           "stops.#.feed_version.feed.onestop_id",
			selectExpectUnique: []string{"HA", "WMATA"},
			selectExpectCount:  4524,
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
			selectExpectUnique: []string{"CT", "HA", "WMATA", "ctran-flex"},
			selectExpectCount:  4881,
		},
	}
	return testcases
}

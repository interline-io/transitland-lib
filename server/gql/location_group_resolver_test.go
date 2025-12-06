package gql

import (
	"testing"
)

func TestLocationGroupResolver(t *testing.T) {
	testcases := []testcase{
		{
			name: "location groups for ctran",
			query: `query {
				feed_versions(where: {sha1: "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"}) {
					location_groups(limit: 1) {
						location_group_id
						location_group_name
						feed_version {
							sha1
						}
						stops {
							stop_id
						}
					}
				}
			}`,
			expect: `{"feed_versions":[{"location_groups":[{"feed_version":{"sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c"},"location_group_id":"location_group_id__138b146e-30ff-4837-baf8-bd75b47bac6a","location_group_name":"Clark County Fairgroun...","stops":[]}]}]}`,
		},
	}
	c, _ := newTestClient(t)
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			queryTestcase(t, c, tc)
		})
	}
}

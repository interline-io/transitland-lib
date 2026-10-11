package gql

import (
	"testing"
)

func TestFeedVersionResolver_LocationGroupStops(t *testing.T) {
	flexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	q := `query($sha1:String!, $limit:Int) {
		feed_versions(where:{sha1:$sha1}) {
			location_group_stops(limit:$limit) { location_group { location_group_id } stop { stop_id } }
		}
	}`
	testcases := []testcase{
		{
			name:  "every location group stop in the feed version",
			query: q,
			vars:  hw{"sha1": flexSha1, "limit": 1000},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.location_group_stops.#.stop.stop_id", expectCount: 293},
				{selector: "feed_versions.0.location_group_stops.#.location_group.location_group_id", expectUniqueCount: 10},
			},
		},
		{
			name:   "first record",
			query:  q,
			vars:   hw{"sha1": flexSha1, "limit": 1},
			expect: `{"feed_versions":[{"location_group_stops":[{"location_group":{"location_group_id":"location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"},"stop":{"stop_id":"stop_id__09a24d02-a63c-4fa4-b625-c9ba13efd2c0"}}]}]}`,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

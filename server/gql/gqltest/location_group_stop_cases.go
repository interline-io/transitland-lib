package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func feedVersionResolverLocationGroupStopsTestcases(t testing.TB, cfg model.Config) []testcase {
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
			name:   "feed version fields",
			query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { location_group_stops(limit:1) { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			vars:   hw{"sha1": flexSha1},
			expect: `{"feed_versions":[{"location_group_stops":[{"feed_version_sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c","feed_onestop_id":"ctran-flex","feed_version":{"sha1":"e8bc76c3c8602cad745f41a49ed5c5627ad6904c"}}]}]}`,
		},
		{
			name:   "first record",
			query:  q,
			vars:   hw{"sha1": flexSha1, "limit": 1},
			expect: `{"feed_versions":[{"location_group_stops":[{"location_group":{"location_group_id":"location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"},"stop":{"stop_id":"stop_id__09a24d02-a63c-4fa4-b625-c9ba13efd2c0"}}]}]}`,
		},
	}
	return testcases
}

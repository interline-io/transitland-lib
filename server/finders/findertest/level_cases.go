package findertest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

// Cases duplicated from server/gql/level_resolver_test.go.

func levelResolverTestcases(t *testing.T, cfg model.Config) []testcase {
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	testcases := []testcase{
		{
			name:   "level by stop",
			query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:"NODE_C08_FG_PAID"}) { level { level_id level_name level_index } } } }`,
			vars:   hw{"sha1": wmataSha1},
			expect: `{"feed_versions":[{"stops":[{"level":{"level_id":"C08_L1","level_index":-1,"level_name":"Mezzanine"}}]}]}`,
		},
		{
			name:   "stop level feed version fields",
			query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:"NODE_C08_FG_PAID"}) { level { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } } }`,
			vars:   hw{"sha1": wmataSha1},
			expect: `{"feed_versions":[{"stops":[{"level":{"feed_version_sha1":"148d00724546e1526d5d84d1f9d095df24f6517c","feed_onestop_id":"WMATA","feed_version":{"sha1":"148d00724546e1526d5d84d1f9d095df24f6517c"}}}]}]}`,
		},
		{
			name:  "stops by level",
			query: `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { levels(where:{level_id:"C08_L1"}) { stops { stop_id } } } }`,
			vars:  hw{"sha1": wmataSha1},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.levels.0.stops.#.stop_id", expectCount: 14},
				{selector: "feed_versions.0.levels.0.stops.#.stop_id", expectContains: []string{"NODE_C08_FG_PAID", "NODE_C08_FG_UNPAID"}},
			},
		},
	}
	return testcases
}

func feedVersionResolverLevelsTestcases(t *testing.T, cfg model.Config) []testcase {
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	q := `query($sha1:String!, $limit:Int, $where:LevelFilter) {
		feed_versions(where:{sha1:$sha1}) {
			levels(limit:$limit, where:$where) { level_id level_name level_index }
		}
	}`
	testcases := []testcase{
		{
			name:              "every level in the feed version",
			query:             q,
			vars:              hw{"sha1": wmataSha1, "limit": 1000},
			selector:          "feed_versions.0.levels.#.level_id",
			selectExpectCount: 277,
		},
		{
			name:     "station levels",
			query:    q,
			vars:     hw{"sha1": wmataSha1, "limit": 1000},
			selector: `feed_versions.0.levels.#(level_id%"C08_*")#.level_name`,
			selectExpect: []string{
				"Street", "Mezzanine", "Platform",
			},
		},
		{
			name:   "level_id",
			query:  q,
			vars:   hw{"sha1": wmataSha1, "where": hw{"level_id": "C08_L2"}},
			expect: `{"feed_versions":[{"levels":[{"level_id":"C08_L2","level_index":-2,"level_name":"Platform"}]}]}`,
		},
		{
			name:   "unknown level_id",
			query:  q,
			vars:   hw{"sha1": wmataSha1, "where": hw{"level_id": "missing"}},
			expect: `{"feed_versions":[{"levels":[]}]}`,
		},
		{
			name:   "feed version fields",
			query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { levels(limit:1) { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			vars:   hw{"sha1": wmataSha1},
			expect: `{"feed_versions":[{"levels":[{"feed_version_sha1":"148d00724546e1526d5d84d1f9d095df24f6517c","feed_onestop_id":"WMATA","feed_version":{"sha1":"148d00724546e1526d5d84d1f9d095df24f6517c"}}]}]}`,
		},
		{
			name:   "feed version without levels",
			query:  q,
			vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			expect: `{"feed_versions":[{"levels":[]}]}`,
		},
	}
	return testcases
}

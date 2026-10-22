package gqltest

import (
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

// WMATA Rail's Pentagon City station (STN_C08) has three levels: street
// (C08_L0), mezzanine (C08_L1) and platform (C08_L2).

func levelResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	testcases := []testcase{
		{
			Name:   "level by stop",
			Query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:"NODE_C08_FG_PAID"}) { level { level_id level_name level_index } } } }`,
			Vars:   hw{"sha1": wmataSha1},
			Expect: `{"feed_versions":[{"stops":[{"level":{"level_id":"C08_L1","level_index":-1,"level_name":"Mezzanine"}}]}]}`,
		},
		{
			Name:   "stop level feed version fields",
			Query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:"NODE_C08_FG_PAID"}) { level { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } } }`,
			Vars:   hw{"sha1": wmataSha1},
			Expect: `{"feed_versions":[{"stops":[{"level":{"feed_version_sha1":"148d00724546e1526d5d84d1f9d095df24f6517c","feed_onestop_id":"WMATA","feed_version":{"sha1":"148d00724546e1526d5d84d1f9d095df24f6517c"}}}]}]}`,
		},
		{
			Name:  "stops by level",
			Query: `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { levels(where:{level_id:"C08_L1"}) { stops { stop_id } } } }`,
			Vars:  hw{"sha1": wmataSha1},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.levels.0.stops.#.stop_id", ExpectCount: 14},
				{Selector: "feed_versions.0.levels.0.stops.#.stop_id", ExpectContains: []string{"NODE_C08_FG_PAID", "NODE_C08_FG_UNPAID"}},
			},
		},
	}
	return testcases
}

func feedVersionResolverLevelsTestcases(t testing.TB, cfg model.Config) []testcase {
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	q := `query($sha1:String!, $limit:Int, $where:LevelFilter) {
		feed_versions(where:{sha1:$sha1}) {
			levels(limit:$limit, where:$where) { level_id level_name level_index }
		}
	}`
	testcases := []testcase{
		{
			Name:              "every level in the feed version",
			Query:             q,
			Vars:              hw{"sha1": wmataSha1, "limit": 1000},
			Selector:          "feed_versions.0.levels.#.level_id",
			SelectExpectCount: 277,
		},
		{
			Name:     "station levels",
			Query:    q,
			Vars:     hw{"sha1": wmataSha1, "limit": 1000},
			Selector: `feed_versions.0.levels.#(level_id%"C08_*")#.level_name`,
			SelectExpect: []string{
				"Street", "Mezzanine", "Platform",
			},
		},
		{
			Name:   "level_id",
			Query:  q,
			Vars:   hw{"sha1": wmataSha1, "where": hw{"level_id": "C08_L2"}},
			Expect: `{"feed_versions":[{"levels":[{"level_id":"C08_L2","level_index":-2,"level_name":"Platform"}]}]}`,
		},
		{
			Name:   "unknown level_id",
			Query:  q,
			Vars:   hw{"sha1": wmataSha1, "where": hw{"level_id": "missing"}},
			Expect: `{"feed_versions":[{"levels":[]}]}`,
		},
		{
			Name:   "feed version fields",
			Query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { levels(limit:1) { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			Vars:   hw{"sha1": wmataSha1},
			Expect: `{"feed_versions":[{"levels":[{"feed_version_sha1":"148d00724546e1526d5d84d1f9d095df24f6517c","feed_onestop_id":"WMATA","feed_version":{"sha1":"148d00724546e1526d5d84d1f9d095df24f6517c"}}]}]}`,
		},
		{
			Name:   "feed version without levels",
			Query:  q,
			Vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			Expect: `{"feed_versions":[{"levels":[]}]}`,
		},
	}
	return testcases
}

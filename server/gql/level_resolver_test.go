package gql

import (
	"testing"
)

// WMATA Rail's Pentagon City station (STN_C08) has three levels: street
// (C08_L0), mezzanine (C08_L1) and platform (C08_L2).

func TestLevelResolver(t *testing.T) {
	c, _ := newTestClient(t)
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	testcases := []testcase{
		{
			name:   "level by stop",
			query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:"NODE_C08_FG_PAID"}) { level { level_id level_name level_index } } } }`,
			vars:   hw{"sha1": wmataSha1},
			expect: `{"feed_versions":[{"stops":[{"level":{"level_id":"C08_L1","level_index":-1,"level_name":"Mezzanine"}}]}]}`,
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
	queryTestcases(t, c, testcases)
}

func TestFeedVersionResolver_Levels(t *testing.T) {
	c, _ := newTestClient(t)
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
			name:   "feed version without levels",
			query:  q,
			vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			expect: `{"feed_versions":[{"levels":[]}]}`,
		},
	}
	queryTestcases(t, c, testcases)
}

package gql

import (
	"testing"
)

func TestFeedVersionResolver_Segments(t *testing.T) {
	testcases := []testcase{
		{
			name:         "two segments",
			query:        `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { segments { id way_id } }}`,
			vars:         hw{"sha1": "c969427f56d3a645195dd8365cde6d7feae7e99b"},
			selector:     "feed_versions.0.segments.#.way_id",
			selectExpect: []string{"645693994", "90865590"},
		},
		{
			name:     "geometries",
			query:    `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { segments { id way_id geometry } }}`,
			vars:     hw{"sha1": "c969427f56d3a645195dd8365cde6d7feae7e99b"},
			selector: "feed_versions.0.segments.#.geometry",
			selectExpect: []string{
				`{"coordinates":[[-82.458062,27.954493],[-82.458044,27.954442]],"type":"LineString"}`,
				`{"coordinates":[[-82.437785,28.058093],[-82.438163,28.058095],[-82.438219,28.058091],[-82.438277,28.058079],[-82.43833,28.058061],[-82.438375,28.058036],[-82.438415,28.058002],[-82.438449,28.057959],[-82.438476,28.057911],[-82.438493,28.05786],[-82.438501,28.057806],[-82.438504,28.057755],[-82.438504,28.057713],[-82.438504,28.057535]],"type":"LineString"}`,
			},
		},
		{
			name:         "segment to patterns",
			query:        `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { segments { id way_id segment_patterns { stop_pattern_id } } }}`,
			vars:         hw{"sha1": "c969427f56d3a645195dd8365cde6d7feae7e99b"},
			selector:     "feed_versions.0.segments.#.segment_patterns.#.stop_pattern_id",
			selectExpect: []string{"[42,40]", "[42]"},
		},
		{
			name:         "patterns to routes",
			query:        `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { segments { id way_id segment_patterns { stop_pattern_id route { route_id } } } }}`,
			vars:         hw{"sha1": "c969427f56d3a645195dd8365cde6d7feae7e99b"},
			selector:     "feed_versions.0.segments.#.segment_patterns.#.route.route_id",
			selectExpect: []string{`["12","19"]`, `["12"]`},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

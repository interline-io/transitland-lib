package gqltest

import (
	"context"
	"slices"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// WMATA Rail provides a station-modeled GTFS feed with pathways and levels.
// Most assertions are anchored on stops in the Pentagon City station (STN_C08),
// which has well-defined platform/entrance/node topology and a small enough
// pathway set to enumerate exactly.

func pathwayResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	// NODE_C08_FG_PAID = Pentagon City fare-gates, paid side.
	// Outgoing pathways: 1 exit gate (mode 7) + 8 walkways/escalators (mode 1).
	// Incoming pathway: 1 fare gate (mode 6) from NODE_C08_FG_UNPAID.
	fgPaid := "NODE_C08_FG_PAID"

	testcases := []testcase{
		{
			// All outgoing pathways from a known stop are returned.
			Name: "pathways_from_stop returns expected pathway_ids",
			Query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop { pathway_id }
					}
				}
			}`,
			Vars:     hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Selector: "feed_versions.0.stops.0.pathways_from_stop.#.pathway_id",
			SelectExpect: []string{
				"C08_137121", "C08_137122", "C08_137125", "C08_137128", "C08_137131",
				"C08_137134", "C08_137137", "C08_137140", "C08_137143",
			},
		},
		{
			// Incoming pathways: NODE_C08_FG_PAID has exactly one entry pathway.
			Name: "pathways_to_stop returns expected pathway_ids",
			Query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_to_stop { pathway_id }
					}
				}
			}`,
			Vars:         hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Selector:     "feed_versions.0.stops.0.pathways_to_stop.#.pathway_id",
			SelectExpect: []string{"C08_137120"},
		},
		{
			// limit is honored by the resolver.
			Name: "pathways_from_stop honors limit",
			Query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop(limit: 3) { pathway_id }
					}
				}
			}`,
			Vars:              hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Selector:          "feed_versions.0.stops.0.pathways_from_stop.#.pathway_id",
			SelectExpectCount: 3,
		},
		{
			// Full pathway field set on a one-way exit gate (pathway_mode=7).
			Name: "pathway fields - one-way exit gate",
			Query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop {
							pathway_id pathway_mode is_bidirectional traversal_time
							from_stop { stop_id }
							to_stop { stop_id }
						}
					}
				}
			}`,
			Vars: hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Sel: []testcaseSelector{
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").pathway_mode`, Expect: []string{"7"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").is_bidirectional`, Expect: []string{"0"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").traversal_time`, Expect: []string{"5"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").from_stop.stop_id`, Expect: []string{"NODE_C08_FG_PAID"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").to_stop.stop_id`, Expect: []string{"NODE_C08_FG_UNPAID"}},
			},
		},
		{
			// Bidirectional walkway: pathway_mode=1, is_bidirectional=1.
			Name: "pathway fields - bidirectional walkway",
			Query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop {
							pathway_id pathway_mode is_bidirectional traversal_time
							to_stop { stop_id }
						}
					}
				}
			}`,
			Vars: hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Sel: []testcaseSelector{
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").pathway_mode`, Expect: []string{"1"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").is_bidirectional`, Expect: []string{"1"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").traversal_time`, Expect: []string{"7"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").to_stop.stop_id`, Expect: []string{"NODE_C08_M_ELE1_TP"}},
			},
		},
		{
			// Stop on level C08_L1 (Mezzanine, level_index -1) resolves the level.
			Name:   "stop level resolves",
			Query:  `query($sha1:String!,$stop_id:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:$stop_id}) { level { level_id level_name level_index } } } }`,
			Vars:   hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Expect: `{"feed_versions":[{"stops":[{"level":{"level_id":"C08_L1","level_index":-1,"level_name":"Mezzanine"}}]}]}`,
		},
		{
			// Stops without a level resolve level=null (the parent station has no level).
			Name:   "stop without level resolves null",
			Query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:"STN_C08"}) { level { level_id } } } }`,
			Vars:   hw{"sha1": wmataSha1},
			Expect: `{"feed_versions":[{"stops":[{"level":null}]}]}`,
		},
		{
			// parent station resolves on a child stop.
			Name:   "stop parent station resolves",
			Query:  `query($sha1:String!,$stop_id:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:$stop_id}) { parent { stop_id stop_name location_type } } } }`,
			Vars:   hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Expect: `{"feed_versions":[{"stops":[{"parent":{"location_type":1,"stop_id":"STN_C08","stop_name":"Pentagon City"}}]}]}`,
		},
		{
			// A station's children include platforms, entrances, and fare-gate nodes.
			Name: "station children include platforms and entrances",
			Query: `query($sha1:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:"STN_C08"}) {
						children { stop_id }
					}
				}
			}`,
			Vars:     hw{"sha1": wmataSha1},
			Selector: "feed_versions.0.stops.0.children.#.stop_id",
			SelectExpectContains: []string{
				"PF_C08_1", "PF_C08_2",
				"ENT_C08_E", "ENT_C08_W",
				"NODE_C08_FG_PAID", "NODE_C08_FG_UNPAID",
			},
		},
		{
			// from_stop and to_stop on a Pathway resolve to full Stop objects,
			// including nested level resolution.
			Name: "pathway from_stop/to_stop resolve full Stop objects",
			Query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop {
							pathway_id
							from_stop { stop_id location_type level { level_id } }
							to_stop   { stop_id location_type }
						}
					}
				}
			}`,
			Vars: hw{"sha1": wmataSha1, "stop_id": fgPaid},
			Sel: []testcaseSelector{
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").from_stop.stop_id`, Expect: []string{"NODE_C08_FG_PAID"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").from_stop.location_type`, Expect: []string{"3"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").from_stop.level.level_id`, Expect: []string{"C08_L1"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").to_stop.stop_id`, Expect: []string{"NODE_C08_M_ELE1_TP"}},
				{Selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").to_stop.location_type`, Expect: []string{"3"}},
			},
		},
	}
	return testcases
}

func feedVersionResolverPathwaysTestcases(t testing.TB, cfg model.Config) []testcase {
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	q := `query($sha1:String!, $limit:Int, $where:PathwayFilter) {
		feed_versions(where:{sha1:$sha1}) {
			pathways(limit:$limit, where:$where) {
				pathway_id pathway_mode is_bidirectional traversal_time
				from_stop { stop_id }
				to_stop { stop_id }
			}
		}
	}`
	testcases := []testcase{
		{
			Name:              "every pathway in the feed version",
			Query:             q,
			Vars:              hw{"sha1": wmataSha1, "limit": 10_000},
			Selector:          "feed_versions.0.pathways.#.pathway_id",
			SelectExpectCount: 2958,
		},
		{
			Name:              "default limit",
			Query:             q,
			Vars:              hw{"sha1": wmataSha1},
			Selector:          "feed_versions.0.pathways.#.pathway_id",
			SelectExpectCount: 100,
		},
		{
			Name:   "pathway_id",
			Query:  q,
			Vars:   hw{"sha1": wmataSha1, "where": hw{"pathway_id": "C08_137121"}},
			Expect: `{"feed_versions":[{"pathways":[{"pathway_id":"C08_137121","pathway_mode":7,"is_bidirectional":0,"traversal_time":5,"from_stop":{"stop_id":"NODE_C08_FG_PAID"},"to_stop":{"stop_id":"NODE_C08_FG_UNPAID"}}]}]}`,
		},
		{
			Name:  "pathway_mode",
			Query: q,
			Vars:  hw{"sha1": wmataSha1, "limit": 10_000, "where": hw{"pathway_mode": 5}},
			Sel: []testcaseSelector{
				{Selector: "feed_versions.0.pathways.#.pathway_id", ExpectCount: 212},
				{Selector: "feed_versions.0.pathways.#.pathway_mode", ExpectUnique: []string{"5"}},
				{Selector: `feed_versions.0.pathways.#(pathway_id%"C08_*")#.pathway_id`, Expect: []string{"C08_137116", "C08_137123", "C08_137126"}},
			},
		},
		{
			// Filter fields combine with AND; C08_137121 is an exit gate, not an elevator.
			Name:   "pathway_id and pathway_mode",
			Query:  q,
			Vars:   hw{"sha1": wmataSha1, "where": hw{"pathway_id": "C08_137121", "pathway_mode": 5}},
			Expect: `{"feed_versions":[{"pathways":[]}]}`,
		},
		{
			Name:   "feed version fields",
			Query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { pathways(limit:1) { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			Vars:   hw{"sha1": wmataSha1},
			Expect: `{"feed_versions":[{"pathways":[{"feed_version_sha1":"148d00724546e1526d5d84d1f9d095df24f6517c","feed_onestop_id":"WMATA","feed_version":{"sha1":"148d00724546e1526d5d84d1f9d095df24f6517c"}}]}]}`,
		},
		{
			Name:   "feed version without pathways",
			Query:  q,
			Vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			Expect: `{"feed_versions":[{"pathways":[]}]}`,
		},
	}
	return testcases
}

// feedVersionResolverPathwaysAndLevelsCursorTestcases pages through a feed
// version's pathways and levels with after, and selects them by id.
func feedVersionResolverPathwaysAndLevelsCursorTestcases(t testing.TB, cfg model.Config) []testcase {
	ctx := model.WithConfig(context.Background(), cfg)

	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	fvs, err := cfg.Finder.FindFeedVersions(ctx, nil, nil, nil, &model.FeedVersionFilter{Sha1: &wmataSha1})
	if err != nil {
		t.Fatal(err)
	}
	if len(fvs) == 0 {
		t.Fatal("wmata feed version not found")
	}
	pathwayGroups, err := cfg.Finder.PathwaysByFeedVersionIDs(ctx, nil, nil, nil, []int{fvs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	levelGroups, err := cfg.Finder.LevelsByFeedVersionIDs(ctx, nil, nil, nil, []int{fvs[0].ID})
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 1, "pathway groups", pathwayGroups)
	needAtLeast(t, 1, "level groups", levelGroups)
	var pathwayIDs, levelIDs []int
	for _, ent := range pathwayGroups[0] {
		pathwayIDs = append(pathwayIDs, ent.ID)
	}
	for _, ent := range levelGroups[0] {
		levelIDs = append(levelIDs, ent.ID)
	}
	needAtLeast(t, 3, "pathways", pathwayIDs)
	needAtLeast(t, 3, "levels", levelIDs)
	// Paging with after relies on ascending ids, so expect that order whatever
	// order the finder returned.
	slices.Sort(pathwayIDs)
	slices.Sort(levelIDs)

	pathwaysQuery := `query($sha1:String!, $after:Int, $where:PathwayFilter) { feed_versions(where:{sha1:$sha1}) { pathways(limit:10000, after:$after, where:$where) { id } } }`
	levelsQuery := `query($sha1:String!, $after:Int, $where:LevelFilter) { feed_versions(where:{sha1:$sha1}) { levels(limit:10000, after:$after, where:$where) { id } } }`
	pathwaysSel := "feed_versions.0.pathways.#.id"
	levelsSel := "feed_versions.0.levels.#.id"
	testcases := []testcase{
		{
			Name:  "pathways no cursor",
			Query: pathwaysQuery,
			Vars:  hw{"sha1": wmataSha1},
			F:     expectIDsInOrder(pathwaysSel, pathwayIDs),
		},
		{
			Name:  "pathways after first",
			Query: pathwaysQuery,
			Vars:  hw{"sha1": wmataSha1, "after": pathwayIDs[0]},
			F:     expectIDsInOrder(pathwaysSel, pathwayIDs[1:]),
		},
		{
			Name:  "pathways by ids",
			Query: pathwaysQuery,
			Vars:  hw{"sha1": wmataSha1, "where": hw{"ids": []int{pathwayIDs[2], pathwayIDs[0]}}},
			F:     expectIDsInOrder(pathwaysSel, []int{pathwayIDs[0], pathwayIDs[2]}),
		},
		{
			Name:  "pathways after first and by ids",
			Query: pathwaysQuery,
			Vars:  hw{"sha1": wmataSha1, "after": pathwayIDs[0], "where": hw{"ids": []int{pathwayIDs[0], pathwayIDs[2]}}},
			F:     expectIDsInOrder(pathwaysSel, []int{pathwayIDs[2]}),
		},
		{
			Name:  "levels no cursor",
			Query: levelsQuery,
			Vars:  hw{"sha1": wmataSha1},
			F:     expectIDsInOrder(levelsSel, levelIDs),
		},
		{
			Name:  "levels after first",
			Query: levelsQuery,
			Vars:  hw{"sha1": wmataSha1, "after": levelIDs[0]},
			F:     expectIDsInOrder(levelsSel, levelIDs[1:]),
		},
		{
			Name:  "levels by ids",
			Query: levelsQuery,
			Vars:  hw{"sha1": wmataSha1, "where": hw{"ids": []int{levelIDs[2], levelIDs[0]}}},
			F:     expectIDsInOrder(levelsSel, []int{levelIDs[0], levelIDs[2]}),
		},
		{
			Name:  "levels after first and by ids",
			Query: levelsQuery,
			Vars:  hw{"sha1": wmataSha1, "after": levelIDs[0], "where": hw{"ids": []int{levelIDs[0], levelIDs[2]}}},
			F:     expectIDsInOrder(levelsSel, []int{levelIDs[2]}),
		},
	}
	return testcases
}

// expectIDsInOrder checks that selector returns exactly these ids, in this order.
func expectIDsInOrder(selector string, ids []int) func(*testing.T, string) {
	return func(t *testing.T, jj string) {
		var got []int
		for _, v := range gjson.Get(jj, selector).Array() {
			got = append(got, int(v.Int()))
		}
		assert.Equal(t, ids, got)
	}
}

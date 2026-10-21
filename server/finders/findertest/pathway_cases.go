package findertest

// Cases duplicated from server/gql/pathway_resolver_test.go.

func pathwayResolverTestcases() []testcase {
	wmataSha1 := "148d00724546e1526d5d84d1f9d095df24f6517c"
	// NODE_C08_FG_PAID = Pentagon City fare-gates, paid side.
	// Outgoing pathways: 1 exit gate (mode 7) + 8 walkways/escalators (mode 1).
	// Incoming pathway: 1 fare gate (mode 6) from NODE_C08_FG_UNPAID.
	fgPaid := "NODE_C08_FG_PAID"

	testcases := []testcase{
		{
			// All outgoing pathways from a known stop are returned.
			name: "pathways_from_stop returns expected pathway_ids",
			query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop { pathway_id }
					}
				}
			}`,
			vars:     hw{"sha1": wmataSha1, "stop_id": fgPaid},
			selector: "feed_versions.0.stops.0.pathways_from_stop.#.pathway_id",
			selectExpect: []string{
				"C08_137121", "C08_137122", "C08_137125", "C08_137128", "C08_137131",
				"C08_137134", "C08_137137", "C08_137140", "C08_137143",
			},
		},
		{
			// Incoming pathways: NODE_C08_FG_PAID has exactly one entry pathway.
			name: "pathways_to_stop returns expected pathway_ids",
			query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_to_stop { pathway_id }
					}
				}
			}`,
			vars:         hw{"sha1": wmataSha1, "stop_id": fgPaid},
			selector:     "feed_versions.0.stops.0.pathways_to_stop.#.pathway_id",
			selectExpect: []string{"C08_137120"},
		},
		{
			// limit is honored by the resolver.
			name: "pathways_from_stop honors limit",
			query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop(limit: 3) { pathway_id }
					}
				}
			}`,
			vars:              hw{"sha1": wmataSha1, "stop_id": fgPaid},
			selector:          "feed_versions.0.stops.0.pathways_from_stop.#.pathway_id",
			selectExpectCount: 3,
		},
		{
			// Full pathway field set on a one-way exit gate (pathway_mode=7).
			name: "pathway fields - one-way exit gate",
			query: `query($sha1:String!,$stop_id:String!) {
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
			vars: hw{"sha1": wmataSha1, "stop_id": fgPaid},
			sel: []testcaseSelector{
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").pathway_mode`, expect: []string{"7"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").is_bidirectional`, expect: []string{"0"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").traversal_time`, expect: []string{"5"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").from_stop.stop_id`, expect: []string{"NODE_C08_FG_PAID"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137121").to_stop.stop_id`, expect: []string{"NODE_C08_FG_UNPAID"}},
			},
		},
		{
			// Bidirectional walkway: pathway_mode=1, is_bidirectional=1.
			name: "pathway fields - bidirectional walkway",
			query: `query($sha1:String!,$stop_id:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:$stop_id}) {
						pathways_from_stop {
							pathway_id pathway_mode is_bidirectional traversal_time
							to_stop { stop_id }
						}
					}
				}
			}`,
			vars: hw{"sha1": wmataSha1, "stop_id": fgPaid},
			sel: []testcaseSelector{
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").pathway_mode`, expect: []string{"1"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").is_bidirectional`, expect: []string{"1"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").traversal_time`, expect: []string{"7"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").to_stop.stop_id`, expect: []string{"NODE_C08_M_ELE1_TP"}},
			},
		},
		{
			// Stop on level C08_L1 (Mezzanine, level_index -1) resolves the level.
			name:   "stop level resolves",
			query:  `query($sha1:String!,$stop_id:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:$stop_id}) { level { level_id level_name level_index } } } }`,
			vars:   hw{"sha1": wmataSha1, "stop_id": fgPaid},
			expect: `{"feed_versions":[{"stops":[{"level":{"level_id":"C08_L1","level_index":-1,"level_name":"Mezzanine"}}]}]}`,
		},
		{
			// Stops without a level resolve level=null (the parent station has no level).
			name:   "stop without level resolves null",
			query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:"STN_C08"}) { level { level_id } } } }`,
			vars:   hw{"sha1": wmataSha1},
			expect: `{"feed_versions":[{"stops":[{"level":null}]}]}`,
		},
		{
			// parent station resolves on a child stop.
			name:   "stop parent station resolves",
			query:  `query($sha1:String!,$stop_id:String!) { feed_versions(where:{sha1:$sha1}) { stops(where:{stop_id:$stop_id}) { parent { stop_id stop_name location_type } } } }`,
			vars:   hw{"sha1": wmataSha1, "stop_id": fgPaid},
			expect: `{"feed_versions":[{"stops":[{"parent":{"location_type":1,"stop_id":"STN_C08","stop_name":"Pentagon City"}}]}]}`,
		},
		{
			// A station's children include platforms, entrances, and fare-gate nodes.
			name: "station children include platforms and entrances",
			query: `query($sha1:String!) {
				feed_versions(where:{sha1:$sha1}) {
					stops(where:{stop_id:"STN_C08"}) {
						children { stop_id }
					}
				}
			}`,
			vars:     hw{"sha1": wmataSha1},
			selector: "feed_versions.0.stops.0.children.#.stop_id",
			selectExpectContains: []string{
				"PF_C08_1", "PF_C08_2",
				"ENT_C08_E", "ENT_C08_W",
				"NODE_C08_FG_PAID", "NODE_C08_FG_UNPAID",
			},
		},
		{
			// from_stop and to_stop on a Pathway resolve to full Stop objects,
			// including nested level resolution.
			name: "pathway from_stop/to_stop resolve full Stop objects",
			query: `query($sha1:String!,$stop_id:String!) {
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
			vars: hw{"sha1": wmataSha1, "stop_id": fgPaid},
			sel: []testcaseSelector{
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").from_stop.stop_id`, expect: []string{"NODE_C08_FG_PAID"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").from_stop.location_type`, expect: []string{"3"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").from_stop.level.level_id`, expect: []string{"C08_L1"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").to_stop.stop_id`, expect: []string{"NODE_C08_M_ELE1_TP"}},
				{selector: `feed_versions.0.stops.0.pathways_from_stop.#(pathway_id=="C08_137122").to_stop.location_type`, expect: []string{"3"}},
			},
		},
	}
	return testcases
}

func feedVersionResolverPathwaysTestcases() []testcase {
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
			name:              "every pathway in the feed version",
			query:             q,
			vars:              hw{"sha1": wmataSha1, "limit": 10_000},
			selector:          "feed_versions.0.pathways.#.pathway_id",
			selectExpectCount: 2958,
		},
		{
			name:              "default limit",
			query:             q,
			vars:              hw{"sha1": wmataSha1},
			selector:          "feed_versions.0.pathways.#.pathway_id",
			selectExpectCount: 100,
		},
		{
			name:   "pathway_id",
			query:  q,
			vars:   hw{"sha1": wmataSha1, "where": hw{"pathway_id": "C08_137121"}},
			expect: `{"feed_versions":[{"pathways":[{"pathway_id":"C08_137121","pathway_mode":7,"is_bidirectional":0,"traversal_time":5,"from_stop":{"stop_id":"NODE_C08_FG_PAID"},"to_stop":{"stop_id":"NODE_C08_FG_UNPAID"}}]}]}`,
		},
		{
			name:  "pathway_mode",
			query: q,
			vars:  hw{"sha1": wmataSha1, "limit": 10_000, "where": hw{"pathway_mode": 5}},
			sel: []testcaseSelector{
				{selector: "feed_versions.0.pathways.#.pathway_id", expectCount: 212},
				{selector: "feed_versions.0.pathways.#.pathway_mode", expectUnique: []string{"5"}},
				{selector: `feed_versions.0.pathways.#(pathway_id%"C08_*")#.pathway_id`, expect: []string{"C08_137116", "C08_137123", "C08_137126"}},
			},
		},
		{
			// Filter fields combine with AND; C08_137121 is an exit gate, not an elevator.
			name:   "pathway_id and pathway_mode",
			query:  q,
			vars:   hw{"sha1": wmataSha1, "where": hw{"pathway_id": "C08_137121", "pathway_mode": 5}},
			expect: `{"feed_versions":[{"pathways":[]}]}`,
		},
		{
			name:   "feed version fields",
			query:  `query($sha1:String!) { feed_versions(where:{sha1:$sha1}) { pathways(limit:1) { feed_version_sha1 feed_onestop_id feed_version { sha1 } } } }`,
			vars:   hw{"sha1": wmataSha1},
			expect: `{"feed_versions":[{"pathways":[{"feed_version_sha1":"148d00724546e1526d5d84d1f9d095df24f6517c","feed_onestop_id":"WMATA","feed_version":{"sha1":"148d00724546e1526d5d84d1f9d095df24f6517c"}}]}]}`,
		},
		{
			name:   "feed version without pathways",
			query:  q,
			vars:   hw{"sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
			expect: `{"feed_versions":[{"pathways":[]}]}`,
		},
	}
	return testcases
}

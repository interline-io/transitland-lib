package resolvers

import (
	"testing"
)

func TestFeedResolver(t *testing.T) {
	testcases := []testcase{
		{
			"basic",
			`query { feeds {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"BA", "CT", "BA~rt", "test"},
		},
		{
			"basic fields",
			`query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {name onestop_id spec languages associated_feeds file}}`,
			hw{"onestop_id": "CT"},
			`{"feeds":[{"associated_feeds":["CT~rt"],"file":"server-test.dmfr.json","languages":["en-US"],"name":"Caltrain","onestop_id":"CT","spec":"gtfs"}]}`,
			"",
			nil,
		},
		// TODO: authorization,
		// TODO: associated_operators
		{
			"urls",
			`query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {urls { static_current static_historic }}}`,
			hw{"onestop_id": "CT"},
			`{"feeds":[{"urls":{"static_current":"../test/data/external/caltrain.zip","static_historic":["https://caltrain.com/old_feed.zip"]}}]}`,
			"",
			nil,
		},
		{
			"license",
			`query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {license {spdx_identifier url use_without_attribution create_derived_product redistribution_allowed commercial_use_allowed share_alike_optional attribution_text attribution_instructions}}}`,
			hw{"onestop_id": "CT"},
			` {"feeds":[{"license":{"attribution_instructions":"test attribution instructions","attribution_text":"data provided by 511.org","commercial_use_allowed":"yes","create_derived_product":"yes","redistribution_allowed":"no","share_alike_optional":"yes","spdx_identifier":"test","url":"http://assets.511.org/pdf/nextgen/developers/511_Data_Agreement_Final.pdf","use_without_attribution":"no"}}]}`,
			"",
			nil,
		},
		{
			"feed_versions",
			`query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {feed_versions { sha1 }}}`,
			hw{"onestop_id": "CT"},
			``,
			"feeds.0.feed_versions.#.sha1",
			[]string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			"feed_state",
			`query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {feed_state { feed_version { sha1 }}}}`,
			hw{"onestop_id": "CT"},
			`{"feeds":[{"feed_state":{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}}]}`,
			"",
			nil,
		},
		// filters
		{
			"where onestop_id",
			`query { feeds(where:{onestop_id:"test"}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"test"},
		},
		{
			"where spec=gtfs",
			`query { feeds(where:{spec:["gtfs"]}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"CT", "BA", "test"},
		},
		{
			"where spec=gtfs-rt",
			`query { feeds(where:{spec:["gtfs-rt"]}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"BA~rt"},
		},
		{
			"where fetch_error=true",
			`query { feeds(where:{fetch_error:true}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"test"},
		},
		{
			"where fetch_error=false",
			`query { feeds(where:{fetch_error:false}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"BA", "CT"},
		},
		{
			"where import_status=success",
			`query { feeds(where:{import_status:success}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"BA", "CT"},
		},
		{
			"where import_status=in_progress", // TODO: mock an in-progress import
			`query { feeds(where:{import_status:in_progress}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{},
		},
		{
			"where import_status=error", // TODO: mock an in-progress import
			`query { feeds(where:{import_status:error}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{},
		},
		{
			"where search", // TODO: mock an in-progress import
			`query { feeds(where:{search:"cal"}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"CT"},
		},
		{
			"where search ba", // TODO: mock an in-progress import
			`query { feeds(where:{search:"BA"}) {onestop_id}}`,
			hw{},
			``,
			"feeds.#.onestop_id",
			[]string{"BA", "BA~rt"},
		},
	}
	c := newTestClient()
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}

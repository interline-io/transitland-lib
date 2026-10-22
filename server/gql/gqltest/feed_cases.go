package gqltest

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/server/model"
)

func feedResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	testcases := []testcase{
		{
			Name:         "basic",
			Query:        `query { feeds {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "test-gbfs", "BA", "HA", "WMATA", "BA~rt", "CT~rt", "test", "EX", "supplemental"},
		},
		{
			Name:   "basic fields",
			Query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {name onestop_id spec languages file}}`,
			Vars:   hw{"onestop_id": "CT"},
			Expect: `{"feeds":[{"file":"server-test.dmfr.json","languages":["en-US"],"name":"Caltrain","onestop_id":"CT","spec":"GTFS"}]}`,
		},
		// urls
		{
			Name:   "urls",
			Query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {urls { static_current static_historic }}}`,
			Vars:   hw{"onestop_id": "CT"},
			Expect: `{"feeds":[{"urls":{"static_current":"file://testdata/server/gtfs/caltrain.zip","static_historic":["https://caltrain.com/old_feed.zip"]}}]}`,
		},
		{
			Name:   "search by url case insensitive",
			Query:  `query($url:String!) { feeds(where:{source_url:{url:$url}}) { onestop_id }}`,
			Vars:   hw{"url": "file://testdata/server/gtfs/Caltrain.zip"},
			Expect: `{"feeds":[{"onestop_id":"CT"}]}`,
		},
		{
			Name:   "search by url case sensitive",
			Query:  `query($url:String!) { feeds(where:{source_url:{url:$url, case_sensitive: true}}) { onestop_id }}`,
			Vars:   hw{"url": "file://testdata/server/gtfs/Caltrain.zip"},
			Expect: `{"feeds":[]}`,
		},
		{
			Name:   "search by url with type specified",
			Query:  `query($url:String!) { feeds(where:{source_url:{url:$url, type: static_current}}) { onestop_id }}`,
			Vars:   hw{"url": "file://testdata/server/gtfs/caltrain.zip"},
			Expect: `{"feeds":[{"onestop_id":"CT"}]}`,
		},
		{
			Name:   "search by url with type realtime_trip_updates",
			Query:  `query($url:String!) { feeds(where:{source_url:{url:$url, type: realtime_trip_updates}}) { onestop_id }}`,
			Vars:   hw{"url": "file://testdata/server/rt/BA.json"},
			Expect: `{"feeds":[{"onestop_id":"BA~rt"}]}`,
		},
		{
			Name:   "search by url with type",
			Query:  `query($url:String) { feeds(where:{source_url:{url: $url, type: realtime_trip_updates}}) { onestop_id }}`,
			Vars:   hw{"url": nil},
			Expect: `{"feeds":[{"onestop_id":"BA~rt"},{"onestop_id":"CT~rt"}]}`,
		},
		{
			Name:   "license",
			Query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {license {spdx_identifier url use_without_attribution create_derived_product redistribution_allowed commercial_use_allowed share_alike_optional attribution_text attribution_instructions}}}`,
			Vars:   hw{"onestop_id": "CT"},
			Expect: ` {"feeds":[{"license":{"attribution_instructions":"test attribution instructions","attribution_text":"test attribution text","commercial_use_allowed":"unknown","create_derived_product":"unknown","redistribution_allowed":"unknown","share_alike_optional":"unknown","spdx_identifier":"test-unknown","url":"http://assets.511.org/pdf/nextgen/developers/511_Data_Agreement_Final.pdf","use_without_attribution":"unknown"}}]}`,
		},
		{
			Name:         "feed_versions",
			Query:        `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {feed_versions { sha1 }}}`,
			Vars:         hw{"onestop_id": "CT"},
			Selector:     "feeds.0.feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			Name:   "feed_state",
			Query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {feed_state { feed_version { sha1 }}}}`,
			Vars:   hw{"onestop_id": "CT"},
			Expect: `{"feeds":[{"feed_state":{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}}]}`,
		},
		// filters
		{
			Name:         "where onestop_id",
			Query:        `query { feeds(where:{onestop_id:"test"}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"test"},
		},
		{
			Name:         "where spec=gtfs",
			Query:        `query { feeds(where:{spec:[GTFS]}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "BA", "HA", "WMATA", "test", "EX", "supplemental"},
		},
		{
			Name:         "where spec=gtfs-rt",
			Query:        `query { feeds(where:{spec:[GTFS_RT]}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA~rt", "CT~rt"},
		},
		{
			Name:         "where fetch_error=true",
			Query:        `query { feeds(where:{fetch_error:true}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"test"},
		},
		{
			Name:         "where fetch_error=false",
			Query:        `query { feeds(where:{fetch_error:false}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "BA", "HA", "WMATA", "EX", "supplemental"},
		},
		{
			Name:         "where import_status=success",
			Query:        `query { feeds(where:{import_status:SUCCESS}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "BA", "HA", "WMATA", "EX", "supplemental"},
		},
		{
			Name:         "where import_status=in_progress", // TODO: mock an in-progress import
			Query:        `query { feeds(where:{import_status:IN_PROGRESS}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "where import_status=error", // TODO: mock an in-progress import
			Query:        `query { feeds(where:{import_status:ERROR}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "where search", // TODO: mock an in-progress import
			Query:        `query { feeds(where:{search:"cal"}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT"},
		},
		{
			Name:         "where search ba", // TODO: mock an in-progress import
			Query:        `query { feeds(where:{search:"BA"}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA", "BA~rt"},
		},
		{
			Name:         "where search on ilike onestop_id",
			Query:        `query { feeds(where:{search:"~r"}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT~rt", "BA~rt"},
		},
		{
			// `_` and `%` are ILIKE wildcards; unescaped they over-match every row.
			Name:         "where search escapes ilike wildcards",
			Query:        `query { feeds(where:{search:"__"}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "where tags test=ok",
			Query:        `query { feeds(where:{tags:{test:"ok"}}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "where tags test=ok foo=fail",
			Query:        `query { feeds(where:{tags:{test:"ok", foo:"fail"}}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{},
		},
		{
			Name:         "where tags test=ok foo=bar",
			Query:        `query { feeds(where:{tags:{test:"ok", foo:"bar"}}) {onestop_id}}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "where tags test is present",
			Query:        `query { feeds(where:{tags:{test:""}}) {onestop_id }}`,
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		// feed fetches
		{
			Name:         "feed fetches",
			Query:        `query { feeds(where:{onestop_id:"BA"}) { onestop_id feed_fetches(limit:1) { success }}}`,
			Selector:     "feeds.0.feed_fetches.#.success",
			SelectExpect: []string{"true"},
		},
		{
			Name:         "feed fetches failed",
			Query:        `query { feeds(where:{onestop_id:"test"}) { onestop_id feed_fetches(limit:1, where:{success:false}) { success }}}`,
			Selector:     "feeds.0.feed_fetches.#.success",
			SelectExpect: []string{"false"},
		},
		// multiple queries
		{
			Name:         "feed fetches multiple queries 1",
			Query:        `query { feeds(where:{onestop_id:"CT"}) { onestop_id ok:feed_fetches(limit:1, where:{success:true}) { success } fail:feed_fetches(limit:1, where:{success:false}) { success }}}`,
			Selector:     "feeds.0.ok.#.success",
			SelectExpect: []string{"true"},
		},
		{
			Name:         "feed fetches multiple queries 2",
			Query:        `query { feeds(where:{onestop_id:"CT"}) { onestop_id ok:feed_fetches(limit:1, where:{success:true}) { success } fail:feed_fetches(limit:1, where:{success:false}) { success }}}`,
			Selector:     "feeds.0.fail.#.success",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func feedResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase {
	q := `query($lic:LicenseFilter) {feeds(where: {license: $lic}) {onestop_id}}`
	testcases := []testcase{
		// license: share_alike_optional
		{
			Name:         "license filter: share_alike_optional = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "YES"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"HA", "WMATA"},
		},
		{
			Name:         "license filter: share_alike_optional = no",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "license filter: share_alike_optional = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "UNKNOWN"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT"},
		},
		{
			Name:         "license filter: share_alike_optional = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "test-gbfs", "HA", "WMATA", "BA~rt", "CT~rt", "test", "EX", "supplemental"},
		},
		// license: create_derived_product
		{
			Name:         "license filter: create_derived_product = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"HA", "WMATA"},
		},
		{
			Name:         "license filter: create_derived_product = no",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "license filter: create_derived_product = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "UNKNOWN"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT"},
		},
		{
			Name:         "license filter: create_derived_product = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "test-gbfs", "HA", "WMATA", "BA~rt", "CT~rt", "test", "EX", "supplemental"},
		},
		// license: commercial_use_allowed
		{
			Name:         "license filter: commercial_use_allowed = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"HA", "WMATA"},
		},
		{
			Name:         "license filter: commercial_use_allowed = no",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "license filter: commercial_use_allowed = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "UNKNOWN"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT"},
		},
		{
			Name:         "license filter: commercial_use_allowed = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "test-gbfs", "HA", "WMATA", "BA~rt", "CT~rt", "test", "EX", "supplemental"},
		},
		// license: redistribution_allowed
		{
			Name:         "license filter: redistribution_allowed = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "YES"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"HA", "WMATA"},
		},
		{
			Name:         "license filter: redistribution_allowed = no",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "license filter: redistribution_allowed = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "UNKNOWN"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT"},
		},
		{
			Name:         "license filter: redistribution_allowed = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "test-gbfs", "HA", "WMATA", "BA~rt", "CT~rt", "test", "EX", "supplemental"},
		},
		// license: use_without_attribution
		{
			Name:         "license filter: use_without_attribution = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "YES"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"HA", "WMATA"},
		},
		{
			Name:         "license filter: use_without_attribution = no",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"BA"},
		},
		{
			Name:         "license filter: use_without_attribution = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "UNKNOWN"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT"},
		},
		{
			Name:         "license filter: use_without_attribution = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: []string{"CT", "ctran-flex", "test-gbfs", "HA", "WMATA", "BA~rt", "CT~rt", "test", "EX", "supplemental"},
		},
	}
	return testcases
}

func feedResolverCursorTestcases(t testing.TB, cfg model.Config) []testcase {
	allEnts, err := cfg.Finder.FindFeeds(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	needAtLeast(t, 2, "feeds", allEnts)
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.FeedID)
	}
	testcases := []testcase{
		{
			Name:         "no cursor",
			Query:        "query{feeds(limit:100){id onestop_id}}",
			Selector:     "feeds.#.onestop_id",
			SelectExpect: allIds,
		},
		{
			Name:         "after 0",
			Query:        "query{feeds(after: 0, limit:100){id onestop_id}}",
			Selector:     "feeds.#.onestop_id",
			SelectExpect: allIds,
		},
		{
			Name:         "after 1st",
			Query:        "query($after: Int!){feeds(after: $after, limit:100){id onestop_id}}",
			Vars:         hw{"after": allEnts[1].ID},
			Selector:     "feeds.#.onestop_id",
			SelectExpect: allIds[2:],
		},
	}
	return testcases
}

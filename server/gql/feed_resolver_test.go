package gql

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/model"
)

func TestFeedResolver(t *testing.T) {
	testcases := []testcase{
		{
			name:         "basic",
			query:        `query { feeds {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT", "test-gbfs", "BA", "HA", "BA~rt", "CT~rt", "test", "EX"},
		},
		{
			name:   "basic fields",
			query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {name onestop_id spec languages file}}`,
			vars:   hw{"onestop_id": "CT"},
			expect: `{"feeds":[{"file":"server-test.dmfr.json","languages":["en-US"],"name":"Caltrain","onestop_id":"CT","spec":"GTFS"}]}`,
		},
		// urls
		{
			name:   "urls",
			query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {urls { static_current static_historic }}}`,
			vars:   hw{"onestop_id": "CT"},
			expect: `{"feeds":[{"urls":{"static_current":"file://testdata/gtfs/caltrain.zip","static_historic":["https://caltrain.com/old_feed.zip"]}}]}`,
		},
		{
			name:   "search by url case insensitive",
			query:  `query($url:String!) { feeds(where:{source_url:{url:$url}}) { onestop_id }}`,
			vars:   hw{"url": "file://testdata/gtfs/Caltrain.zip"},
			expect: `{"feeds":[{"onestop_id":"CT"}]}`,
		},
		{
			name:   "search by url case sensitive",
			query:  `query($url:String!) { feeds(where:{source_url:{url:$url, case_sensitive: true}}) { onestop_id }}`,
			vars:   hw{"url": "file://testdata/gtfs/Caltrain.zip"},
			expect: `{"feeds":[]}`,
		},
		{
			name:   "search by url with type specified",
			query:  `query($url:String!) { feeds(where:{source_url:{url:$url, type: static_current}}) { onestop_id }}`,
			vars:   hw{"url": "file://testdata/gtfs/caltrain.zip"},
			expect: `{"feeds":[{"onestop_id":"CT"}]}`,
		},
		{
			name:   "search by url with type realtime_trip_updates",
			query:  `query($url:String!) { feeds(where:{source_url:{url:$url, type: realtime_trip_updates}}) { onestop_id }}`,
			vars:   hw{"url": "file://testdata/rt/BA.json"},
			expect: `{"feeds":[{"onestop_id":"BA~rt"}]}`,
		},
		{
			name:   "search by url with type",
			query:  `query($url:String) { feeds(where:{source_url:{url: $url, type: realtime_trip_updates}}) { onestop_id }}`,
			vars:   hw{"url": nil},
			expect: `{"feeds":[{"onestop_id":"BA~rt"},{"onestop_id":"CT~rt"}]}`,
		},
		{
			name:   "license",
			query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {license {spdx_identifier url use_without_attribution create_derived_product redistribution_allowed commercial_use_allowed share_alike_optional attribution_text attribution_instructions}}}`,
			vars:   hw{"onestop_id": "CT"},
			expect: ` {"feeds":[{"license":{"attribution_instructions":"test attribution instructions","attribution_text":"test attribution text","commercial_use_allowed":"unknown","create_derived_product":"unknown","redistribution_allowed":"unknown","share_alike_optional":"unknown","spdx_identifier":"test-unknown","url":"http://assets.511.org/pdf/nextgen/developers/511_Data_Agreement_Final.pdf","use_without_attribution":"unknown"}}]}`,
		},
		{
			name:         "feed_versions",
			query:        `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {feed_versions { sha1 }}}`,
			vars:         hw{"onestop_id": "CT"},
			selector:     "feeds.0.feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			name:   "feed_state",
			query:  `query($onestop_id:String!) { feeds(where:{onestop_id:$onestop_id}) {feed_state { feed_version { sha1 }}}}`,
			vars:   hw{"onestop_id": "CT"},
			expect: `{"feeds":[{"feed_state":{"feed_version":{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}}}]}`,
		},
		// filters
		{
			name:         "where onestop_id",
			query:        `query { feeds(where:{onestop_id:"test"}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"test"},
		},
		{
			name:         "where spec=gtfs",
			query:        `query { feeds(where:{spec:[GTFS]}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT", "BA", "test", "HA", "EX"},
		},
		{
			name:         "where spec=gtfs-rt",
			query:        `query { feeds(where:{spec:[GTFS_RT]}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA~rt", "CT~rt"},
		},
		{
			name:         "where fetch_error=true",
			query:        `query { feeds(where:{fetch_error:true}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"test"},
		},
		{
			name:         "where fetch_error=false",
			query:        `query { feeds(where:{fetch_error:false}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA", "CT", "HA", "EX"},
		},
		{
			name:         "where import_status=success",
			query:        `query { feeds(where:{import_status:SUCCESS}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA", "CT", "HA", "EX"},
		},
		{
			name:         "where import_status=in_progress", // TODO: mock an in-progress import
			query:        `query { feeds(where:{import_status:IN_PROGRESS}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "where import_status=error", // TODO: mock an in-progress import
			query:        `query { feeds(where:{import_status:ERROR}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "where search", // TODO: mock an in-progress import
			query:        `query { feeds(where:{search:"cal"}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT"},
		},
		{
			name:         "where search ba", // TODO: mock an in-progress import
			query:        `query { feeds(where:{search:"BA"}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA", "BA~rt"},
		},
		{
			name:         "where search on ilike onestop_id",
			query:        `query { feeds(where:{search:"~r"}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT~rt", "BA~rt"},
		},
		{
			name:         "where tags test=ok",
			query:        `query { feeds(where:{tags:{test:"ok"}}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "where tags test=ok foo=fail",
			query:        `query { feeds(where:{tags:{test:"ok", foo:"fail"}}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:         "where tags test=ok foo=bar",
			query:        `query { feeds(where:{tags:{test:"ok", foo:"bar"}}) {onestop_id}}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "where tags test is present",
			query:        `query { feeds(where:{tags:{test:""}}) {onestop_id }}`,
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		// feed fetches
		{
			name:         "feed fetches",
			query:        `query { feeds(where:{onestop_id:"BA"}) { onestop_id feed_fetches(limit:1) { success }}}`,
			selector:     "feeds.0.feed_fetches.#.success",
			selectExpect: []string{"true"},
		},
		{
			name:         "feed fetches failed",
			query:        `query { feeds(where:{onestop_id:"test"}) { onestop_id feed_fetches(limit:1, where:{success:false}) { success }}}`,
			selector:     "feeds.0.feed_fetches.#.success",
			selectExpect: []string{"false"},
		},
		// multiple queries
		{
			name:         "feed fetches multiple queries 1",
			query:        `query { feeds(where:{onestop_id:"CT"}) { onestop_id ok:feed_fetches(limit:1, where:{success:true}) { success } fail:feed_fetches(limit:1, where:{success:false}) { success }}}`,
			selector:     "feeds.0.ok.#.success",
			selectExpect: []string{"true"},
		},
		{
			name:         "feed fetches multiple queries 2",
			query:        `query { feeds(where:{onestop_id:"CT"}) { onestop_id ok:feed_fetches(limit:1, where:{success:true}) { success } fail:feed_fetches(limit:1, where:{success:false}) { success }}}`,
			selector:     "feeds.0.fail.#.success",
			selectExpect: []string{},
		},
		// spatial
		{
			name:         "radius",
			query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "radius 2",
			query:        `query($near:PointRadius) {feeds(where: {near:$near}) {onestop_id}}`,
			vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"HA"},
		},
		{
			name:  "within",
			query: `query($within:Polygon) {feeds(where: {within:$within}) {onestop_id}}`,
			vars: hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
				{-122.39803791046143, 37.794626736533836},
				{-122.40106344223022, 37.792303711508595},
				{-122.3965573310852, 37.789641468930114},
				{-122.3938751220703, 37.792354581451946},
				{-122.39803791046143, 37.794626736533836},
			}}}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "bbox 1",
			query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "bbox 2",
			query:        `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{},
		},
		{
			name:        "bbox too large",
			query:       `query($bbox:BoundingBox) {feeds(where: {bbox:$bbox}) {onestop_id}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			selector:    "feeds.#.onestop_id",
			expectError: true,
			f: func(t *testing.T, jj string) {
			},
		},
		// TODO: authorization,
		// TODO: associated_operators
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestFeedResolver_Cursor(t *testing.T) {
	c, cfg := newTestClient(t)
	allEnts, err := cfg.Finder.FindFeeds(model.WithConfig(context.Background(), cfg), nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	allIds := []string{}
	for _, ent := range allEnts {
		allIds = append(allIds, ent.FeedID)
	}
	testcases := []testcase{
		{
			name:         "no cursor",
			query:        "query{feeds(limit:10){id onestop_id}}",
			selector:     "feeds.#.onestop_id",
			selectExpect: allIds,
		},
		{
			name:         "after 0",
			query:        "query{feeds(after: 0, limit:10){id onestop_id}}",
			selector:     "feeds.#.onestop_id",
			selectExpect: allIds,
		},
		{
			name:         "after 1st",
			query:        "query($after: Int!){feeds(after: $after, limit:10){id onestop_id}}",
			vars:         hw{"after": allEnts[1].ID},
			selector:     "feeds.#.onestop_id",
			selectExpect: allIds[2:],
		},
	}
	queryTestcases(t, c, testcases)
}

func TestFeedResolver_License(t *testing.T) {
	q := `query($lic:LicenseFilter) {feeds(where: {license: $lic}) {onestop_id}}`
	testcases := []testcase{
		// license: share_alike_optional
		{
			name:         "license filter: share_alike_optional = yes",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"HA"},
		},
		{
			name:         "license filter: share_alike_optional = no",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "license filter: share_alike_optional = unknown",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "UNKNOWN"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT"},
		},
		{
			name:         "license filter: share_alike_optional = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT", "test-gbfs", "HA", "BA~rt", "CT~rt", "test", "EX"},
		},
		// license: create_derived_product
		{
			name:         "license filter: create_derived_product = yes",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "YES"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"HA"},
		},
		{
			name:         "license filter: create_derived_product = no",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "license filter: create_derived_product = unknown",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "UNKNOWN"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT"},
		},
		{
			name:         "license filter: create_derived_product = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT", "test-gbfs", "HA", "BA~rt", "CT~rt", "test", "EX"},
		},
		// license: commercial_use_allowed
		{
			name:         "license filter: commercial_use_allowed = yes",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"HA"},
		},
		{
			name:         "license filter: commercial_use_allowed = no",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "license filter: commercial_use_allowed = unknown",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "UNKNOWN"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT"},
		},
		{
			name:         "license filter: commercial_use_allowed = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT", "test-gbfs", "HA", "BA~rt", "CT~rt", "test", "EX"},
		},
		// license: redistribution_allowed
		{
			name:         "license filter: redistribution_allowed = yes",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"HA"},
		},
		{
			name:         "license filter: redistribution_allowed = no",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "license filter: redistribution_allowed = unknown",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "UNKNOWN"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT"},
		},
		{
			name:         "license filter: redistribution_allowed = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT", "test-gbfs", "HA", "BA~rt", "CT~rt", "test", "EX"},
		},

		// license: use_without_attribution
		{
			name:         "license filter: use_without_attribution = yes",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"HA"},
		},
		{
			name:         "license filter: use_without_attribution = no",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"BA"},
		},
		{
			name:         "license filter: use_without_attribution = unknown",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "UNKNOWN"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT"},
		},
		{
			name:         "license filter: use_without_attribution = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:     "feeds.#.onestop_id",
			selectExpect: []string{"CT", "test-gbfs", "HA", "BA~rt", "CT~rt", "test", "EX"},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

package gqltest

import (
	"fmt"
	"slices"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func feedVersionResolverLocationsTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		{
			Name:  "locations",
			Query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { locations { location_id } }}`,
			Vars:  hw{"sha1": ctranFlexSha1},
			F: func(t *testing.T, jj string) {
				locs := gjson.Get(jj, "feed_versions.0.locations").Array()
				assert.Greater(t, len(locs), 0, "expected at least one location")
			},
		},
		{
			Name:   "location filter by id",
			Query:  `query($sha1: String!, $location_id: String) { feed_versions(where:{sha1:$sha1}) { locations(where:{location_id:$location_id}) { location_id stop_name } }}`,
			Vars:   hw{"sha1": ctranFlexSha1, "location_id": "location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"},
			Expect: `{"feed_versions":[{"locations":[{"location_id":"location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59","stop_name":"Rose Village"}]}]}`,
		},
	}
	return testcases
}

func feedVersionResolverBookingRulesTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		{
			Name:  "booking rules",
			Query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { booking_rules { booking_rule_id booking_type prior_notice_start_day } }}`,
			Vars:  hw{"sha1": ctranFlexSha1},
			F: func(t *testing.T, jj string) {
				brs := gjson.Get(jj, "feed_versions.0.booking_rules").Array()
				assert.Greater(t, len(brs), 0, "expected at least one booking rule")
			},
		},
		{
			Name:   "booking rule filter by id",
			Query:  `query($sha1: String!, $booking_rule_id: String) { feed_versions(where:{sha1:$sha1}) { booking_rules(where:{booking_rule_id:$booking_rule_id}) { booking_rule_id booking_type } }}`,
			Vars:   hw{"sha1": ctranFlexSha1, "booking_rule_id": "booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a"},
			Expect: `{"feed_versions":[{"booking_rules":[{"booking_rule_id":"booking_rule_id__2bc6804f-9e24-4b91-8947-c73a2363e7b6_MTWTFxx_20220107_20320522__053000_190000__053000_190000__m_b3a73dc523608998d850c431bf49b740093fd69415233fb3e74709073b335b6a","booking_type":1}]}]}`,
		},
	}
	return testcases
}

func feedVersionResolverLocationGroupsTestcases(t testing.TB, cfg model.Config) []testcase {
	ctranFlexSha1 := "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"
	testcases := []testcase{
		{
			Name:  "location groups",
			Query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { location_groups { location_group_id location_group_name } }}`,
			Vars:  hw{"sha1": ctranFlexSha1},
			F: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				assert.Greater(t, len(lgs), 0, "expected at least one location group")
			},
		},
		{
			Name:  "location groups with limit",
			Query: `query($sha1: String!) { feed_versions(where:{sha1:$sha1}) { location_groups(limit:2) { location_group_id } }}`,
			Vars:  hw{"sha1": ctranFlexSha1},
			F: func(t *testing.T, jj string) {
				lgs := gjson.Get(jj, "feed_versions.0.location_groups").Array()
				assert.LessOrEqual(t, len(lgs), 2, "expected at most 2 location groups with limit:2")
			},
		},
		{
			Name:   "location group filter by id",
			Query:  `query($sha1: String!, $location_group_id: String) { feed_versions(where:{sha1:$sha1}) { location_groups(where:{location_group_id:$location_group_id}) { location_group_id location_group_name } }}`,
			Vars:   hw{"sha1": ctranFlexSha1, "location_group_id": "location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e"},
			Expect: `{"feed_versions":[{"location_groups":[{"location_group_id":"location_group_id__db7489d3-7478-4d3b-a47f-60c58e3fed6e","location_group_name":"VA Medical Center (fixed route stop)"}]}]}`,
		},
	}
	return testcases
}

func feedVersionResolverTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	vars := hw{"feed_version_sha1": "d2813c293bcfd7a97dde599527ae6c62c98e66c6"}
	testcases := []testcase{
		{
			Name:         "basic",
			Query:        `query {  feed_versions {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6", "c969427f56d3a645195dd8365cde6d7feae7e99b", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "43e2278aa272879c79460582152b04e7487f0493", "96b67c0934b689d9085c52967365d8c233ea321d", "e8bc76c3c8602cad745f41a49ed5c5627ad6904c", "148d00724546e1526d5d84d1f9d095df24f6517c", supplementalSha1},
		},
		{
			Name:   "basic fields",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {sha1 url earliest_calendar_date latest_calendar_date name description} }`,
			Vars:   vars,
			Expect: `{"feed_versions":[{"description":null,"earliest_calendar_date":"2017-10-02","latest_calendar_date":"2019-10-06","name":null,"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","url":"file://testdata/server/gtfs/caltrain.zip"}]}`,
		},
		{
			Name:   "sha1_dir",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {sha1 sha1_dir} }`,
			Vars:   vars,
			Expect: `{"feed_versions":[{"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","sha1_dir":"d9b0478a85b85fc68f14136fd96c0551fc0bfc98"}]}`,
		},
		// children
		{
			Name:   "feed",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed{onestop_id}} }`,
			Vars:   vars,
			Expect: `{"feed_versions":[{"feed":{"onestop_id":"CT"}}]}`,
		},
		{
			Name:   "feed_version_gtfs_import",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed_version_gtfs_import{success in_progress}} }`,
			Vars:   vars,
			Expect: `{"feed_versions":[{"feed_version_gtfs_import":{"in_progress":false,"success":true}}]}`,
		},
		{
			Name:   "service_window",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {service_window{default_timezone feed_start_date feed_end_date earliest_calendar_date latest_calendar_date fallback_week}} }`,
			Vars:   vars,
			Expect: `{"feed_versions":[{"service_window":{"default_timezone":"America/Los_Angeles","earliest_calendar_date":"2017-10-02","fallback_week":"2018-06-18","feed_end_date":null,"feed_start_date":null,"latest_calendar_date":"2019-10-06"}}]}`,
		},
		{
			Name:   "geohashes precision 3",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {geohashes(precision: 3) {geohash stop_count}} }`,
			Vars:   hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"}, // BART
			Expect: `{"feed_versions":[{"geohashes":[{"geohash":"9q8","stop_count":15},{"geohash":"9q9","stop_count":30},{"geohash":"9qc","stop_count":5}]}]}`,
		},
		{
			Name:  "geohashes default precision",
			Query: `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {geohashes {geohash stop_count}} }`,
			Vars:  hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"}, // BART
			F: func(t *testing.T, jj string) {
				cells := gjson.Get(jj, "feed_versions.0.geohashes").Array()
				assert.Len(t, cells, 33)
				stops := int64(0)
				for _, cell := range cells {
					assert.Len(t, cell.Get("geohash").String(), 5)
					stops += cell.Get("stop_count").Int()
				}
				assert.Equal(t, int64(50), stops)
				assert.Equal(t, "9q8vy", cells[0].Get("geohash").String())
			},
		},
		{
			Name:  "geohashes batched across feed versions",
			Query: `query {  feed_versions(limit: 100) {sha1 geohashes {geohash}} }`,
			F: func(t *testing.T, jj string) {
				expect := map[string]int{
					"e535eb2b3b9ac3ef15d82c56575e914575e732e0": 33,
					"d2813c293bcfd7a97dde599527ae6c62c98e66c6": 21,
					"c969427f56d3a645195dd8365cde6d7feae7e99b": 56,
					"43e2278aa272879c79460582152b04e7487f0493": 8,
				}
				for _, fv := range gjson.Get(jj, "feed_versions").Array() {
					if n, ok := expect[fv.Get("sha1").String()]; ok {
						assert.Len(t, fv.Get("geohashes").Array(), n, fv.Get("sha1").String())
						delete(expect, fv.Get("sha1").String())
					}
				}
				assert.Empty(t, expect, "feed versions missing from the result")
			},
		},
		{
			Name:   "geohashes precision not stored",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {geohashes(precision: 7) {geohash}} }`,
			Vars:   hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"}, // BART
			Expect: `{"feed_versions":[{"geohashes":[]}]}`,
		},
		{
			Name:   "feed_infos",
			Query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed_infos {feed_publisher_name feed_publisher_url feed_lang feed_version feed_start_date feed_end_date}} }`,
			Vars:   hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"}, // check BART instead
			Expect: `{"feed_versions":[{"feed_infos":[{"feed_end_date":"2019-07-01","feed_lang":"en","feed_publisher_name":"Bay Area Rapid Transit","feed_publisher_url":"http://www.bart.gov","feed_start_date":"2018-05-26","feed_version":"47"}]}]}`,
		},
		{
			Name:         "files",
			Query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {files {name rows sha1 header csv_like size}} }`,
			Vars:         vars,
			Selector:     "feed_versions.0.files.#.name",
			SelectExpect: []string{"agency.txt", "calendar.txt", "calendar_attributes.txt", "calendar_dates.txt", "directions.txt", "fare_attributes.txt", "fare_rules.txt", "farezone_attributes.txt", "frequencies.txt", "realtime_routes.txt", "routes.txt", "shapes.txt", "stop_attributes.txt", "stop_times.txt", "stops.txt", "transfers.txt", "trips.txt"},
		},
		{
			Name:     "file details",
			Query:    `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {files {name rows sha1 header csv_like size values_count values_unique}} }`,
			Vars:     vars,
			Selector: "feed_versions.0.files.#.name",
			F: func(t *testing.T, jj string) {
				for _, fvfile := range gjson.Get(jj, "feed_versions.0.files").Array() {
					if fvfile.Get("name").String() != "trips.txt" {
						continue
					}
					assert.Equal(t, fvfile.Get("rows").Int(), int64(185))
					assert.Equal(t, fvfile.Get("size").Int(), int64(14648))
					assert.Equal(t, fvfile.Get("sha1").String(), "1ad77955e41e33cb1fceb694df27ced80e0ecbd3")
					assert.Equal(t, fvfile.Get("header").String(), "route_id,service_id,trip_id,trip_headsign,direction_id,block_id,shape_id,wheelchair_accessible,bikes_allowed,trip_short_name")
					assert.Equal(t, fvfile.Get("values_unique.service_id").Int(), int64(27))
					assert.Equal(t, fvfile.Get("values_unique.wheelchair_accessible").Int(), int64(2))
					assert.Equal(t, fvfile.Get("values_count.service_id").Int(), int64(185))
					assert.Equal(t, fvfile.Get("values_count.trip_id").Int(), int64(185))
				}

			},
		},
		{
			Name:         "agencies",
			Query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {agencies {agency_id}} }`,
			Vars:         vars,
			Selector:     "feed_versions.0.agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us"},
		},
		{
			Name:         "routes",
			Query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {routes {route_id}} }`,
			Vars:         vars,
			Selector:     "feed_versions.0.routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			Name:         "stops",
			Query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {stops(limit:1000) {stop_id}} }`,
			Vars:         vars,
			Selector:     "feed_versions.0.stops.#.stop_id",
			SelectExpect: []string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322", "777402", "777403"},
		},
		{
			Name:         "locations",
			Query:        `query { feed_versions(where:{sha1:"e8bc76c3c8602cad745f41a49ed5c5627ad6904c"}) {locations(limit:1000) {location_id}} }`,
			Selector:     "feed_versions.0.locations.#.location_id",
			SelectExpect: []string{"location_id__75e0de0d-d90c-4f15-a6cc-001f734e0f13", "location_id__8d41f4d3-7760-457e-94e1-6f7980cb3c20", "location_id__ac79ba5e-31ae-4879-a455-a053862dbe59", "location_id__2a077a44-c1e9-44c6-8b26-6ece58b64db6", "location_id__43ca2d5b-a235-4669-a27e-371a7c528cca", "location_id__bb80cf18-9fa7-498a-b22f-1f66eb4214a6", "location_id__11f830d0-adec-468a-a8d6-513184e476a1", "location_id__c7400cc8-959c-42c8-991f-8f601ec9ea59"},
		},
		// where
		{
			Name:         "where feed_onestop_id",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT"}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			Name:         "where sha1",
			Query:        `query{feed_versions(where:{sha1:"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			Name:         "where sha1_dir",
			Query:        `query{feed_versions(where:{sha1_dir:"d9b0478a85b85fc68f14136fd96c0551fc0bfc98"}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			Name:         "where sha1 does not match sha1_dir",
			Query:        `query{feed_versions(where:{sha1:"d9b0478a85b85fc68f14136fd96c0551fc0bfc98"}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		{
			Name:         "where import_status success",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", import_status:SUCCESS}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		// feed version coverage
		// start date - feed start date before start_date
		{
			Name:         "covers start_date using feed info",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-12-31"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			Name:         "covers start_date using feed info 2",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-02-08"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			Name:         "covers start_date using feed info 3",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-02-07"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		{
			Name:         "covers start_date using earliest and latest calendar dates",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{start_date:"2016-02-07"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		{
			Name:         "covers start_date using earliest and latest calendar dates 2",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{start_date:"2018-02-07"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		// end date -- feed end date after end_date
		{
			Name:         "covers end_date using feed info",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{end_date:"2016-12-31"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			Name:         "covers end_date using feed info 2",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{end_date:"2017-01-01"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			Name:         "covers end_date using feed info 3",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{end_date:"2017-01-02"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		{
			Name:         "covers end_date using earliest and latest calendar dates",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{end_date:"2019-10-01"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			Name:         "covers end_date using earliest and latest calendar dates 2",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{end_date:"2022-05-01"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		// start date + end date -- feed includes in window
		{
			Name:         "covers start_date and end_date",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-08-01", end_date:"2016-08-30"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			Name:         "covers start_date and end_date 2",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2018-06-01", end_date:"2018-06-30"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
		},
		{
			Name:         "covers start_date and end_date using earliest and latest calendar date",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{start_date:"2018-06-01", end_date:"2018-06-30"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		// covers fetched_before
		{
			Name:         "covers fetched_before",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_before:"2123-04-05T06:07:08.9Z"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907", "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "covers fetched_before 2",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_before:"2009-08-07T06:05:04.3Z"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		// covers fetched_after
		{
			Name:         "covers fetched_after",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_after:"2009-08-07T06:05:04.3Z"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907", "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			Name:         "covers fetched_after 2",
			Query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_after:"2123-04-05T06:07:08.9Z"}}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		// there isnt a fv with this import status in test db
		{
			Name:         "where import_status error",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", import_status:ERROR}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
		// there isnt a fv with this import status in test db
		{
			Name:         "where import_status error",
			Query:        `query{feed_versions(where:{feed_onestop_id:"CT", import_status:IN_PROGRESS}) {sha1} }`,
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{},
		},
	}
	return testcases
}

func feedVersionResolverLicenseTestcases(t testing.TB, cfg model.Config) []testcase {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	q := `query($lic:LicenseFilter) {feed_versions(where: {license: $lic}) {sha1 feed { onestop_id} }}`
	baFvs := []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"}
	haFvs := []string{"c969427f56d3a645195dd8365cde6d7feae7e99b", "148d00724546e1526d5d84d1f9d095df24f6517c"}
	ctFvs := []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}
	HaCtExFvs := []string{"43e2278aa272879c79460582152b04e7487f0493", "c969427f56d3a645195dd8365cde6d7feae7e99b", "148d00724546e1526d5d84d1f9d095df24f6517c", "d2813c293bcfd7a97dde599527ae6c62c98e66c6", "e8bc76c3c8602cad745f41a49ed5c5627ad6904c", supplementalSha1}
	testcases := []testcase{
		// license: share_alike_optional
		{
			Name:         "license filter: share_alike_optional = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "YES"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: haFvs,
		},
		{
			Name:         "license filter: share_alike_optional = no",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: baFvs,
		},
		{
			Name:         "license filter: share_alike_optional = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "UNKNOWN"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: ctFvs,
		},
		{
			Name:         "license filter: share_alike_optional = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: HaCtExFvs,
		},
		// license: create_derived_product
		{
			Name:         "license filter: create_derived_product = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "YES"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: haFvs,
		},
		{
			Name:         "license filter: create_derived_product = no",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: baFvs,
		},
		{
			Name:         "license filter: create_derived_product = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "UNKNOWN"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: ctFvs,
		},
		{
			Name:         "license filter: create_derived_product = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: HaCtExFvs,
		},
		// license: commercial_use_allowed
		{
			Name:         "license filter: commercial_use_allowed = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "YES"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: haFvs,
		},
		{
			Name:         "license filter: commercial_use_allowed = no",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: baFvs,
		},
		{
			Name:         "license filter: commercial_use_allowed = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "UNKNOWN"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			Name:         "license filter: commercial_use_allowed = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: HaCtExFvs,
		},
		// license: redistribution_allowed
		{
			Name:         "license filter: redistribution_allowed = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "YES"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: haFvs,
		},
		{
			Name:         "license filter: redistribution_allowed = no",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: baFvs,
		},
		{
			Name:         "license filter: redistribution_allowed = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "UNKNOWN"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: ctFvs,
		},
		{
			Name:         "license filter: redistribution_allowed = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: HaCtExFvs,
		},
		// license: use_without_attribution
		{
			Name:         "license filter: use_without_attribution = yes",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "YES"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: haFvs,
		},
		{
			Name:         "license filter: use_without_attribution = no",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: baFvs,
		},
		{
			Name:         "license filter: use_without_attribution = unknown",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "UNKNOWN"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: ctFvs,
		},
		{
			Name:         "license filter: use_without_attribution = exclude_no",
			Query:        q,
			Vars:         hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			Selector:     "feed_versions.#.sha1",
			SelectExpect: HaCtExFvs,
		},
	}
	return testcases
}

func feedVersionResolverTripsDateTestcases() []testcaseWithClock {
	weekdayTrips := []string{"STBA", "CITY1", "CITY2", "AB1", "AB2", "BFC1", "BFC2"}
	weekendTrips := []string{"STBA", "CITY1", "CITY2", "AB1", "AB2", "BFC1", "BFC2", "AAMV1", "AAMV2", "AAMV3", "AAMV4"} // all trips
	const q = `
	query ($sha1: String, $service_date: Date, $relative_date: RelativeDate, $use_service_window:Boolean) {
		feed_versions(where: {sha1: $sha1}) {
			id
			sha1
			trips(where: {relative_date: $relative_date, service_date: $service_date,use_service_window:$use_service_window}) {
				id
				trip_id
			}
		}
	}`
	testcases := []testcaseWithClock{
		{
			whenUtc: "2007-02-05T22:00:00Z",
			testcase: testcase{
				Name:         "trips, no filters",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493"},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekendTrips,
			},
		},
		{
			whenUtc: "2007-02-05T22:00:00Z",
			testcase: testcase{
				Name:         "trips, service date (tuesday)",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2007-02-06"},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekdayTrips,
			},
		},
		{
			whenUtc: "2007-02-05T22:00:00Z",
			testcase: testcase{
				Name:         "trips, service date (saturday)",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2007-02-10"},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekendTrips, // all trips
			},
		},
		// Relative dates
		{
			whenUtc: "2007-02-03T22:00:00Z",
			testcase: testcase{
				Name:         "trips, relative date (today, today is saturday)",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "TODAY"},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekendTrips,
			},
		},
		{
			whenUtc: "2007-02-03T22:00:00Z",
			testcase: testcase{
				Name:         "trips, relative date (next-monday, today is saturday)",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "NEXT_MONDAY"},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekdayTrips,
			},
		},
		{
			whenUtc: "2007-02-05T22:00:00Z",
			testcase: testcase{
				Name:         "trips, relative date (next-saturday, today is monday)",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "NEXT_SATURDAY"},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekendTrips, // all trips
			},
		},
		// Window
		{
			whenUtc: "2024-07-23T22:00:00Z",
			testcase: testcase{
				Name:         "trips, service date (tuesday), outside of window",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2024-07-23"},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: []string{},
			},
		},
		{
			whenUtc: "2024-07-23T22:00:00Z",
			testcase: testcase{
				Name:         "trips, service date (tuesday), outside of window, use fallback",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2024-07-23", "use_service_window": true},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekdayTrips,
			},
		},
		{
			whenUtc: "2024-07-23T22:00:00Z",
			testcase: testcase{
				Name:         "trips, relative date (next-saturday, today is tuesday), outside of window, use fallback",
				Query:        q,
				Vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "NEXT_SATURDAY", "use_service_window": true},
				Selector:     "feed_versions.0.trips.#.trip_id",
				SelectExpect: weekendTrips,
			},
		},
	}
	return testcases
}

// feedVersionResolverTableCursorsTest pages through each GTFS table list with
// `after`, and selects records from it with `where: {ids}`.
func feedVersionResolverTableCursorsTest(t *testing.T, c *client.Client, cfg model.Config) {
	supplementalSha1 := testconfig.SupplementalSha1(t, cfg)
	tcs := []struct {
		field string
		sha1  string
	}{
		{"frequencies", "43e2278aa272879c79460582152b04e7487f0493"},
		{"transfers", "e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
		{"location_group_stops", "e8bc76c3c8602cad745f41a49ed5c5627ad6904c"},
		{"stop_areas", supplementalSha1},
		{"translations", supplementalSha1},
		{"attributions", supplementalSha1},
	}
	for _, tc := range tcs {
		t.Run(tc.field, func(t *testing.T) {
			q := fmt.Sprintf(`query($sha1: String!, $after: Int, $ids: [Int!]) { feed_versions(where: {sha1: $sha1}) { %s(limit: 100000, after: $after, where: {ids: $ids}) { id } } }`, tc.field)
			queryIDs := func(vars hw) []int64 {
				var resp map[string]any
				opts := []client.Option{}
				for k, v := range vars {
					opts = append(opts, client.Var(k, v))
				}
				if err := c.Post(q, &resp, opts...); err != nil {
					t.Fatal(err)
				}
				var ids []int64
				for _, v := range gjson.Get(gqlcase.ToJSON(resp), fmt.Sprintf("feed_versions.0.%s.#.id", tc.field)).Array() {
					ids = append(ids, v.Int())
				}
				return ids
			}
			ids := queryIDs(hw{"sha1": tc.sha1})
			if len(ids) < 3 {
				t.Fatalf("got %d records, need at least 3", len(ids))
			}
			// Paging with after relies on ascending ids.
			assert.True(t, slices.IsSorted(ids), "ids are not in ascending order: %v", ids)
			assert.Equal(t, ids[1:], queryIDs(hw{"sha1": tc.sha1, "after": ids[0]}))
			assert.Equal(t, []int64{ids[0], ids[2]}, queryIDs(hw{"sha1": tc.sha1, "ids": []int64{ids[2], ids[0]}}))
			assert.Equal(t, []int64{ids[2]}, queryIDs(hw{"sha1": tc.sha1, "after": ids[0], "ids": []int64{ids[0], ids[2]}}))
		})
	}
}

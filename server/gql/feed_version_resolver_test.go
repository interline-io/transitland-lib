package gql

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestFeedVersionResolver(t *testing.T) {
	vars := hw{"feed_version_sha1": "d2813c293bcfd7a97dde599527ae6c62c98e66c6"}
	within := hw{"within": hw{"type": "Polygon", "coordinates": [][][]float64{{
		{-122.39803791046143, 37.794626736533836},
		{-122.40106344223022, 37.792303711508595},
		{-122.3965573310852, 37.789641468930114},
		{-122.3938751220703, 37.792354581451946},
		{-122.39803791046143, 37.794626736533836},
	}}}}
	testcases := []testcase{
		{
			name:         "basic",
			query:        `query {  feed_versions {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6", "c969427f56d3a645195dd8365cde6d7feae7e99b", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "43e2278aa272879c79460582152b04e7487f0493", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:   "basic fields",
			query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {sha1 url earliest_calendar_date latest_calendar_date name description} }`,
			vars:   vars,
			expect: `{"feed_versions":[{"description":null,"earliest_calendar_date":"2017-10-02","latest_calendar_date":"2019-10-06","name":null,"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","url":"file://testdata/server/gtfs/caltrain.zip"}]}`,
		},
		// children
		{
			name:   "feed",
			query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed{onestop_id}} }`,
			vars:   vars,
			expect: `{"feed_versions":[{"feed":{"onestop_id":"CT"}}]}`,
		},
		{
			name:   "feed_version_gtfs_import",
			query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed_version_gtfs_import{success in_progress}} }`,
			vars:   vars,
			expect: `{"feed_versions":[{"feed_version_gtfs_import":{"in_progress":false,"success":true}}]}`,
		},
		{
			name:   "service_window",
			query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {service_window{default_timezone feed_start_date feed_end_date earliest_calendar_date latest_calendar_date fallback_week}} }`,
			vars:   vars,
			expect: `{"feed_versions":[{"service_window":{"default_timezone":"America/Los_Angeles","earliest_calendar_date":"2017-10-02","fallback_week":"2018-06-18","feed_end_date":null,"feed_start_date":null,"latest_calendar_date":"2019-10-06"}}]}`,
		},
		{
			name:   "feed_infos",
			query:  `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed_infos {feed_publisher_name feed_publisher_url feed_lang feed_version feed_start_date feed_end_date}} }`,
			vars:   hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"}, // check BART instead
			expect: `{"feed_versions":[{"feed_infos":[{"feed_end_date":"2019-07-01","feed_lang":"en","feed_publisher_name":"Bay Area Rapid Transit","feed_publisher_url":"http://www.bart.gov","feed_start_date":"2018-05-26","feed_version":"47"}]}]}`,
		},
		{
			name:         "files",
			query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {files {name rows sha1 header csv_like size}} }`,
			vars:         vars,
			selector:     "feed_versions.0.files.#.name",
			selectExpect: []string{"agency.txt", "calendar.txt", "calendar_attributes.txt", "calendar_dates.txt", "directions.txt", "fare_attributes.txt", "fare_rules.txt", "farezone_attributes.txt", "frequencies.txt", "realtime_routes.txt", "routes.txt", "shapes.txt", "stop_attributes.txt", "stop_times.txt", "stops.txt", "transfers.txt", "trips.txt"},
		},
		{
			name:     "file details",
			query:    `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {files {name rows sha1 header csv_like size values_count values_unique}} }`,
			vars:     vars,
			selector: "feed_versions.0.files.#.name",
			f: func(t *testing.T, jj string) {

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
			name:         "agencies",
			query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {agencies {agency_id}} }`,
			vars:         vars,
			selector:     "feed_versions.0.agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us"},
		},
		{
			name:         "routes",
			query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {routes {route_id}} }`,
			vars:         vars,
			selector:     "feed_versions.0.routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			name:         "stops",
			query:        `query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {stops(limit:1000) {stop_id}} }`,
			vars:         vars,
			selector:     "feed_versions.0.stops.#.stop_id",
			selectExpect: []string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322", "777402", "777403"},
		},
		// where
		{
			name:         "where feed_onestop_id",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT"}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			name:         "where sha1",
			query:        `query{feed_versions(where:{sha1:"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			name:         "where import_status success",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", import_status:SUCCESS}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			name: "multiple where values",
			query: `query($within:Polygon) {
				feeds(where:{onestop_id:"BA"}) {
					all:    feed_versions {sha1} 
					all1:   feed_versions(limit:1) {sha1} 
					all2:   feed_versions(limit:2) {sha1} 
					ok:     feed_versions(where:{import_status:SUCCESS}) {sha1} 
					fail:   feed_versions(where:{import_status:ERROR}) {sha1} 	
					ip:     feed_versions(where:{import_status:IN_PROGRESS}) {sha1} 	
					within: feed_versions(where:{within:$within}) {sha1}
					covers: feed_versions(where:{covers:{start_date:"2016-12-31"}}) {sha1}
				}
			}`,
			vars: within,
			// selector:     "ok.#.sha1",
			// selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
			f: func(t *testing.T, jj string) {

				exp := []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"}
				var empty []string
				assert.Equal(t, exp, astr(gjson.Get(jj, "feeds.0.all.#.sha1").Array()))
				assert.Equal(t, exp[0:1], astr(gjson.Get(jj, "feeds.0.all1.#.sha1").Array()))
				assert.Equal(t, exp[0:2], astr(gjson.Get(jj, "feeds.0.all2.#.sha1").Array()))
				assert.Equal(t, exp, astr(gjson.Get(jj, "feeds.0.ok.#.sha1").Array()))
				assert.Equal(t, empty, astr(gjson.Get(jj, "feeds.0.fail.#.sha1").Array()))
				assert.Equal(t, empty, astr(gjson.Get(jj, "feeds.0.ip.#.sha1").Array()))
				assert.Equal(t, exp, astr(gjson.Get(jj, "feeds.0.within.#.sha1").Array()))
				assert.Equal(t, []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"}, astr(gjson.Get(jj, "feeds.0.covers.#.sha1").Array()))
			},
		},
		// feed version coverage
		// start date - feed start date before start_date
		{
			name:         "covers start_date using feed info",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-12-31"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			name:         "covers start_date using feed info 2",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-02-08"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			name:         "covers start_date using feed info 3",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-02-07"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		{
			name:         "covers start_date using earliest and latest calendar dates",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{start_date:"2016-02-07"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		{
			name:         "covers start_date using earliest and latest calendar dates 2",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{start_date:"2018-02-07"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		// end date -- feed end date after end_date
		{
			name:         "covers end_date using feed info",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{end_date:"2016-12-31"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			name:         "covers end_date using feed info 2",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{end_date:"2017-01-01"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			name:         "covers end_date using feed info 3",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{end_date:"2017-01-02"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		{
			name:         "covers end_date using earliest and latest calendar dates",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{end_date:"2019-10-01"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			name:         "covers end_date using earliest and latest calendar dates 2",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{end_date:"2022-05-01"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		// start date + end date -- feed includes in window
		{
			name:         "covers start_date and end_date",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2016-08-01", end_date:"2016-08-30"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907"},
		},
		{
			name:         "covers start_date and end_date 2",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{start_date:"2018-06-01", end_date:"2018-06-30"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0"},
		},
		{
			name:         "covers start_date and end_date using earliest and latest calendar date",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", covers:{start_date:"2018-06-01", end_date:"2018-06-30"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		// covers fetched_before
		{
			name:         "covers fetched_before",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_before:"2123-04-05T06:07:08.9Z"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907", "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "covers fetched_before 2",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_before:"2009-08-07T06:05:04.3Z"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		// covers fetched_after
		{
			name:         "covers fetched_after",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_after:"2009-08-07T06:05:04.3Z"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"dd7aca4a8e4c90908fd3603c097fabee75fea907", "e535eb2b3b9ac3ef15d82c56575e914575e732e0", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "covers fetched_after 2",
			query:        `query{feed_versions(where:{feed_onestop_id:"BA", covers:{fetched_after:"2123-04-05T06:07:08.9Z"}}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		// there isnt a fv with this import status in test db
		{
			name:         "where import_status error",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", import_status:ERROR}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		// there isnt a fv with this import status in test db
		{
			name:         "where import_status error",
			query:        `query{feed_versions(where:{feed_onestop_id:"CT", import_status:IN_PROGRESS}) {sha1} }`,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		// spatial
		{
			name:         "radius",
			query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			vars:         hw{"near": hw{"lon": -122.2698781543005, "lat": 37.80700393130445, "radius": 1000}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "radius 2",
			query:        `query($near:PointRadius) {feed_versions(where: {near:$near}) {sha1}}`,
			vars:         hw{"near": hw{"lon": -82.45717479225324, "lat": 27.95070842389974, "radius": 1000}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"c969427f56d3a645195dd8365cde6d7feae7e99b"},
		},
		{
			name:         "within",
			query:        `query($within:Polygon) {feed_versions(where: {within:$within}) {sha1}}`,
			vars:         within,
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "bbox 1",
			query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			vars:         hw{"bbox": hw{"min_lon": -122.2698781543005, "min_lat": 37.80700393130445, "max_lon": -122.2677640139239, "max_lat": 37.8088734037938}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"},
		},
		{
			name:         "bbox 2",
			query:        `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			vars:         hw{"bbox": hw{"min_lon": -124.3340029563042, "min_lat": 40.65505368922123, "max_lon": -123.9653594784379, "max_lat": 40.896440342606525}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{},
		},
		{
			name:        "bbox too large",
			query:       `query($bbox:BoundingBox) {feed_versions(where: {bbox:$bbox}) {sha1}}`,
			vars:        hw{"bbox": hw{"min_lon": -137.88020156441956, "min_lat": 30.072648315782004, "max_lon": -109.00421121090919, "max_lat": 45.02437957865729}},
			selector:    "feed_versions.#.sha1",
			expectError: true,
			f: func(t *testing.T, jj string) {
			},
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

func TestFeedVersionResolver_Trips_Date(t *testing.T) {
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
				name:         "trips, no filters",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493"},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekendTrips,
			},
		},
		{
			whenUtc: "2007-02-05T22:00:00Z",
			testcase: testcase{
				name:         "trips, service date (tuesday)",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2007-02-06"},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekdayTrips,
			},
		},
		{
			whenUtc: "2007-02-05T22:00:00Z",
			testcase: testcase{
				name:         "trips, service date (saturday)",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2007-02-10"},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekendTrips, // all trips
			},
		},
		// Relative dates
		{
			whenUtc: "2007-02-03T22:00:00Z",
			testcase: testcase{
				name:         "trips, relative date (today, today is saturday)",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "TODAY"},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekendTrips,
			},
		},
		{
			whenUtc: "2007-02-03T22:00:00Z",
			testcase: testcase{
				name:         "trips, relative date (next-monday, today is saturday)",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "NEXT_MONDAY"},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekdayTrips,
			},
		},
		{
			whenUtc: "2007-02-05T22:00:00Z",
			testcase: testcase{
				name:         "trips, relative date (next-saturday, today is monday)",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "NEXT_SATURDAY"},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekendTrips, // all trips
			},
		},
		// Window
		{
			whenUtc: "2024-07-23T22:00:00Z",
			testcase: testcase{
				name:         "trips, service date (tuesday), outside of window",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2024-07-23"},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: []string{},
			},
		},
		{
			whenUtc: "2024-07-23T22:00:00Z",
			testcase: testcase{
				name:         "trips, service date (tuesday), outside of window, use fallback",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "service_date": "2024-07-23", "use_service_window": true},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekdayTrips,
			},
		},
		{
			whenUtc: "2024-07-23T22:00:00Z",
			testcase: testcase{
				name:         "trips, relative date (next-saturday, today is tuesday), outside of window, use fallback",
				query:        q,
				vars:         hw{"sha1": "43e2278aa272879c79460582152b04e7487f0493", "relative_date": "NEXT_SATURDAY", "use_service_window": true},
				selector:     "feed_versions.0.trips.#.trip_id",
				selectExpect: weekendTrips,
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c, _ := newTestClientWithOpts(t, testconfig.Options{
				WhenUtc: tc.whenUtc,
			})
			queryTestcase(t, c, tc.testcase)
		})
	}
}

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

func TestFeedVersionResolver_License(t *testing.T) {
	q := `query($lic:LicenseFilter) {feed_versions(where: {license: $lic}) {sha1 feed { onestop_id} }}`
	baFvs := []string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "96b67c0934b689d9085c52967365d8c233ea321d"}
	haFvs := []string{"c969427f56d3a645195dd8365cde6d7feae7e99b"}
	ctFvs := []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}
	HaCtExFvs := []string{"43e2278aa272879c79460582152b04e7487f0493", "c969427f56d3a645195dd8365cde6d7feae7e99b", "d2813c293bcfd7a97dde599527ae6c62c98e66c6"}
	testcases := []testcase{
		// license: share_alike_optional
		{
			name:         "license filter: share_alike_optional = yes",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "YES"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: haFvs,
		},
		{
			name:         "license filter: share_alike_optional = no",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: baFvs,
		},
		{
			name:         "license filter: share_alike_optional = unknown",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "UNKNOWN"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: ctFvs,
		},
		{
			name:         "license filter: share_alike_optional = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"share_alike_optional": "EXCLUDE_NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: HaCtExFvs,
		},
		// license: create_derived_product
		{
			name:         "license filter: create_derived_product = yes",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "YES"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: haFvs,
		},
		{
			name:         "license filter: create_derived_product = no",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: baFvs,
		},
		{
			name:         "license filter: create_derived_product = unknown",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "UNKNOWN"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: ctFvs,
		},
		{
			name:         "license filter: create_derived_product = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"create_derived_product": "EXCLUDE_NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: HaCtExFvs,
		},
		// license: commercial_use_allowed
		{
			name:         "license filter: commercial_use_allowed = yes",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "YES"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: haFvs,
		},
		{
			name:         "license filter: commercial_use_allowed = no",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: baFvs,
		},
		{
			name:         "license filter: commercial_use_allowed = unknown",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "UNKNOWN"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: []string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			name:         "license filter: commercial_use_allowed = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"commercial_use_allowed": "EXCLUDE_NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: HaCtExFvs,
		},
		// license: redistribution_allowed
		{
			name:         "license filter: redistribution_allowed = yes",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "YES"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: haFvs,
		},
		{
			name:         "license filter: redistribution_allowed = no",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: baFvs,
		},
		{
			name:         "license filter: redistribution_allowed = unknown",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "UNKNOWN"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: ctFvs,
		},
		{
			name:         "license filter: redistribution_allowed = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"redistribution_allowed": "EXCLUDE_NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: HaCtExFvs,
		},

		// license: use_without_attribution
		{
			name:         "license filter: use_without_attribution = yes",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "YES"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: haFvs,
		},
		{
			name:         "license filter: use_without_attribution = no",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: baFvs,
		},
		{
			name:         "license filter: use_without_attribution = unknown",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "UNKNOWN"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: ctFvs,
		},
		{
			name:         "license filter: use_without_attribution = exclude_no",
			query:        q,
			vars:         hw{"lic": hw{"use_without_attribution": "EXCLUDE_NO"}},
			selector:     "feed_versions.#.sha1",
			selectExpect: HaCtExFvs,
		},
	}
	c, _ := newTestClient(t)
	queryTestcases(t, c, testcases)
}

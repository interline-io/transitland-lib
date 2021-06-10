package resolvers

import (
	"testing"

	"github.com/99designs/gqlgen/client"
)

func TestFeedVersionResolver(t *testing.T) {
	vars := hw{"feed_version_sha1": "d2813c293bcfd7a97dde599527ae6c62c98e66c6"}
	testcases := []testcase{
		{
			"basic",
			`query {  feed_versions {sha1} }`,
			hw{},
			``,
			"feed_versions.#.sha1",
			[]string{"e535eb2b3b9ac3ef15d82c56575e914575e732e0", "d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			"basic fields",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {sha1 url earliest_calendar_date latest_calendar_date name description} }`,
			vars,
			`{"feed_versions":[{"description":null,"earliest_calendar_date":"2017-10-02","latest_calendar_date":"2019-10-06","name":null,"sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","url":"../test/data/external/caltrain.zip"}]}`,
			"",
			nil,
		},
		// children
		{
			"feed",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed{onestop_id}} }`,
			vars,
			`{"feed_versions":[{"feed":{"onestop_id":"CT"}}]}`,
			"",
			nil,
		},
		{
			"feed_version_gtfs_import",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed_version_gtfs_import{success in_progress}} }`,
			vars,
			`{"feed_versions":[{"feed_version_gtfs_import":{"in_progress":false,"success":true}}]}`,
			"",
			nil,
		},
		{
			"feed_infos",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {feed_infos {feed_publisher_name feed_publisher_url feed_lang feed_version feed_start_date feed_end_date}} }`,
			hw{"feed_version_sha1": "e535eb2b3b9ac3ef15d82c56575e914575e732e0"}, // check BART instead
			`{"feed_versions":[{"feed_infos":[{"feed_end_date":"2019-07-01","feed_lang":"en","feed_publisher_name":"Bay Area Rapid Transit","feed_publisher_url":"http://www.bart.gov","feed_start_date":"2018-05-26","feed_version":"47"}]}]}`,
			"", nil,
		},
		{
			"files",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {files {name rows sha1 header csv_like size}} }`,
			vars,
			``,
			"feed_versions.0.files.#.name",
			[]string{"agency.txt", "calendar.txt", "calendar_attributes.txt", "calendar_dates.txt", "directions.txt", "fare_attributes.txt", "fare_rules.txt", "farezone_attributes.txt", "frequencies.txt", "realtime_routes.txt", "routes.txt", "shapes.txt", "stop_attributes.txt", "stop_times.txt", "stops.txt", "transfers.txt", "trips.txt"},
		},
		{
			"agencies",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {agencies {agency_id}} }`,
			vars,
			``,
			"feed_versions.0.agencies.#.agency_id",
			[]string{"caltrain-ca-us"},
		},
		{
			"routes",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {routes {route_id}} }`,
			vars,
			``,
			"feed_versions.0.routes.#.route_id",
			[]string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			"stops",
			`query($feed_version_sha1: String!) {  feed_versions(where:{sha1:$feed_version_sha1}) {stops {stop_id}} }`,
			vars,
			``,
			"feed_versions.0.stops.#.stop_id",
			[]string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322", "777402", "777403"},
		},
		// where
		{
			"where feed_onestop_id",
			`query{feed_versions(where:{feed_onestop_id:"CT"}) {sha1} }`,
			hw{},
			``,
			"feed_versions.#.sha1",
			[]string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
		{
			"where sha1",
			`query{feed_versions(where:{sha1:"d2813c293bcfd7a97dde599527ae6c62c98e66c6"}) {sha1} }`,
			hw{},
			``,
			"feed_versions.#.sha1",
			[]string{"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},
		},
	}
	c := client.New(NewServer())
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
}

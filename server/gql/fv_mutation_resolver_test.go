package gql

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
)

func TestFeedVersionFetchResolver(t *testing.T) {
	expectFile := testdata.Path("server/gtfs/bart.zip")
	ts200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := os.ReadFile(expectFile)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	t.Run("found sha1", func(t *testing.T) {
		testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
			srv, _ := NewServer()
			srv = model.AddConfigAndPerms(cfg, srv)
			srv = usercheck.AdminDefaultMiddleware("test")(srv) // Run all requests as admin
			// Run all requests as admin
			c := client.New(srv)
			resp := make(map[string]interface{})
			err := c.Post(`mutation($url:String!) {feed_version_fetch(feed_onestop_id:"BA",url:$url){found_sha1 feed_version{sha1}}}`, &resp, client.Var("url", ts200.URL))
			if err != nil {
				t.Error(err)
			}
			assert.JSONEq(t, `{"feed_version_fetch":{"found_sha1":true,"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}}`, toJson(resp))
		})
	})
}

func TestValidateGtfsResolver(t *testing.T) {
	ts200 := testutil.NewTestServer(testdata.Path())
	defer ts200.Close()

	vars := hw{
		"url": ts200.URL + "/server/gtfs/caltrain.zip",
	}
	testcases := []testcase{
		{
			name:   "basic",
			query:  `mutation($url:String!) {validate_gtfs(url:$url){success failure_reason details{sha1 earliest_calendar_date latest_calendar_date}}}`,
			vars:   vars,
			expect: `{"validate_gtfs":{"details":{"earliest_calendar_date":"2017-10-02","latest_calendar_date":"2019-10-06","sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},"failure_reason":null,"success":true}}`,
		},
		{
			name:         "files",
			query:        `mutation($url:String!) {validate_gtfs(url:$url){details{files{name size rows sha1 header csv_like}}}}`,
			vars:         vars,
			selector:     "validate_gtfs.details.files.#.name",
			selectExpect: []string{"agency.txt", "calendar.txt", "calendar_attributes.txt", "calendar_dates.txt", "directions.txt", "fare_attributes.txt", "fare_rules.txt", "farezone_attributes.txt", "frequencies.txt", "realtime_routes.txt", "routes.txt", "shapes.txt", "stop_attributes.txt", "stop_times.txt", "stops.txt", "transfers.txt", "trips.txt"},
		},
		{
			name:         "agencies",
			query:        `mutation($url:String!) {validate_gtfs(url:$url){details{agencies{agency_id}}}}`,
			vars:         vars,
			selector:     "validate_gtfs.details.agencies.#.agency_id",
			selectExpect: []string{"caltrain-ca-us"},
		},
		{
			name:         "routes",
			query:        `mutation($url:String!) {validate_gtfs(url:$url){details{routes{route_id}}}}`,
			vars:         vars,
			selector:     "validate_gtfs.details.routes.#.route_id",
			selectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			name:         "stops",
			query:        `mutation($url:String!) {validate_gtfs(url:$url){details{stops{stop_id}}}}`,
			vars:         vars,
			selector:     "validate_gtfs.details.stops.#.stop_id",
			selectExpect: []string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322", "777402", "777403"},
		},
		{
			name:         "feed_infos", // none present :(
			query:        `mutation($url:String!) {validate_gtfs(url:$url){details{feed_infos{feed_publisher_name}}}}`,
			vars:         vars,
			selector:     "validate_gtfs.details.feed_infos.#.feed_publisher_name",
			selectExpect: []string{},
		},
		{
			name:         "errors", // none present :(
			query:        `mutation($url:String!) {validate_gtfs(url:$url){errors{filename}}}`,
			vars:         vars,
			selector:     "validate_gtfs.errors.#.filename",
			selectExpect: []string{},
		},
		{
			name:         "warnings",
			query:        `mutation($url:String!) {validate_gtfs(url:$url){warnings{filename}}}`,
			vars:         vars,
			selector:     "validate_gtfs.warnings.#.filename",
			selectExpect: []string{"routes.txt", "routes.txt", "trips.txt"}, // now includes an additional validation warning
		},
		{
			name:         "service_levels",
			query:        `mutation($url:String!) {validate_gtfs(url:$url){details{service_levels{start_date end_date monday tuesday wednesday thursday friday saturday sunday}}}}`,
			vars:         vars,
			selector:     "validate_gtfs.details.service_levels.#.thursday",
			selectExpect: []string{"485220", "485220", "485220", "485220", "155940", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "490680", "485220", "485220", "485220", "485220"}, // todo: better checking...
		},
		// RT tests
		{
			name:  "rt1",
			query: `mutation($url:String!, $realtime_urls:[String!]) {validate_gtfs(url:$url,realtime_urls:$realtime_urls){success errors{filename error_code field}}}`,
			vars: hw{
				"url":           ts200.URL + "/server/gtfs/caltrain.zip",
				"realtime_urls": []string{ts200.URL + "/server/rt/CT-missing-trip.json"},
			},
			selector:     "validate_gtfs.errors.0.error_code",
			selectExpect: []string{"E003"},
		},
		{
			name:  "rt2",
			query: `mutation($url:String!, $realtime_urls:[String!]) {validate_gtfs(url:$url,realtime_urls:$realtime_urls){success errors{filename errors{filename field error_code message}}}}`,
			vars: hw{
				"url":           ts200.URL + "/server/gtfs/caltrain.zip",
				"realtime_urls": []string{ts200.URL + "/server/rt/CT-missing-trip.json"},
			},
			selector:     "validate_gtfs.errors.0.errors.0.error_code",
			selectExpect: []string{"E003"},
		},
		{
			name:  "rt-bad-vp",
			query: `mutation($url:String!, $realtime_urls:[String!]) {validate_gtfs(url:$url,realtime_urls:$realtime_urls){success errors{filename errors{filename field error_code message geometry}}}}`,
			vars: hw{
				"url":           ts200.URL + "/server/gtfs/caltrain.zip",
				"realtime_urls": []string{ts200.URL + "/server/rt/CT-bad-vp.json"},
			},
			selector:     "validate_gtfs.errors.0.errors.0.geometry.geometries.#.type",
			selectExpect: []string{"Point", "LineString"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
				srv, _ := NewServer()
				srv = model.AddConfigAndPerms(cfg, srv)
				srv = usercheck.UserDefaultMiddleware("test")(srv) // Run all requests as user
				c := client.New(srv)
				queryTestcase(t, c, tc)
			})
		})
	}
	// t.Run("requires user access", func(t *testing.T) {
	// 	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
	// 		srv, _ := NewServer() // all requests run as anonymous context by default
	// 		srv = model.AddConfigAndPerms(cfg, srv)
	// 		c := client.New(srv)
	// 		resp := make(map[string]interface{})
	// 		err := c.Post(`mutation($url:String!) {validate_gtfs(url:$url){success}}`, &resp, client.Var("url", ts200.URL))
	// 		if err == nil {
	// 			t.Errorf("expected error")
	// 		}
	// 	})
	// })
}

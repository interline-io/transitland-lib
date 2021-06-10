package resolvers

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/server/config"
	"github.com/stretchr/testify/assert"
)

func TestFetchResolver(t *testing.T) {
	expectFile := testutil.RelPath("test/data/external/bart.zip")
	ts200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(expectFile)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	t.Run("found sha1", func(t *testing.T) {
		srv := NewServer(config.Config{UseAuth: "admin"})
		c := client.New(srv)
		resp := make(map[string]interface{})
		err := c.Post(`mutation($url:String!) {feed_version_fetch(feed_onestop_id:"BA",url:$url){found_sha1 feed_version{sha1}}}`, &resp, client.Var("url", ts200.URL))
		if err != nil {
			t.Error(err)
		}
		assert.JSONEq(t, `{"feed_version_fetch":{"found_sha1":true,"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0"}}}`, toJson(resp))
	})
	t.Run("requires admin access", func(t *testing.T) {
		srv := NewServer(config.Config{UseAuth: "user"})
		c := client.New(srv)
		resp := make(map[string]interface{})
		err := c.Post(`mutation($url:String!) {feed_version_fetch(feed_onestop_id:"BA",url:$url){found_sha1}}`, &resp, client.Var("url", ts200.URL))
		if err == nil {
			t.Errorf("expected error")
		}
	})
}

func TestValidationResolver(t *testing.T) {
	expectFile := testutil.RelPath("test/data/external/caltrain.zip")
	ts200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(expectFile)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	c := client.New(NewServer(config.Config{UseAuth: "user"}))
	vars := hw{"url": ts200.URL}
	testcases := []testcase{
		{
			"basic",
			`mutation($url:String!) {validate_gtfs(url:$url){success failure_reason sha1 earliest_calendar_date latest_calendar_date}}`,
			vars,
			`{"validate_gtfs":{"earliest_calendar_date":"2017-10-02","failure_reason":"","latest_calendar_date":"2019-10-06","sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6","success":true}}`,
			"",
			nil,
		},
		{
			"files",
			`mutation($url:String!) {validate_gtfs(url:$url){files{name size rows sha1 header csv_like}}}`,
			vars,
			``,
			"validate_gtfs.files.#.name",
			[]string{"agency.txt", "calendar.txt", "calendar_attributes.txt", "calendar_dates.txt", "directions.txt", "fare_attributes.txt", "fare_rules.txt", "farezone_attributes.txt", "frequencies.txt", "realtime_routes.txt", "routes.txt", "shapes.txt", "stop_attributes.txt", "stop_times.txt", "stops.txt", "transfers.txt", "trips.txt"},
		},
		{
			"agencies",
			`mutation($url:String!) {validate_gtfs(url:$url){agencies{agency_id}}}`,
			vars,
			``,
			"validate_gtfs.agencies.#.agency_id",
			[]string{"caltrain-ca-us"},
		},
		{
			"routes",
			`mutation($url:String!) {validate_gtfs(url:$url){routes{route_id}}}`,
			vars,
			``,
			"validate_gtfs.routes.#.route_id",
			[]string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			"stops",
			`mutation($url:String!) {validate_gtfs(url:$url){stops{stop_id}}}`,
			vars,
			``,
			"validate_gtfs.stops.#.stop_id",
			[]string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322", "777402", "777403"},
		},
		{
			"feed_infos", // none present :(
			`mutation($url:String!) {validate_gtfs(url:$url){feed_infos{feed_publisher_name}}}`,
			vars,
			``,
			"validate_gtfs.feed_infos.#.feed_publisher_name",
			[]string{},
		},
		{
			"errors", // none present :(
			`mutation($url:String!) {validate_gtfs(url:$url){errors{filename}}}`,
			vars,
			``,
			"validate_gtfs.errors.#.filename",
			[]string{},
		},
		{
			"warnings",
			`mutation($url:String!) {validate_gtfs(url:$url){warnings{filename}}}`,
			vars,
			``,
			"validate_gtfs.warnings.#.filename",
			[]string{"routes.txt", "trips.txt"},
		},
		{
			"service_levels",
			`mutation($url:String!) {validate_gtfs(url:$url){service_levels{start_date end_date monday tuesday wednesday thursday friday saturday sunday}}}`,
			vars,
			``,
			"validate_gtfs.service_levels.#.thursday",
			[]string{"0", "165720", "165720", "125400", "165720", "165720", "165720", "165720", "165720", "165720", "165720", "15900", "89160", "89160", "89160", "89160", "89160", "89160", "89160", "89160", "89160", "230340", "230340", "230340", "230340", "230340", "230340", "230340", "230340", "0", "230340", "0", "0", "0", "0", "0", "0", "0", "0", "14640", "0", "0", "0", "5460", "0", "0", "0", "0", "0", "0"}, // todo: better checking...
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testquery(t, c, tc)
		})
	}
	t.Run("requires user access", func(t *testing.T) {
		c := client.New(NewServer(config.Config{UseAuth: ""}))
		resp := make(map[string]interface{})
		err := c.Post(`mutation($url:String!) {validate_gtfs(url:$url){success}}`, &resp, client.Var("url", ts200.URL))
		if err == nil {
			t.Errorf("expected error")
		}
	})
}

package gql

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/jobs"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
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
		testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
			srv := NewDefaultHandler()
			srv = model.AddConfigAndPerms(cfg, srv)
			srv = usercheck.AdminDefaultMiddleware("test")(srv) // Run all requests as admin
			// Run all requests as admin
			c := client.New(srv)
			resp := make(map[string]interface{})
			err := c.Post(`mutation($url:String!) {feed_version_fetch(feed_onestop_id:"BA",url:$url){found_sha1 feed_version{sha1 sha1_dir}}}`, &resp, client.Var("url", ts200.URL))
			if err != nil {
				t.Error(err)
			}
			assert.JSONEq(t, `{"feed_version_fetch":{"found_sha1":true,"feed_version":{"sha1":"e535eb2b3b9ac3ef15d82c56575e914575e732e0","sha1_dir":"9f0f389f9195114f964da269f35f753b3d8ca1c8"}}}`, gqlcase.ToJSON(resp))
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
			Name:   "basic",
			Query:  `mutation($url:String!) {validate_gtfs(url:$url){success failure_reason details{sha1 earliest_calendar_date latest_calendar_date}}}`,
			Vars:   vars,
			Expect: `{"validate_gtfs":{"details":{"earliest_calendar_date":"2017-10-02","latest_calendar_date":"2019-10-06","sha1":"d2813c293bcfd7a97dde599527ae6c62c98e66c6"},"failure_reason":null,"success":true}}`,
		},
		{
			Name:         "files",
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){details{files{name size rows sha1 header csv_like}}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.details.files.#.name",
			SelectExpect: []string{"agency.txt", "calendar.txt", "calendar_attributes.txt", "calendar_dates.txt", "directions.txt", "fare_attributes.txt", "fare_rules.txt", "farezone_attributes.txt", "frequencies.txt", "realtime_routes.txt", "routes.txt", "shapes.txt", "stop_attributes.txt", "stop_times.txt", "stops.txt", "transfers.txt", "trips.txt"},
		},
		{
			Name:         "agencies",
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){details{agencies{agency_id}}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.details.agencies.#.agency_id",
			SelectExpect: []string{"caltrain-ca-us"},
		},
		{
			Name:         "routes",
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){details{routes{route_id}}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.details.routes.#.route_id",
			SelectExpect: []string{"Bu-130", "Li-130", "Lo-130", "TaSj-130", "Gi-130", "Sp-130"},
		},
		{
			Name:         "stops",
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){details{stops{stop_id}}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.details.stops.#.stop_id",
			SelectExpect: []string{"70011", "70012", "70021", "70022", "70031", "70032", "70041", "70042", "70051", "70052", "70061", "70062", "70071", "70072", "70081", "70082", "70091", "70092", "70101", "70102", "70111", "70112", "70121", "70122", "70131", "70132", "70141", "70142", "70151", "70152", "70161", "70162", "70171", "70172", "70191", "70192", "70201", "70202", "70211", "70212", "70221", "70222", "70231", "70232", "70241", "70242", "70251", "70252", "70261", "70262", "70271", "70272", "70281", "70282", "70291", "70292", "70301", "70302", "70311", "70312", "70321", "70322", "777402", "777403"},
		},
		{
			Name:         "feed_infos", // none present :(
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){details{feed_infos{feed_publisher_name}}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.details.feed_infos.#.feed_publisher_name",
			SelectExpect: []string{},
		},
		{
			Name:         "errors", // none present :(
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){errors{filename}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.errors.#.filename",
			SelectExpect: []string{},
		},
		{
			Name:         "warnings",
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){warnings{filename}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.warnings.#.filename",
			SelectExpect: []string{"routes.txt", "routes.txt", "trips.txt"}, // now includes an additional validation warning
		},
		{
			Name:         "service_levels",
			Query:        `mutation($url:String!) {validate_gtfs(url:$url){details{service_levels{start_date end_date monday tuesday wednesday thursday friday saturday sunday}}}}`,
			Vars:         vars,
			Selector:     "validate_gtfs.details.service_levels.#.thursday",
			SelectExpect: []string{"485220", "485220", "485220", "485220", "155940", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "485220", "490680", "485220", "485220", "485220", "485220"}, // todo: better checking...
		},
		// RT tests
		{
			Name:  "rt1",
			Query: `mutation($url:String!, $realtime_urls:[String!]) {validate_gtfs(url:$url,realtime_urls:$realtime_urls){success errors{filename error_code field}}}`,
			Vars: hw{
				"url":           ts200.URL + "/server/gtfs/caltrain.zip",
				"realtime_urls": []string{ts200.URL + "/server/rt/CT-missing-trip.json"},
			},
			Selector:     "validate_gtfs.errors.0.error_code",
			SelectExpect: []string{"E003"},
		},
		{
			Name:  "rt2",
			Query: `mutation($url:String!, $realtime_urls:[String!]) {validate_gtfs(url:$url,realtime_urls:$realtime_urls){success errors{filename errors{filename field error_code message}}}}`,
			Vars: hw{
				"url":           ts200.URL + "/server/gtfs/caltrain.zip",
				"realtime_urls": []string{ts200.URL + "/server/rt/CT-missing-trip.json"},
			},
			Selector:     "validate_gtfs.errors.0.errors.0.error_code",
			SelectExpect: []string{"E003"},
		},
		{
			Name:  "rt-bad-vp",
			Query: `mutation($url:String!, $realtime_urls:[String!]) {validate_gtfs(url:$url,realtime_urls:$realtime_urls){success errors{filename errors{filename field error_code message geometry}}}}`,
			Vars: hw{
				"url":           ts200.URL + "/server/gtfs/caltrain.zip",
				"realtime_urls": []string{ts200.URL + "/server/rt/CT-bad-vp.json"},
			},
			Selector:     "validate_gtfs.errors.0.errors.0.geometry.geometries.#.type",
			SelectExpect: []string{"Point", "LineString"},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.Name, func(t *testing.T) {
			testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
				srv := NewDefaultHandler()
				srv = model.AddConfigAndPerms(cfg, srv)
				srv = usercheck.UserDefaultMiddleware("test")(srv) // Run all requests as user
				c := client.New(srv)
				gqlcase.QueryTestcase(t, c, tc)
			})
		})
	}
	// t.Run("requires user access", func(t *testing.T) {
	// 	testconfig.ConfigTxRollback(t, testconfig.Options{}, func(cfg model.Config) {
	// 		srv := NewDefaultHandler() // all requests run as anonymous context by default
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

// stubJobWorker stands in for the deployment's real import/unimport worker so
// the resolver's submit-and-watch path can be tested without running an import.
type stubJobWorker struct {
	kind string
	err  error
}

func (w *stubJobWorker) Kind() string                  { return w.kind }
func (w *stubJobWorker) Run(ctx context.Context) error { return w.err }

// TestFeedVersionImportEnqueue verifies the import/unimport mutations enqueue a
// job and report the job's terminal state as their synchronous result.
func TestFeedVersionImportEnqueue(t *testing.T) {
	cases := []struct {
		name        string
		kind        string
		query       string
		field       string
		workerErr   error
		wantSuccess bool
		wantErr     bool
	}{
		{"import success", "feed-version-import", `mutation { feed_version_import(id:1){success} }`, "feed_version_import", nil, true, false},
		{"unimport success", "feed-version-unimport", `mutation { feed_version_unimport(id:1){success} }`, "feed_version_unimport", nil, true, false},
		{"import failure", "feed-version-import", `mutation { feed_version_import(id:1){success} }`, "feed_version_import", errors.New("import boom"), false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testconfig.ConfigTxRollback(t, testconfig.Options{AllowAll: true}, func(cfg model.Config) {
				if err := cfg.JobRunner.Register(func() jobs.Worker {
					return &stubJobWorker{kind: tc.kind, err: tc.workerErr}
				}); err != nil {
					t.Fatal(err)
				}
				runCtx, cancel := context.WithCancel(context.Background())
				defer cancel()
				go func() { _ = cfg.Jobs.Run(runCtx) }()

				srv := NewDefaultHandler()
				srv = model.AddConfigAndPerms(cfg, srv)
				srv = usercheck.AdminDefaultMiddleware("test")(srv)
				c := client.New(srv)
				resp := make(map[string]interface{})
				err := c.Post(tc.query, &resp)
				if tc.wantErr {
					assert.Error(t, err)
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				got := gjson.Get(gqlcase.ToJSON(resp), tc.field+".success").Bool()
				assert.Equal(t, tc.wantSuccess, got)
			})
		})
	}
}

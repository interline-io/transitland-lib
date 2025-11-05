package gql

import (
	"encoding/json"
	"log"
	"os"
	"testing"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

const DEFAULT_WHEN = "2022-09-01T00:00:00Z"

type hw = map[string]interface{}

type testcaseSelector struct {
	selector     string
	expect       []string
	expectUnique []string
	expectCount  int
}

type testcase struct {
	name               string
	query              string
	vars               hw
	expect             string
	user               string
	selector           string
	expectError        bool
	selectExpect       []string
	selectExpectUnique []string
	selectExpectCount  int
	sel                []testcaseSelector
	f                  func(*testing.T, string)
}

type testcaseWithClock struct {
	testcase
	whenUtc string
}

func TestMain(m *testing.M) {
	// Increase default limit for testing purposes
	RESOLVER_MAXLIMIT = 100_000
	if a, ok := testutil.CheckTestDB(); !ok {
		log.Print(a)
		return
	}
	os.Exit(m.Run())
}

// Test helpers

func newTestClient(t testing.TB) (*client.Client, model.Config) {
	return newTestClientWithOpts(t, testconfig.Options{
		RTJsons: testconfig.DefaultRTJson(),
	})
}

func newTestClientWithOpts(t testing.TB, opts testconfig.Options) (*client.Client, model.Config) {
	if opts.WhenUtc == "" {
		opts.WhenUtc = DEFAULT_WHEN
	}
	cfg := testconfig.Config(t, opts)
	srv, _ := NewServer()
	graphqlServer := model.AddConfigAndPerms(cfg, srv)
	srvMiddleware := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})
	return client.New(srvMiddleware(graphqlServer)), cfg
}

func toJson(m map[string]interface{}) string {
	rr, _ := json.Marshal(&m)
	return string(rr)
}

func queryTestcase(t *testing.T, c *client.Client, tc testcase) {
	tested := false
	var resp map[string]interface{}
	opts := []client.Option{}
	for k, v := range tc.vars {
		opts = append(opts, client.Var(k, v))
	}
	if err := c.Post(tc.query, &resp, opts...); err != nil {
		if tc.expectError {
			// ok
		} else {
			t.Error(err)
			return
		}
	} else if tc.expectError {
		t.Error("expected error")
	}
	jj := toJson(resp)
	if tc.expect != "" {
		tested = true
		if !assert.JSONEq(t, tc.expect, jj) {
			t.Errorf("got %s -- expect %s\n", jj, tc.expect)
		}
	}
	if tc.f != nil {
		tested = true
		tc.f(t, jj)
	}
	if tc.selector != "" {
		tc.sel = append(tc.sel, testcaseSelector{
			selector:     tc.selector,
			expect:       tc.selectExpect,
			expectCount:  tc.selectExpectCount,
			expectUnique: tc.selectExpectUnique,
		})
	}
	for _, sel := range tc.sel {
		a := []string{}
		for _, v := range gjson.Get(jj, sel.selector).Array() {
			a = append(a, v.String())
		}
		if sel.expectCount != 0 {
			tested = true
			if len(a) != sel.expectCount {
				t.Errorf("selector returned %d elements, expected %d", len(a), sel.expectCount)
			}
		}
		if sel.expectUnique != nil {
			tested = true
			mm := map[string]int{}
			for _, v := range a {
				mm[v] += 1
			}
			var keys []string
			for k := range mm {
				keys = append(keys, k)
			}
			assert.ElementsMatch(t, sel.expectUnique, keys)
		}
		if sel.expect != nil {
			tested = true
			if !assert.ElementsMatch(t, sel.expect, a) {
				t.Errorf("got %#v -- expect %#v\n\n", a, sel.expect)
			}
		}
	}
	if !tested {
		t.Errorf("no test performed, check test case")
	}
}

func queryTestcases(t *testing.T, c *client.Client, tcs []testcase) {
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			queryTestcase(t, c, tc)
		})
	}
}

func benchmarkTestcases(b *testing.B, c *client.Client, tcs []testcase) {
	for _, tc := range tcs {
		b.Run(tc.name, func(b *testing.B) {
			benchmarkTestcase(b, c, tc)
		})
	}
}

func benchmarkTestcase(b *testing.B, c *client.Client, tc testcase) {
	opts := []client.Option{}
	for k, v := range tc.vars {
		opts = append(opts, client.Var(k, v))
	}
	var resp map[string]any
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		c.MustPost(tc.query, &resp, opts...)
	}
}

func Test_checkBbox(t *testing.T) {
	tcs := []struct {
		name   string
		bbox   model.BoundingBox
		area   float64
		expect bool
	}{
		{
			name:   "small 1",
			bbox:   model.BoundingBox{MinLon: -122.27023950636408, MinLat: 37.80659174954722, MaxLon: -122.26747210439623, MaxLat: 37.80928736232528},
			area:   0.08 * 1e6,
			expect: true,
		},
		{
			name:   "small 2",
			bbox:   model.BoundingBox{MinLon: -122.27023950636408, MinLat: 37.80659174954722, MaxLon: -122.26747210439623, MaxLat: 37.80928736232528},
			area:   0.07 * 1e6,
			expect: false,
		},
		{
			name:   "big 1",
			bbox:   model.BoundingBox{MinLon: -123.08182748515924, MinLat: 37.100203650623826, MaxLon: -121.31739022765265, MaxLat: 38.31972646345332},
			area:   22000.0 * 1e6,
			expect: true,
		},
		{
			name:   "big 2",
			bbox:   model.BoundingBox{MinLon: -123.08182748515924, MinLat: 37.100203650623826, MaxLon: -121.31739022765265, MaxLat: 38.31972646345332},
			area:   20000.0 * 1e6,
			expect: false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			v := checkBbox(&tc.bbox, tc.area)
			assert.Equal(t, tc.expect, v, "expected result")

		})
	}
}

func astr(a []gjson.Result) []string {
	var ret []string
	for _, b := range a {
		ret = append(ret, b.String())
	}
	return ret
}

// RT helpers

// Additional tests for RT data on StopResolver
var rtTestStopQuery = `
fragment alert on Alert {
	cause
	effect
	severity_level
	url {
		language
		text
	}
	header_text {
		language
		text
	}
	description_text {
		language
		text
	}
	tts_header_text {
		language
		text
	}
	tts_description_text {
		language
		text
	}
	active_period {
		start
		end
	}
}

query($stop_id:String!, $stf:StopTimeFilter!, $active:Boolean) {
	stops(where: { stop_id: $stop_id }) {
	  id
	  stop_id
	  stop_name
	  alerts(active:$active, limit:5) {
		  ...alert
	  }
	  stop_times(where:$stf) {
		stop_sequence
		service_date
		date
		schedule_relationship
		trip {
		  alerts(active:$active) {
			...alert
		  }
		  trip_id
		  schedule_relationship
		  timestamp
		  route {
			  route_id
			  route_short_name
			  route_long_name
			  alerts(active:$active) {
				  ...alert
			  }
			  agency {
				  agency_id
				  agency_name
				  alerts(active:$active) {
					  ...alert
				  }
			  }
		  }
		}
		arrival {
			scheduled
			scheduled_local
			scheduled_utc
			scheduled_unix
			estimated
			estimated_utc
			estimated_unix
			estimated_local
			estimated_delay
			stop_timezone
			time_unix
			delay
			uncertainty
		}
		departure {
			scheduled
			scheduled_local
			scheduled_utc
			scheduled_unix
			estimated
			estimated_utc
			estimated_unix
			estimated_local
			estimated_delay
			stop_timezone
			time_unix
			delay
			uncertainty
		}
	  }
	}
  }
`

func rtTestStopQueryVars() hw {
	return hw{
		"stop_id": "FTVL",
		"stf": hw{
			"service_date": "2018-05-30",
			"start_time":   57600,
			"end_time":     57900,
		},
	}
}

type rtTestCase struct {
	name    string
	query   string
	vars    map[string]interface{}
	rtfiles []testconfig.RTJsonFile
	cb      func(t *testing.T, jj string)
	whenUtc string
}

func testRt(t *testing.T, tc rtTestCase) {
	t.Run(tc.name, func(t *testing.T) {
		// Create a new RT Finder for each test...
		c, _ := newTestClientWithOpts(t, testconfig.Options{
			WhenUtc: tc.whenUtc,
			RTJsons: tc.rtfiles,
		})
		var resp map[string]interface{}
		opts := []client.Option{}
		for k, v := range tc.vars {
			opts = append(opts, client.Var(k, v))
		}
		if err := c.Post(tc.query, &resp, opts...); err != nil {
			t.Error(err)
			return
		}
		jj := toJson(resp)
		if tc.cb != nil {
			tc.cb(t, jj)
		}
	})
}

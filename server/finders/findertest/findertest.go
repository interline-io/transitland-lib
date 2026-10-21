// Package findertest runs GraphQL resolver test cases against any model.Finder
// loaded with the server test feeds.
//
// A Finder implementation calls Run from its own tests, so every implementation
// answers the same queries with the same results. The cases expect the feeds
// that testdata/test_setup.sh loads into the test database.
package findertest

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// NewConfig returns a server config whose finder holds the server test feeds.
type NewConfig func(t testing.TB) model.Config

// caseGroup is one resolver test function's cases, run against one client.
type caseGroup struct {
	name  string
	cases func() []testcase
}

// clockGroup is one resolver test function's cases, each run at its own time.
type clockGroup struct {
	name  string
	cases func() []testcaseWithClock
}

// The resolver test functions the suite runs, by name.
var (
	caseGroups = []caseGroup{
		{"BookingRuleResolver", bookingRuleResolverTestcases},
		{"BookingRuleResolver_FeedVersion", bookingRuleResolverFeedVersionTestcases},
		{"BookingRuleResolver_PriorNoticeService", bookingRuleResolverPriorNoticeServiceTestcases},
		{"CalendarResolver", calendarResolverTestcases},
		{"FeedVersionResolver_Locations", feedVersionResolverLocationsTestcases},
		{"FeedVersionResolver_BookingRules", feedVersionResolverBookingRulesTestcases},
		{"FeedVersionResolver_LocationGroups", feedVersionResolverLocationGroupsTestcases},
		{"LevelResolver", levelResolverTestcases},
		{"FeedVersionResolver_Levels", feedVersionResolverLevelsTestcases},
		{"LocationResolver", locationResolverTestcases},
		{"LocationResolver_StopTimes", locationResolverStopTimesTestcases},
		{"LocationGroupResolver", locationGroupResolverTestcases},
		{"LocationGroupResolver_StopTimes", locationGroupResolverStopTimesTestcases},
		{"LocationGroupResolver_Stops", locationGroupResolverStopsTestcases},
		{"LocationGroupResolver_FeedVersion", locationGroupResolverFeedVersionTestcases},
		{"FeedVersionResolver_LocationGroupStops", feedVersionResolverLocationGroupStopsTestcases},
		{"PathwayResolver", pathwayResolverTestcases},
		{"FeedVersionResolver_Pathways", feedVersionResolverPathwaysTestcases},
		{"RouteResolver_License", routeResolverLicenseTestcases},
		{"FeedVersionResolver_Shapes", feedVersionResolverShapesTestcases},
		{"ShapeResolver_Trips", shapeResolverTripsTestcases},
		{"StopResolver_StopTimes_Dates", stopResolverStopTimesDatesTestcases},
		{"StopResolver_StopTimes_DateWindows", stopResolverStopTimesDateWindowsTestcases},
		{"StopResolver_LocationGroups", stopResolverLocationGroupsTestcases},
		{"TripResolver", tripResolverTestcases},
		{"TripResolver_Frequencies", tripResolverFrequenciesTestcases},
		{"TripResolver_License", tripResolverLicenseTestcases},
		{"TripResolver_FlexStopTimes", tripResolverFlexStopTimesTestcases},
		{"TripResolver_SafeDuration", tripResolverSafeDurationTestcases},
		{"FeedVersionResolver_Frequencies", feedVersionResolverFrequenciesTestcases},
		{"AgencyResolver_License", agencyResolverLicenseTestcases},
		{"OperatorResolver_License", operatorResolverLicenseTestcases},
	}
	clockGroups = []clockGroup{
		{"RouteResolver_Date", routeResolverDateTestcases},
	}
)

// Run runs every case against the finder that newConfig builds, setting the
// config's clock to the time the case runs at. Some cases read more rows than
// the API allows by default, so callers first raise gql.RESOLVER_MAXLIMIT, as
// the resolver tests do.
func Run(t *testing.T, newConfig NewConfig) {
	for _, g := range caseGroups {
		t.Run(g.name, func(t *testing.T) {
			queryTestcases(t, newTestClient(t, newConfig, defaultWhen), g.cases())
		})
	}
	for _, g := range clockGroups {
		t.Run(g.name, func(t *testing.T) {
			for _, tc := range g.cases() {
				t.Run(tc.name, func(t *testing.T) {
					queryTestcase(t, newTestClient(t, newConfig, tc.whenUtc), tc.testcase)
				})
			}
		})
	}
}

// newTestClient serves the config through the GraphQL handler, as a signed-in
// test user.
func newTestClient(t testing.TB, newConfig NewConfig, whenUtc string) *client.Client {
	when, err := time.Parse(time.RFC3339, whenUtc)
	if err != nil {
		t.Fatal(err)
	}
	cfg := newConfig(t)
	cfg.Clock = &clock.Mock{T: when}
	srv := gql.NewDefaultHandler()
	graphqlServer := model.AddConfigAndPerms(cfg, srv)
	srvMiddleware := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})
	return client.New(srvMiddleware(graphqlServer))
}

// defaultWhen is the time a case runs at unless it sets its own.
const defaultWhen = "2022-09-01T00:00:00Z"

type hw = map[string]interface{}

type testcaseSelector struct {
	selector          string
	expect            []string
	expectUnique      []string
	expectContains    []string
	expectCount       int
	expectUniqueCount int
}

// testcase defines a GraphQL test case. See server/gql/RESOLVER_TEST_GUIDE.md for full documentation.
type testcase struct {
	name        string
	query       string
	vars        hw
	expect      string
	expectError bool
	// expectErrorContains expects an error containing this text
	expectErrorContains string
	// sel allows multiple selector checks against the same query result
	sel []testcaseSelector
	// f is a custom test function for complex assertions
	f func(*testing.T, string)
	// Convenience fields: these wrap to sel with a single testcaseSelector.
	// Use these for simple single-selector tests; use sel[] for multiple checks.
	selector                string   // convenience: wraps to sel[].selector
	selectExpect            []string // convenience: wraps to sel[].expect
	selectExpectUnique      []string // convenience: wraps to sel[].expectUnique
	selectExpectContains    []string // convenience: wraps to sel[].expectContains
	selectExpectCount       int      // convenience: wraps to sel[].expectCount
	selectExpectUniqueCount int      // convenience: wraps to sel[].expectUniqueCount
}

type testcaseWithClock struct {
	testcase
	whenUtc string
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
	expectError := tc.expectError || tc.expectErrorContains != ""
	if err := c.Post(tc.query, &resp, opts...); err != nil {
		if expectError {
			tested = true // expectError counts as a test
			if tc.expectErrorContains != "" {
				assert.ErrorContains(t, err, tc.expectErrorContains)
			}
		} else {
			t.Error(err)
			return
		}
	} else if expectError {
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
			selector:          tc.selector,
			expect:            tc.selectExpect,
			expectCount:       tc.selectExpectCount,
			expectUnique:      tc.selectExpectUnique,
			expectContains:    tc.selectExpectContains,
			expectUniqueCount: tc.selectExpectUniqueCount,
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
		if sel.expectUniqueCount != 0 {
			tested = true
			mm := map[string]int{}
			for _, v := range a {
				mm[v] += 1
			}
			if len(mm) != sel.expectUniqueCount {
				t.Errorf("selector returned %d unique elements, expected %d", len(mm), sel.expectUniqueCount)
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
		if sel.expectContains != nil {
			tested = true
			mm := map[string]bool{}
			for _, v := range a {
				mm[v] = true
			}
			for _, v := range sel.expectContains {
				if !mm[v] {
					t.Errorf("expected result to contain %q", v)
				}
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

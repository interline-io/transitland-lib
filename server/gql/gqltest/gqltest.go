// Package gqltest runs GraphQL resolver test cases against any model.Finder
// loaded with the server test feeds.
//
// A Finder implementation calls Run from its own tests, so every implementation
// answers the same queries with the same results. The cases expect the feeds
// that testdata/test_setup.sh loads into the test database.
package gqltest

import (
	"context"
	"encoding/json"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/99designs/gqlgen/client"
	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/auth/authn"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
)

// NewConfig returns a server config whose finder holds the server test feeds.
// Finders that keep their own clock read the time now.
type NewConfig func(t testing.TB, now time.Time) model.Config

// Capability names an optional feature of a finder. Only finders that claim a
// capability run the cases that need one.
type Capability string

// Spatial is filtering feeds, feed versions, operators and agencies by bbox,
// within and near, against their polygon geometries.
const Spatial Capability = "Spatial"

// Realtime is matching GTFS-RT trip updates, alerts and vehicle positions to the
// finder's static data, through the config's RTFinder.
const Realtime Capability = "Realtime"

// caseGroup is a named set of cases, run against one client. The cases are
// built from the config of the finder under test, so they can look up ids, and
// configure, if set, sets the config's options for the group.
type caseGroup struct {
	name      string
	cases     func(t testing.TB, cfg model.Config) []testcase
	configure func(*model.Config)
}

// clockGroup is a named set of cases, each run at its own time.
type clockGroup struct {
	name  string
	cases func() []testcaseWithClock
}

// testGroup is a named test that queries the client itself, for checks that
// span several queries.
type testGroup struct {
	name string
	run  func(t *testing.T, c *client.Client, cfg model.Config)
}

// rtGroup is a named realtime test, which builds a client for each case with
// that case's realtime fixtures and time.
type rtGroup struct {
	name string
	run  func(t *testing.T, newConfig NewConfig)
}

// The case groups the suite runs.
var (
	caseGroups = []caseGroup{
		{name: "BookingRuleResolver", cases: bookingRuleResolverTestcases},
		{name: "BookingRuleResolver_FeedVersion", cases: bookingRuleResolverFeedVersionTestcases},
		{name: "BookingRuleResolver_PriorNoticeService", cases: bookingRuleResolverPriorNoticeServiceTestcases},
		{name: "CalendarResolver", cases: calendarResolverTestcases},
		{name: "FeedVersionResolver_Locations", cases: feedVersionResolverLocationsTestcases},
		{name: "FeedVersionResolver_BookingRules", cases: feedVersionResolverBookingRulesTestcases},
		{name: "FeedVersionResolver_LocationGroups", cases: feedVersionResolverLocationGroupsTestcases},
		{name: "LevelResolver", cases: levelResolverTestcases},
		{name: "FeedVersionResolver_Levels", cases: feedVersionResolverLevelsTestcases},
		{name: "LocationResolver", cases: locationResolverTestcases},
		{name: "LocationResolver_StopTimes", cases: locationResolverStopTimesTestcases},
		{name: "LocationGroupResolver", cases: locationGroupResolverTestcases},
		{name: "LocationGroupResolver_StopTimes", cases: locationGroupResolverStopTimesTestcases},
		{name: "LocationGroupResolver_Stops", cases: locationGroupResolverStopsTestcases},
		{name: "LocationGroupResolver_FeedVersion", cases: locationGroupResolverFeedVersionTestcases},
		{name: "FeedVersionResolver_LocationGroupStops", cases: feedVersionResolverLocationGroupStopsTestcases},
		{name: "PathwayResolver", cases: pathwayResolverTestcases},
		{name: "FeedVersionResolver_Pathways", cases: feedVersionResolverPathwaysTestcases},
		{name: "RouteResolver_License", cases: routeResolverLicenseTestcases},
		{name: "FeedVersionResolver_Shapes", cases: feedVersionResolverShapesTestcases},
		{name: "ShapeResolver_Trips", cases: shapeResolverTripsTestcases},
		{name: "StopResolver_StopTimes_Dates", cases: stopResolverStopTimesDatesTestcases},
		{name: "StopResolver_StopTimes_DateWindows", cases: stopResolverStopTimesDateWindowsTestcases},
		{name: "StopResolver_LocationGroups", cases: stopResolverLocationGroupsTestcases},
		{name: "TripResolver", cases: tripResolverTestcases},
		{name: "TripResolver_Frequencies", cases: tripResolverFrequenciesTestcases},
		{name: "TripResolver_License", cases: tripResolverLicenseTestcases},
		{name: "TripResolver_FlexStopTimes", cases: tripResolverFlexStopTimesTestcases},
		{name: "TripResolver_SafeDuration", cases: tripResolverSafeDurationTestcases},
		{name: "FeedVersionResolver_Frequencies", cases: feedVersionResolverFrequenciesTestcases},
		{name: "AgencyResolver_License", cases: agencyResolverLicenseTestcases},
		{name: "OperatorResolver_License", cases: operatorResolverLicenseTestcases},
		{name: "AgencyResolver", cases: agencyResolverTestcases},
		{name: "AgencyResolver_Stops", cases: agencyResolverStopsTestcases},
		{name: "FareResolver", cases: fareResolverTestcases},
		{name: "FeedVersionResolver_StopAreas", cases: feedVersionResolverStopAreasTestcases},
		{name: "FeedVersionResolver_Translations", cases: feedVersionResolverTranslationsTestcases},
		{name: "FeedVersionResolver_Attributions", cases: feedVersionResolverAttributionsTestcases},
		{name: "FeedVersionResolver_Transfers", cases: feedVersionResolverTransfersTestcases},
		{name: "FeedResolver", cases: feedResolverTestcases},
		{name: "FeedResolver_License", cases: feedResolverLicenseTestcases},
		{name: "FeedVersionResolver", cases: feedVersionResolverTestcases},
		{name: "FeedVersionResolver_License", cases: feedVersionResolverLicenseTestcases},
		{name: "OperatorResolver", cases: operatorResolverTestcases},
		{name: "RouteResolver", cases: routeResolverTestcases},
		{name: "AgencyResolver_Cursor", cases: agencyResolverCursorTestcases},
		{name: "FareResolver_Cursor", cases: fareResolverCursorTestcases},
		{name: "AgencyResolver_StopsCursor", cases: agencyResolverStopsCursorTestcases},
		{name: "FeedResolver_Cursor", cases: feedResolverCursorTestcases},
		{name: "FeedVersionResolver_PathwaysAndLevels_Cursor", cases: feedVersionResolverPathwaysAndLevelsCursorTestcases},
		{name: "RouteResolver_Cursor", cases: routeResolverCursorTestcases},
		{name: "FeedVersionResolver_Shapes_Cursor", cases: feedVersionResolverShapesCursorTestcases},
		{name: "StopResolverCursor", cases: stopResolverCursorTestcases},
		{name: "StopResolverLicense", cases: stopResolverLicenseTestcases},
		{name: "StopResolver", cases: stopResolverTestcases},
		{name: "StopResolverLocation", cases: stopResolverLocationTestcases},
		{name: "RouteResolver_Location", cases: routeResolverLocationTestcases},
		{name: "AgencyResolver_Location", cases: agencyResolverLocationTestcases},
		{name: "QueryResolver", cases: queryResolverTestcases},
		{name: "StopResolver_StopTimes", cases: stopResolverStopTimesTestcases},
	}
	optionalGroups = map[Capability][]caseGroup{
		Spatial: {
			{name: "FeedResolver", cases: feedResolverSpatialTestcases},
			{name: "FeedVersionResolver", cases: feedVersionResolverSpatialTestcases},
			{name: "OperatorResolver", cases: operatorResolverSpatialTestcases},
			{name: "AgencyResolver_Location", cases: agencyResolverLocationSpatialTestcases},
			{name: "FeedResolver_GeohashFilter", cases: feedResolverGeohashFilterSpatialTestcases, configure: useGeohashFilter},
			{name: "FeedVersionResolver_GeohashFilter", cases: feedVersionResolverGeohashFilterSpatialTestcases, configure: useGeohashFilter},
		},
	}
	clockGroups = []clockGroup{
		{name: "RouteResolver_Date", cases: routeResolverDateTestcases},
		{name: "StopResolver_StopTimes_RelativeDates", cases: stopResolverStopTimesRelativeDatesTestcases},
		{name: "StopResolver_StopTimes_Next", cases: stopResolverStopTimesNextTestcases},
		{name: "StopResolver_StopTimes_Frequencies", cases: stopResolverStopTimesFrequenciesTestcases},
		{name: "FeedVersionResolver_Trips_Date", cases: feedVersionResolverTripsDateTestcases},
	}
	testGroups = []testGroup{
		{name: "TripResolver_StopPatternID", run: tripResolverStopPatternIDTest},
		{name: "RouteResolver_TripsAcrossFeedVersions", run: routeResolverTripsAcrossFeedVersionsTest},
		{name: "StopResolver_ServiceWindowIsPerFeedVersion", run: stopResolverServiceWindowIsPerFeedVersionTest},
		{name: "FeedVersionResolver_TableCursors", run: feedVersionResolverTableCursorsTest},
	}
	rtGroups = []rtGroup{
		{name: "AgencyRT_Alerts", run: agencyRTAlertsTest},
		{name: "AgencyRT_AlertsIncludeModes", run: agencyRTAlertsIncludeModesTest},
		{name: "RouteRT_Alerts", run: routeRTAlertsTest},
		{name: "RouteRT_AlertsAtStop", run: routeRTAlertsAtStopTest},
		{name: "RouteRT_TripAlerts", run: routeRTTripAlertsTest},
		{name: "StopRT_Basic", run: stopRTBasicTest},
		{name: "StopRT_BeforeMidnight", run: stopRTBeforeMidnightTest},
		{name: "StopRT_AfterMidnight", run: stopRTAfterMidnightTest},
		{name: "StopRT_ArrivalFallback", run: stopRTArrivalFallbackTest},
		{name: "StopRT_DepartureFallback", run: stopRTDepartureFallbackTest},
		{name: "StopRT_CarriedDelay", run: stopRTCarriedDelayTest},
		{name: "StopRT_StopIDFallback", run: stopRTStopIDFallbackTest},
		{name: "StopRT_StopIDFallback_DoubleVisit", run: stopRTStopIDFallbackDoubleVisitTest},
		{name: "StopRT_SkippedAndNoData", run: stopRTSkippedAndNoDataTest},
		{name: "StopRT_FrequencyInstances", run: stopRTFrequencyInstancesTest},
		{name: "StopRT_NoRT", run: stopRTNoRTTest},
		{name: "StopRT_AddedTrip", run: stopRTAddedTripTest},
		{name: "StopRT_ScheduleRelationship", run: stopRTScheduleRelationshipTest},
		{name: "StopRT_CanceledTrip", run: stopRTCanceledTripTest},
		{name: "StopRT_Alerts", run: stopRTAlertsTest},
		{name: "StopRT_ScheduleRelationshipWithoutArrivalDeparture", run: stopRTScheduleRelationshipWithoutArrivalDepartureTest},
		{name: "StopRT_StalePayload", run: stopRTStalePayloadTest},
		{name: "StopRT_AddedTripUnknownRoute", run: stopRTAddedTripUnknownRouteTest},
		{name: "TripRT_Alerts", run: tripRTAlertsTest},
		{name: "TripRT_StopTimes", run: tripRTStopTimesTest},
		{name: "TripRT_ScheduleRelationshipWithoutArrivalDeparture", run: tripRTScheduleRelationshipWithoutArrivalDepartureTest},
		{name: "TripRT_Runs_Departures", run: tripRTRunsDeparturesTest},
		{name: "TripRT_Runs_FallbackScheduledTime", run: tripRTRunsFallbackScheduledTimeTest},
		{name: "TripRT_Runs_Trips", run: tripRTRunsTripsTest},
		{name: "TripRT_Runs_AddedTrips", run: tripRTRunsAddedTripsTest},
		{name: "TripRT_Runs_PatternTrips", run: tripRTRunsPatternTripsTest},
		{name: "TripRT_Runs_RelativeDates", run: tripRTRunsRelativeDatesTest},
		{name: "VehiclePositionResolver", run: vehiclePositionResolverTest},
		{name: "VehiclePositionResolver_OrderAndLimit", run: vehiclePositionResolverOrderAndLimitTest},
		{name: "VehiclePositionResolver_Fields", run: vehiclePositionResolverFieldsTest},
		{name: "VehiclePositionResolver_MatchedEntities", run: vehiclePositionResolverMatchedEntitiesTest},
		{name: "VehiclePositionResolver_RouteFromTrip", run: vehiclePositionResolverRouteFromTripTest},
		{name: "VehiclePositionResolver_Nested", run: vehiclePositionResolverNestedTest},
		{name: "VehiclePositionResolver_BboxTooLarge", run: vehiclePositionResolverBboxTooLargeTest},
		{name: "VehiclePositionResolver_TripCopies", run: vehiclePositionResolverTripCopiesTest},
	}
)

// Run runs the suite against the finder that newConfig builds, skipping the
// cases that need a capability missing from supports.
//
// Each client's config reads the time at which its cases run, holds the
// realtime fixtures, and has the search radius limit the cases expect. Run
// raises gql.RESOLVER_MAXLIMIT while it runs, since some cases read more rows
// than the API allows by default.
func Run(t *testing.T, newConfig NewConfig, supports ...Capability) {
	raiseMaxLimit(t)
	for _, g := range caseGroups {
		runCaseGroup(t, newConfig, g)
	}
	for _, capability := range slices.Sorted(maps.Keys(optionalGroups)) {
		t.Run(string(capability), func(t *testing.T) {
			if !slices.Contains(supports, capability) {
				t.Skipf("finder does not support %s", capability)
			}
			for _, g := range optionalGroups[capability] {
				runCaseGroup(t, newConfig, g)
			}
		})
	}
	for _, g := range clockGroups {
		runClockGroup(t, newConfig, g)
	}
	for _, g := range testGroups {
		runTestGroup(t, newConfig, g)
	}
	t.Run(string(Realtime), func(t *testing.T) {
		if !slices.Contains(supports, Realtime) {
			t.Skipf("finder does not support %s", Realtime)
		}
		for _, g := range rtGroups {
			t.Run(g.name, func(t *testing.T) {
				g.run(t, newConfig)
			})
		}
	})
}

func runCaseGroup(t *testing.T, newConfig NewConfig, g caseGroup) {
	t.Run(g.name, func(t *testing.T) {
		c, cfg := newTestClient(t, newConfig, defaultWhen, g.configure, testconfig.DefaultRTJson())
		gqlcase.QueryTestcases(t, c, g.cases(t, cfg))
	})
}

func runClockGroup(t *testing.T, newConfig NewConfig, g clockGroup) {
	t.Run(g.name, func(t *testing.T) {
		for _, tc := range g.cases() {
			t.Run(tc.Name, func(t *testing.T) {
				c, _ := newTestClient(t, newConfig, caseTime(t, tc.whenUtc), nil, testconfig.DefaultRTJson())
				gqlcase.QueryTestcase(t, c, tc.testcase)
			})
		}
	})
}

func runTestGroup(t *testing.T, newConfig NewConfig, g testGroup) {
	t.Run(g.name, func(t *testing.T) {
		c, cfg := newTestClient(t, newConfig, defaultWhen, nil, testconfig.DefaultRTJson())
		g.run(t, c, cfg)
	})
}

// raiseMaxLimit raises gql.RESOLVER_MAXLIMIT to the largest limit the cases
// request, until the test ends.
func raiseMaxLimit(t testing.TB) {
	limit := gql.RESOLVER_MAXLIMIT
	gql.RESOLVER_MAXLIMIT = 100_000
	t.Cleanup(func() { gql.RESOLVER_MAXLIMIT = limit })
}

// testRt runs a realtime case against a client whose realtime finder holds
// only the case's fixtures, at the case's time.
func testRt(t *testing.T, newConfig NewConfig, tc rtTestCase) {
	t.Run(tc.name, func(t *testing.T) {
		c, _ := newTestClient(t, newConfig, caseTime(t, tc.whenUtc), nil, tc.rtfiles)
		var resp map[string]interface{}
		opts := []client.Option{}
		for k, v := range tc.vars {
			opts = append(opts, client.Var(k, v))
		}
		if err := c.Post(tc.query, &resp, opts...); err != nil {
			t.Error(err)
			return
		}
		jj := gqlcase.ToJSON(resp)
		if tc.cb != nil {
			tc.cb(t, jj)
		}
	})
}

// caseTime returns when a case runs: whenUtc, or defaultWhen when the case
// sets none.
func caseTime(t testing.TB, whenUtc string) time.Time {
	if whenUtc == "" {
		return defaultWhen
	}
	when, err := time.Parse(time.RFC3339, whenUtc)
	if err != nil {
		t.Fatal(err)
	}
	return when
}

// newTestClient serves the config that newConfig builds at now through the
// GraphQL handler, as a signed-in test user, and returns the config. The
// config's realtime finder holds the fixtures rtfiles names.
func newTestClient(t testing.TB, newConfig NewConfig, now time.Time, configure func(*model.Config), rtfiles []testconfig.RTJsonFile) (*client.Client, model.Config) {
	cfg := newConfig(t, now)
	cfg.Clock = &clock.Mock{T: now}
	cfg.MaxRadius = maxRadius
	if configure != nil {
		configure(&cfg)
	}
	if err := testconfig.AddRTJsons(context.Background(), cfg.RTFinder, now, rtfiles); err != nil {
		t.Fatal(err)
	}
	srv := gql.NewDefaultHandler()
	graphqlServer := model.AddConfigAndPerms(cfg, srv)
	srvMiddleware := usercheck.NewUserDefaultMiddleware(func() authn.User {
		return authn.NewCtxUser("testuser", "", "").WithRoles("testrole")
	})
	return client.New(srvMiddleware(graphqlServer)), cfg
}

// useGeohashFilter turns on the geohash secondary filter for spatial queries.
func useGeohashFilter(cfg *model.Config) {
	cfg.UseGeohashFilter = true
}

// needAtLeast fails the test unless ents holds at least n of what, so a case
// can't index past a short finder result.
func needAtLeast[T any](t testing.TB, n int, what string, ents []T) {
	t.Helper()
	if len(ents) < n {
		t.Fatalf("found %d %s, need at least %d", len(ents), what, n)
	}
}

// decodeGeojson decodes a GeoJSON feature for a query variable.
func decodeGeojson(t testing.TB, d string) hw {
	t.Helper()
	var feature hw
	if err := json.Unmarshal([]byte(d), &feature); err != nil {
		t.Fatal(err)
	}
	return feature
}

// defaultWhen is the time a case runs at unless it sets its own.
var defaultWhen = time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)

// maxRadius is the search radius limit, in meters, that the cases' too-large
// searches exceed.
const maxRadius = 100_000

type hw = map[string]interface{}

type testcaseSelector = gqlcase.TestcaseSelector

type testcase = gqlcase.Testcase

type testcaseWithClock struct {
	testcase
	whenUtc string
}

// rtTestCase is a query run against its own realtime fixtures, at its own time,
// and a callback that checks the result.
type rtTestCase struct {
	name    string
	query   string
	vars    map[string]interface{}
	rtfiles []testconfig.RTJsonFile
	cb      func(t *testing.T, jj string)
	whenUtc string
}

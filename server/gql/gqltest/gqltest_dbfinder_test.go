package gqltest

import (
	"os"
	"testing"
	"time"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/testconfig"
	"github.com/interline-io/transitland-lib/server/finders/dbfinder"
	"github.com/interline-io/transitland-lib/server/gql/internal/gqlcase"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
)

func TestMain(m *testing.M) {
	if a, ok := testutil.CheckTestDB(); !ok {
		log.Print(a)
		return
	}
	os.Exit(m.Run())
}

func TestDBFinder(t *testing.T) {
	Run(t, dbfinderConfig(testconfig.Options{}), Spatial, Realtime, Census)
}

// TestDBFinder_StopLocator runs the census groups again with a finder that
// reads the stops of stop buffers through a stop locator. The locator is the
// finder itself, so the results must not change.
func TestDBFinder_StopLocator(t *testing.T) {
	raiseMaxLimit(t)
	newConfig := func(t testing.TB, now time.Time) model.Config {
		cfg := dbfinderConfig(testconfig.Options{})(t, now)
		f, ok := cfg.Finder.(*dbfinder.Finder)
		if !ok {
			t.Fatalf("finder is %T, want *dbfinder.Finder", cfg.Finder)
		}
		f.SetStopLocator(f)
		return cfg
	}
	for _, g := range optionalGroups[Census] {
		runCaseGroup(t, newConfig, g)
	}
}

// TestDBFinder_MaterializedDepartures runs the groups that query stop times on
// a service day again, planned from the materialized departures CTE. Their
// results must not change.
func TestDBFinder_MaterializedDepartures(t *testing.T) {
	raiseMaxLimit(t)
	newConfig := dbfinderConfig(testconfig.Options{UseMaterializedDepartures: true})
	for _, g := range []caseGroup{
		{name: "LocationResolver_StopTimes", cases: locationResolverStopTimesTestcases},
		{name: "StopResolver", cases: stopResolverTestcases},
		{name: "StopResolver_StopTimes", cases: stopResolverStopTimesTestcases},
		{name: "StopResolver_StopTimes_Dates", cases: stopResolverStopTimesDatesTestcases},
		{name: "StopResolver_StopTimes_DateWindows", cases: stopResolverStopTimesDateWindowsTestcases},
	} {
		runCaseGroup(t, newConfig, g)
	}
	for _, g := range []clockGroup{
		{name: "StopResolver_StopTimes_RelativeDates", cases: stopResolverStopTimesRelativeDatesTestcases},
		{name: "StopResolver_StopTimes_Next", cases: stopResolverStopTimesNextTestcases},
		{name: "StopResolver_StopTimes_Frequencies", cases: stopResolverStopTimesFrequenciesTestcases},
	} {
		runClockGroup(t, newConfig, g)
	}
	runTestGroup(t, newConfig, testGroup{name: "StopResolver_ServiceWindowIsPerFeedVersion", run: stopResolverServiceWindowIsPerFeedVersionTest})
}

func BenchmarkDBFinder_StopResolver(b *testing.B) {
	raiseMaxLimit(b)
	c, cfg := newTestClient(b, dbfinderConfig(testconfig.Options{}), defaultWhen, nil, testconfig.DefaultRTJson())
	gqlcase.BenchmarkTestcases(b, c, stopResolverTestcases(b, cfg))
}

// dbfinderConfig returns the test database's config with these options, with
// its finders' clocks at now.
func dbfinderConfig(opts testconfig.Options) NewConfig {
	return func(t testing.TB, now time.Time) model.Config {
		opts.WhenUtc = now.UTC().Format("2006-01-02T15:04:05Z")
		return testconfig.Config(t, opts)
	}
}

package rtfinder

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/clock"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestMakeAlert_ActivePeriod(t *testing.T) {
	a := &pb.Alert{
		ActivePeriod: []*pb.TimeRange{
			{Start: proto.Uint64(100), End: proto.Uint64(200)},
			{Start: proto.Uint64(300)},
			{End: proto.Uint64(400)},
		},
	}
	got := makeAlert(alertEntity{Alert: a}, "").ActivePeriod
	if !assert.Len(t, got, 3) {
		return
	}
	check := func(idx int, start, end *int) {
		assert.Equal(t, start, got[idx].Start, "active_period[%d].start", idx)
		assert.Equal(t, end, got[idx].End, "active_period[%d].end", idx)
	}
	check(0, intp(100), intp(200))
	check(1, intp(300), nil)
	check(2, nil, intp(400))
}

func TestMakeAlert_Entity(t *testing.T) {
	got := makeAlert(alertEntity{ID: "ent-1", Alert: &pb.Alert{}}, "BA~rt")
	assert.Equal(t, "ent-1", got.EntityID)
	assert.Equal(t, "BA~rt", got.RtFeedOnestopID)
}

func intp(v int) *int {
	return &v
}

func testTrip(tripId, routeId string) *pb.TripDescriptor {
	td := &pb.TripDescriptor{}
	if tripId != "" {
		td.TripId = proto.String(tripId)
	}
	if routeId != "" {
		td.RouteId = proto.String(routeId)
	}
	return td
}

func TestOnRoute(t *testing.T) {
	routeAt := func(rid string) *pb.EntitySelector { return &pb.EntitySelector{RouteId: proto.String(rid)} }
	tcs := []struct {
		name   string
		s      *pb.EntitySelector
		modes  bool
		trips  bool
		expect bool
		lookup bool
	}{
		{"route", routeAt("05"), false, false, true, false},
		{"other route", routeAt("03"), true, true, false, false},
		{"route at stop", &pb.EntitySelector{RouteId: proto.String("05"), StopId: proto.String("FTVL")}, false, false, true, false},
		{"route of its mode", &pb.EntitySelector{RouteId: proto.String("05"), RouteType: proto.Int32(1)}, false, false, true, false},
		{"route of another mode", &pb.EntitySelector{RouteId: proto.String("05"), RouteType: proto.Int32(3)}, false, false, false, false},
		{"route with empty trip", &pb.EntitySelector{RouteId: proto.String("05"), Trip: &pb.TripDescriptor{}}, false, false, true, false},
		// A trip descriptor naming only a route picks out no trip: it is the route.
		{"route by trip descriptor", &pb.EntitySelector{Trip: testTrip("", "05")}, false, false, true, false},
		{"route by trip descriptor and directly", &pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("", "05")}, false, false, true, false},
		{"routes disagreeing", &pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("", "03")}, true, true, false, false},
		{"stop", &pb.EntitySelector{StopId: proto.String("FTVL")}, true, true, false, false},
		{"mode", &pb.EntitySelector{RouteType: proto.Int32(1)}, false, false, false, false},
		{"mode, include_modes", &pb.EntitySelector{RouteType: proto.Int32(1)}, true, false, true, false},
		{"extended type of mode, include_modes", &pb.EntitySelector{RouteType: proto.Int32(401)}, true, false, true, false},
		{"mode of agency, include_modes", &pb.EntitySelector{AgencyId: proto.String("BART"), RouteType: proto.Int32(1)}, true, false, true, false},
		{"other mode, include_modes", &pb.EntitySelector{RouteType: proto.Int32(3)}, true, false, false, false},
		{"mode at stop, include_modes", &pb.EntitySelector{RouteType: proto.Int32(1), StopId: proto.String("FTVL")}, true, false, false, false},
		{"mode on trip, include_modes", &pb.EntitySelector{RouteType: proto.Int32(1), Trip: testTrip("T05", "")}, true, false, false, false},
		{"trip of route", &pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("T05", "")}, false, false, false, false},
		{"trip of route, include_trips", &pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("T05", "")}, false, true, true, false},
		{"trip by descriptor route, include_trips", &pb.EntitySelector{Trip: testTrip("T05", "05")}, false, true, true, false},
		{"trip by descriptor of other route, include_trips", &pb.EntitySelector{Trip: testTrip("T05", "03")}, false, true, false, false},
		{"trip by trip_id", &pb.EntitySelector{Trip: testTrip("T05", "")}, false, false, false, false},
		{"trip by trip_id, include_trips", &pb.EntitySelector{Trip: testTrip("T05", "")}, false, true, true, true},
		{"trip of other route by trip_id, include_trips", &pb.EntitySelector{Trip: testTrip("T03", "")}, false, true, false, true},
		{"trip by start date, include_trips", &pb.EntitySelector{Trip: &pb.TripDescriptor{RouteId: proto.String("05"), StartDate: proto.String("20180530")}}, false, true, true, false},
		{"trip at stop, include_trips", &pb.EntitySelector{StopId: proto.String("FTVL"), Trip: testTrip("T05", "")}, false, true, true, true},
		{"trip of another mode, include_trips", &pb.EntitySelector{RouteType: proto.Int32(3), Trip: testTrip("T05", "")}, false, true, false, false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			looked := false
			tripRoutes := func() map[string]string {
				looked = true
				return map[string]string{"T05": "05", "T03": "03"}
			}
			assert.Equal(t, tc.expect, onRoute(tc.s, "05", 1, tc.modes, tc.trips, tripRoutes))
			assert.Equal(t, tc.lookup, looked, "trip lookup")
		})
	}
	// A trip that resolves to no route matches none.
	assert.False(t, onRoute(&pb.EntitySelector{Trip: testTrip("T99", "")}, "05", 1, false, true, func() map[string]string { return nil }))
}

func TestTripAgrees(t *testing.T) {
	tcs := []struct {
		name   string
		s      *pb.EntitySelector
		expect bool
		lookup bool
	}{
		{"trip only", &pb.EntitySelector{Trip: testTrip("T05", "")}, true, false},
		{"agency", &pb.EntitySelector{AgencyId: proto.String("BART"), Trip: testTrip("T05", "")}, true, true},
		{"other agency", &pb.EntitySelector{AgencyId: proto.String("AC"), Trip: testTrip("T05", "")}, false, true},
		{"route", &pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("T05", "")}, true, true},
		{"route by trip descriptor", &pb.EntitySelector{Trip: testTrip("T05", "05")}, true, true},
		{"other route", &pb.EntitySelector{Trip: testTrip("T05", "03")}, false, true},
		{"routes disagreeing", &pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("T05", "03")}, false, false},
		{"mode", &pb.EntitySelector{RouteType: proto.Int32(1), Trip: testTrip("T05", "")}, true, true},
		{"other mode", &pb.EntitySelector{RouteType: proto.Int32(3), Trip: testTrip("T05", "")}, false, true},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			looked := false
			route := func() (GtfsRoute, bool) {
				looked = true
				return GtfsRoute{RouteID: "05", RouteType: 1, AgencyID: "BART"}, true
			}
			assert.Equal(t, tc.expect, tripAgrees(tc.s, route))
			assert.Equal(t, tc.lookup, looked, "route lookup")
		})
	}
	// Without the trip's route, a selector giving an agency, route or mode can't agree.
	assert.False(t, tripAgrees(&pb.EntitySelector{AgencyId: proto.String("BART"), Trip: testTrip("T05", "")}, func() (GtfsRoute, bool) { return GtfsRoute{}, false }))
}

func TestStopAgrees(t *testing.T) {
	fv := feedVersionRefs{
		agencyIds:  func() ([]string, bool) { return []string{"BART"}, true },
		routeTypes: func() ([]int, bool) { return []int{1}, true },
		route: func(routeId string) (GtfsRoute, bool) {
			if routeId == "05" {
				return GtfsRoute{RouteID: "05", RouteType: 1, AgencyID: "BART"}, true
			}
			return GtfsRoute{}, false
		},
	}
	stop := func(s *pb.EntitySelector) *pb.EntitySelector {
		s.StopId = proto.String("FTVL")
		return s
	}
	tcs := []struct {
		name   string
		s      *pb.EntitySelector
		expect bool
	}{
		{"stop", stop(&pb.EntitySelector{}), true},
		{"agency", stop(&pb.EntitySelector{AgencyId: proto.String("BART")}), true},
		{"other agency", stop(&pb.EntitySelector{AgencyId: proto.String("AC")}), false},
		{"mode", stop(&pb.EntitySelector{RouteType: proto.Int32(1)}), true},
		{"mode not run", stop(&pb.EntitySelector{RouteType: proto.Int32(3)}), false},
		{"route", stop(&pb.EntitySelector{RouteId: proto.String("05")}), true},
		{"route by trip descriptor", stop(&pb.EntitySelector{Trip: testTrip("", "05")}), true},
		{"route not in feed version", stop(&pb.EntitySelector{RouteId: proto.String("51A")}), false},
		{"route of other agency", stop(&pb.EntitySelector{AgencyId: proto.String("AC"), RouteId: proto.String("05")}), false},
		{"route of other mode", stop(&pb.EntitySelector{RouteId: proto.String("05"), RouteType: proto.Int32(3)}), false},
		{"routes disagreeing", stop(&pb.EntitySelector{RouteId: proto.String("05"), Trip: testTrip("", "03")}), false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, stopAgrees(tc.s, fv))
		})
	}
}

func TestMatchesAgencyMode(t *testing.T) {
	tcs := []struct {
		name     string
		s        *pb.EntitySelector
		agencyId string
		expect   bool
	}{
		{"mode of agency", &pb.EntitySelector{AgencyId: proto.String("BART"), RouteType: proto.Int32(1)}, "BART", true},
		{"mode", &pb.EntitySelector{RouteType: proto.Int32(1)}, "BART", true},
		{"extended type of mode", &pb.EntitySelector{RouteType: proto.Int32(401)}, "BART", true},
		{"basic type of extended type", &pb.EntitySelector{RouteType: proto.Int32(2)}, "BART", true},
		{"mode with empty trip", &pb.EntitySelector{RouteType: proto.Int32(1), Trip: &pb.TripDescriptor{}}, "BART", true},
		{"mode not run", &pb.EntitySelector{RouteType: proto.Int32(3)}, "BART", false},
		{"mode of agency without agency_id", &pb.EntitySelector{AgencyId: proto.String("BART"), RouteType: proto.Int32(1)}, "", false},
		{"mode, agency without agency_id", &pb.EntitySelector{RouteType: proto.Int32(1)}, "", true},
		{"mode of other agency", &pb.EntitySelector{AgencyId: proto.String("AC"), RouteType: proto.Int32(1)}, "BART", false},
		{"mode on route", &pb.EntitySelector{RouteType: proto.Int32(1), RouteId: proto.String("05")}, "BART", false},
		{"mode at stop", &pb.EntitySelector{RouteType: proto.Int32(1), StopId: proto.String("FTVL")}, "BART", false},
		{"mode on trip", &pb.EntitySelector{RouteType: proto.Int32(1), Trip: testTrip("T05", "")}, "BART", false},
		{"agency", &pb.EntitySelector{AgencyId: proto.String("BART")}, "BART", false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, matchesAgencyMode(tc.s, tc.agencyId, []int{1, 109}))
		})
	}
	// Without route types, as when mode-wide alerts are not asked for, no mode matches.
	assert.False(t, matchesAgencyMode(&pb.EntitySelector{RouteType: proto.Int32(1)}, "BART", nil))
}

func TestNamesAgency(t *testing.T) {
	tcs := []struct {
		name     string
		s        *pb.EntitySelector
		agencyId string
		expect   bool
	}{
		{"agency", &pb.EntitySelector{AgencyId: proto.String("BART")}, "BART", true},
		{"agency with empty trip", &pb.EntitySelector{AgencyId: proto.String("BART"), Trip: &pb.TripDescriptor{}}, "BART", true},
		{"other agency", &pb.EntitySelector{AgencyId: proto.String("AC")}, "BART", false},
		{"mode of agency", &pb.EntitySelector{AgencyId: proto.String("BART"), RouteType: proto.Int32(1)}, "BART", false},
		{"route of agency", &pb.EntitySelector{AgencyId: proto.String("BART"), RouteId: proto.String("05")}, "BART", false},
		{"agency at stop", &pb.EntitySelector{AgencyId: proto.String("BART"), StopId: proto.String("FTVL")}, "BART", false},
		{"trip of agency", &pb.EntitySelector{AgencyId: proto.String("BART"), Trip: testTrip("T05", "")}, "BART", false},
		{"agency without agency_id", &pb.EntitySelector{AgencyId: proto.String("BART")}, "", false},
		{"empty selector, agency without agency_id", &pb.EntitySelector{}, "", false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, namesAgency(tc.s, tc.agencyId))
		})
	}
}

func TestAgencyIdMatches(t *testing.T) {
	tcs := []struct {
		name     string
		s        *pb.EntitySelector
		agencyId string
		expect   bool
	}{
		{"agency", &pb.EntitySelector{AgencyId: proto.String("BART")}, "BART", true},
		{"other agency", &pb.EntitySelector{AgencyId: proto.String("AC")}, "BART", false},
		{"no agency", &pb.EntitySelector{RouteId: proto.String("05")}, "BART", true},
		{"agency without agency_id", &pb.EntitySelector{AgencyId: proto.String("BART")}, "", false},
		{"no agency, agency without agency_id", &pb.EntitySelector{RouteId: proto.String("05")}, "", true},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, agencyIdMatches(tc.s, tc.agencyId))
		})
	}
}

// Route.alerts across every route of a feed version resolves the trips named
// without a route in one query, however many routes ask.
func TestFindAlertsForRoute_TripLookup(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	raw := testutil.MustOpenTestDB(t)
	routes := testRoutes(t, raw, testBartFeedVersion(t, raw))
	msg := testReadRT(t, "BA-alerts-trips.json")
	ctx := context.Background()
	headers := func(alerts []*model.Alert) []string {
		var ret []string
		for _, a := range alerts {
			ret = append(ret, a.HeaderText[0].Text)
		}
		return ret
	}
	t.Run("concurrent routes", func(t *testing.T) {
		db := &countingDB{Ext: raw}
		f := testFinder(t, NewPostgresLookup(db), "realtime_alerts", msg)
		found := map[string][]string{}
		var lock sync.Mutex
		run := func() {
			var wg sync.WaitGroup
			for rid, r := range routes {
				for range 10 {
					wg.Go(func() {
						h := headers(f.FindAlertsForRoute(ctx, r, nil, nil, false, true))
						lock.Lock()
						found[rid] = append(found[rid], h...)
						lock.Unlock()
					})
				}
			}
			wg.Wait()
		}
		run()
		assert.EqualValues(t, 1, db.tripQueries.Load(), "cold")
		run()
		assert.EqualValues(t, 1, db.tripQueries.Load(), "warm")
		assert.Contains(t, found["05"], "Trip 1031527WKDY")
		assert.Contains(t, found["03"], "Trip 2211533WKDY")
		assert.NotContains(t, found["05"], "Trip 2211533WKDY")
	})
	t.Run("without include_trips", func(t *testing.T) {
		// Trip alerts are left out, and so is the lookup that places them. A trip
		// descriptor naming only the route is about the route.
		db := &countingDB{Ext: raw}
		f := testFinder(t, NewPostgresLookup(db), "realtime_alerts", msg)
		assert.Equal(t, []string{"Route 05 by trip descriptor"}, headers(f.FindAlertsForRoute(ctx, routes["05"], nil, nil, false, false)))
		assert.EqualValues(t, 0, db.tripQueries.Load())
	})
	t.Run("failing lookup", func(t *testing.T) {
		// Route 05's call reaches three trips named by trip_id alone, which
		// share one lookup per call; a canceled call starts none.
		db := &countingDB{Ext: raw}
		db.fail.Store(true)
		f := testFinder(t, NewPostgresLookup(db), "realtime_alerts", msg)
		assert.NotContains(t, headers(f.FindAlertsForRoute(ctx, routes["05"], nil, nil, false, true)), "Trip 1031527WKDY")
		assert.EqualValues(t, 1, db.tripQueries.Load(), "failing")
		canceled, cancel := context.WithCancel(ctx)
		cancel()
		f.FindAlertsForRoute(canceled, routes["05"], nil, nil, false, true)
		assert.EqualValues(t, 1, db.tripQueries.Load(), "canceled")
		db.fail.Store(false)
		assert.Contains(t, headers(f.FindAlertsForRoute(ctx, routes["05"], nil, nil, false, true)), "Trip 1031527WKDY")
		assert.EqualValues(t, 2, db.tripQueries.Load(), "recovered")
	})
}

// Stop.alerts looks a feed version's agencies up at most once per call, even
// while the lookup fails.
func TestFindAlertsForStop_Lookups(t *testing.T) {
	if a, ok := testutil.CheckTestDB(); !ok {
		t.Skip(a)
	}
	raw := testutil.MustOpenTestDB(t)
	stop := testStop(t, raw, testBartFeedVersion(t, raw), "FTVL")
	msg := testReadRT(t, "BA-alerts-informed-entity.json")
	ctx := context.Background()
	headers := func(alerts []*model.Alert) []string {
		var ret []string
		for _, a := range alerts {
			ret = append(ret, a.HeaderText[0].Text)
		}
		return ret
	}
	db := &countingDB{Ext: raw}
	db.fail.Store(true)
	f := testFinder(t, NewPostgresLookup(db), "realtime_alerts", msg)
	found := headers(f.FindAlertsForStop(ctx, stop, nil, nil))
	assert.Contains(t, found, "Route 05 at Fruitvale")
	assert.NotContains(t, found, "BART at Fruitvale")
	assert.EqualValues(t, 1, db.agencyQueries.Load(), "failing")
	db.fail.Store(false)
	assert.Contains(t, headers(f.FindAlertsForStop(ctx, stop, nil, nil)), "BART at Fruitvale")
	assert.EqualValues(t, 2, db.agencyQueries.Load(), "recovered")
}

// unexpectedLookup fails the test on any lookup. A test's Lookup embeds it and
// answers the lookups the test expects.
type unexpectedLookup struct {
	t *testing.T
}

func (l unexpectedLookup) unexpected(name string) {
	l.t.Errorf("unexpected lookup: %s", name)
}

func (l unexpectedLookup) GetFeedVersionRTFeeds(context.Context, int) ([]string, bool) {
	l.unexpected("GetFeedVersionRTFeeds")
	return nil, false
}

func (l unexpectedLookup) FeedVersionTimezone(context.Context, int) (*time.Location, bool) {
	l.unexpected("FeedVersionTimezone")
	return nil, false
}

func (l unexpectedLookup) GetTripSpan(context.Context, int, int) (TripSpan, bool) {
	l.unexpected("GetTripSpan")
	return TripSpan{}, false
}

func (l unexpectedLookup) GetGtfsStopID(context.Context, int) (string, bool) {
	l.unexpected("GetGtfsStopID")
	return "", false
}

func (l unexpectedLookup) StopTimezone(context.Context, int, string) (*time.Location, bool) {
	l.unexpected("StopTimezone")
	return nil, false
}

func (l unexpectedLookup) GetGtfsAgencyID(context.Context, int) (string, bool) {
	l.unexpected("GetGtfsAgencyID")
	return "", false
}

func (l unexpectedLookup) GetRoute(context.Context, int) (GtfsRoute, bool) {
	l.unexpected("GetRoute")
	return GtfsRoute{}, false
}

func (l unexpectedLookup) GetFeedVersionRoute(context.Context, int, string) (GtfsRoute, bool) {
	l.unexpected("GetFeedVersionRoute")
	return GtfsRoute{}, false
}

func (l unexpectedLookup) GetFeedVersionAgencyIDs(context.Context, int) ([]string, bool) {
	l.unexpected("GetFeedVersionAgencyIDs")
	return nil, false
}

func (l unexpectedLookup) GetFeedVersionRouteTypes(context.Context, int) ([]int, bool) {
	l.unexpected("GetFeedVersionRouteTypes")
	return nil, false
}

func (l unexpectedLookup) GetTripRouteIDs(context.Context, int, []string) (map[string]string, bool) {
	l.unexpected("GetTripRouteIDs")
	return nil, false
}

func (l unexpectedLookup) GetAgencyRouteIDs(context.Context, int) (map[string]bool, bool) {
	l.unexpected("GetAgencyRouteIDs")
	return nil, false
}

func (l unexpectedLookup) GetFeedOperatorCount(context.Context, string) (int, bool) {
	l.unexpected("GetFeedOperatorCount")
	return 0, false
}

// tripRouteLookup resolves trips to routes for feed version 1, counting its
// calls and failing while fail is set.
type tripRouteLookup struct {
	unexpectedLookup
	routes map[string]string
	calls  atomic.Int64
	fail   atomic.Bool
}

func (l *tripRouteLookup) GetTripRouteIDs(_ context.Context, fvid int, tripIds []string) (map[string]string, bool) {
	l.calls.Add(1)
	if l.fail.Load() {
		return nil, false
	}
	ret := map[string]string{}
	for _, tid := range tripIds {
		if rid, ok := l.routes[tid]; ok && fvid == 1 {
			ret[tid] = rid
		}
	}
	return ret, true
}

func TestFinder_tripRouteIDs(t *testing.T) {
	lookup := &tripRouteLookup{unexpectedLookup: unexpectedLookup{t}, routes: map[string]string{"1031527WKDY": "05"}}
	f := &Finder{lookup: lookup}
	ctx := context.Background()
	newSource := func(tripIds ...string) *Source {
		msg := &pb.FeedMessage{}
		for _, tid := range tripIds {
			msg.Entity = append(msg.Entity, &pb.FeedEntity{Alert: &pb.Alert{InformedEntity: []*pb.EntitySelector{{Trip: testTrip(tid, "")}}}})
		}
		src, err := NewSource("BA")
		if err != nil {
			t.Fatal(err)
		}
		if err := src.processMessage(ctx, msg); err != nil {
			t.Fatal(err)
		}
		return src
	}
	expect := map[string]string{"1031527WKDY": "05"}

	// Concurrent callers share one lookup.
	src := newSource("1031527WKDY", "no-such-trip")
	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			assert.Equal(t, expect, f.tripRouteIDs(ctx, src, 1))
		})
	}
	wg.Wait()
	assert.EqualValues(t, 1, lookup.calls.Load(), "concurrent callers")

	// Another feed version resolves the same message on its own.
	assert.Empty(t, f.tripRouteIDs(ctx, src, 2))
	assert.EqualValues(t, 2, lookup.calls.Load(), "other feed version")

	// A message naming every trip's route needs no lookup.
	assert.Nil(t, f.tripRouteIDs(ctx, newSource(), 1))
	assert.EqualValues(t, 2, lookup.calls.Load(), "nothing unrouted")

	// A caller that has gone away is served what is held, but starts no lookup.
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	assert.Equal(t, expect, f.tripRouteIDs(canceled, src, 1))
	assert.Nil(t, f.tripRouteIDs(canceled, newSource("1031527WKDY"), 1))
	assert.EqualValues(t, 2, lookup.calls.Load(), "canceled")

	// A failed lookup is not cached.
	src = newSource("1031527WKDY")
	lookup.fail.Store(true)
	assert.Nil(t, f.tripRouteIDs(ctx, src, 1))
	lookup.fail.Store(false)
	assert.Equal(t, expect, f.tripRouteIDs(ctx, src, 1))
	assert.EqualValues(t, 4, lookup.calls.Load(), "retry after failure")
}

// feedVersionLookup answers the feed version lookups that trip update matching
// makes, for feed version 1: all that matching runs and stops given by value
// needs.
type feedVersionLookup struct {
	unexpectedLookup
	tz *time.Location
}

func (l feedVersionLookup) GetFeedVersionRTFeeds(_ context.Context, fvid int) ([]string, bool) {
	if fvid != 1 {
		return nil, false
	}
	return []string{"BA"}, true
}

func (l feedVersionLookup) FeedVersionTimezone(_ context.Context, fvid int) (*time.Location, bool) {
	return l.tz, fvid == 1 && l.tz != nil
}

// tripUpdateLookup adds the trip and stop lookups that matching a trip's runs
// and stop times makes.
type tripUpdateLookup struct {
	feedVersionLookup
	spans         map[int]TripSpan // by trip id
	stopIds       map[int]string   // GTFS stop_ids by stop id
	stopIdLookups *atomic.Int64    // counts GetGtfsStopID calls, when set
}

func (l tripUpdateLookup) GetTripSpan(_ context.Context, _ int, tripId int) (TripSpan, bool) {
	return l.spans[tripId], true
}

func (l tripUpdateLookup) GetGtfsStopID(_ context.Context, id int) (string, bool) {
	if l.stopIdLookups != nil {
		l.stopIdLookups.Add(1)
	}
	sid, ok := l.stopIds[id]
	return sid, ok
}

// The stop ids tripUpdateLookup gives Fruitvale and Lake Merritt.
const (
	testFtvl = 100
	testLake = 101
)

// testRunSpan is when trip 5172328WKDY runs on its service day: 23:40 to 24:31.
var testRunSpan = TripSpan{FirstDeparture: 23*3600 + 40*60, LastArrival: 24*3600 + 31*60}

func testLosAngeles(t *testing.T) *time.Location {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

// testRunsFinder returns a Finder over lookup, holding the BA-runs trip
// updates, at ten past midnight on 2018-05-31: while trip 5172328WKDY's run
// from the day before is still going.
func testRunsFinder(t *testing.T, lookup Lookup, loc *time.Location) *Finder {
	now := time.Date(2018, 5, 31, 0, 10, 0, 0, loc)
	return testFinderAt(t, NewFinder(kvcache.NewMemoryStore(), lookup), now, "realtime_trip_updates", testReadRT(t, "BA-runs-trip-updates.json"))
}

// testLookupFinder returns a testRunsFinder over a tripUpdateLookup, where
// trip 2 is 5172328WKDY.
func testLookupFinder(t *testing.T) *Finder {
	loc := testLosAngeles(t)
	lookup := tripUpdateLookup{
		feedVersionLookup: feedVersionLookup{unexpectedLookup: unexpectedLookup{t}, tz: loc},
		spans:             map[int]TripSpan{2: testRunSpan},
		stopIds:           map[int]string{testFtvl: "FTVL", testLake: "LAKE"},
	}
	return testRunsFinder(t, lookup, loc)
}

// testDate returns a day in May 2018.
func testDate(day int) tt.Date {
	return tt.NewDate(time.Date(2018, 5, day, 0, 0, 0, 0, time.UTC))
}

// testRun returns a trip of feed version 1 as its run on a day in May 2018.
func testRun(id int, tripId string, day int) *model.Trip {
	ret := &model.Trip{}
	ret.ID = id
	ret.FeedVersionID = 1
	ret.TripID.Set(tripId)
	ret.RunDates = []tt.Date{testDate(day)}
	return ret
}

// testStopTime returns a stop time at a stop, by stop id and stop_sequence.
func testStopTime(stopId int, seq int) *model.StopTime {
	st := &model.StopTime{}
	st.StopID.SetInt(stopId)
	st.StopSequence = tt.NewInt(seq)
	return st
}

// A Finder matches trip updates to runs and stops without a database, asking
// its Lookup about the static feed.
func TestFinder_TripUpdatesFromLookup(t *testing.T) {
	f := testLookupFinder(t)
	tcs := []struct {
		name  string
		trip  *model.Trip
		seq   int
		delay int32 // of the update matched at FTVL; zero when none matches
	}{
		{"run by start_date", testRun(1, "1031527WKDY", 31), 12, 300},
		{"stop by stop_id", testRun(1, "1031527WKDY", 30), 99, 60},
		{"current run, from the trip's span", testRun(2, "5172328WKDY", 30), 12, 180},
		{"current run not requested", testRun(2, "5172328WKDY", 31), 12, 0},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ste, ok := f.FindStopTimeUpdate(context.Background(), tc.trip, testStopTime(testFtvl, tc.seq))
			if tc.delay == 0 {
				assert.False(t, ok)
				return
			}
			if assert.True(t, ok) && assert.NotNil(t, ste.StopTimeUpdate) {
				assert.Equal(t, "FTVL", ste.StopTimeUpdate.GetStopId())
				assert.Equal(t, tc.delay, ste.StopTimeUpdate.GetDeparture().GetDelay())
			}
		})
	}
}

// FindRunUpdate matches trip updates to a run given by value, asking its Lookup
// only about the feed version.
func TestFinder_FindRunUpdate(t *testing.T) {
	loc := testLosAngeles(t)
	f := testRunsFinder(t, feedVersionLookup{unexpectedLookup: unexpectedLookup{t}, tz: loc}, loc)
	ctx := context.Background()
	tcs := []struct {
		name  string
		run   Run
		found bool  // the run has an update
		delay int32 // of the update matched at FTVL
	}{
		{"run by start_date", Run{FeedVersionID: 1, TripID: "1031527WKDY", Date: testDate(31)}, true, 300},
		{"current run, from the run's span", Run{FeedVersionID: 1, TripID: "5172328WKDY", Date: testDate(30), Span: testRunSpan}, true, 180},
		{"current run not requested", Run{FeedVersionID: 1, TripID: "5172328WKDY", Date: testDate(31), Span: testRunSpan}, false, 0},
		{"feed version without realtime feeds", Run{FeedVersionID: 2, TripID: "1031527WKDY", Date: testDate(31)}, false, 0},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ru := f.FindRunUpdate(ctx, tc.run)
			if !tc.found {
				assert.Nil(t, ru)
				return
			}
			if !assert.NotNil(t, ru) {
				return
			}
			assert.Equal(t, tc.run.TripID, ru.TripUpdate().GetTrip().GetTripId())
			got := ru.MatchStops(ctx, []StopRef{{StopSequence: 12, StopID: "FTVL"}})
			if assert.Len(t, got, 1) && assert.NotNil(t, got[0]) && assert.NotNil(t, got[0].StopTimeUpdate) {
				assert.Equal(t, tc.delay, got[0].StopTimeUpdate.GetDeparture().GetDelay())
			}
		})
	}
}

// FindStopTimeUpdates, and a RunUpdate's MatchStops, match each of a run's stop
// times to its stop time update, by stop_sequence or by a stop_id the update
// names once.
func TestFinder_MatchStops(t *testing.T) {
	departs := func(delay int32) *pb.TripUpdate_StopTimeEvent {
		return &pb.TripUpdate_StopTimeEvent{Delay: proto.Int32(delay)}
	}
	// The first stop named by stop_sequence and the rest by stop_id alone, on a
	// loop that visits LOOP twice.
	stus := []*pb.TripUpdate_StopTimeUpdate{
		{StopSequence: proto.Uint32(1), StopId: proto.String("A"), Departure: departs(60)},
		{StopId: proto.String("B"), Departure: departs(120)},
		{StopId: proto.String("LOOP"), Departure: departs(180)},
		{StopId: proto.String("LOOP"), Departure: departs(240)},
	}
	msg := &pb.FeedMessage{
		Header: &pb.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
		Entity: []*pb.FeedEntity{{
			Id: proto.String("loop"),
			TripUpdate: &pb.TripUpdate{
				Trip:           &pb.TripDescriptor{TripId: proto.String("loop"), StartDate: proto.String("20180531")},
				StopTimeUpdate: stus,
			},
		}},
	}
	ctx := context.Background()
	// Matching the trip's stop times asks the lookup for GTFS stop_ids by stop
	// id, and stop 5 is one it can't find. Matching by value is given the
	// stop_ids, with none for the last stop.
	stopIdLookups := &atomic.Int64{}
	lookupFinder := testFinder(t, tripUpdateLookup{
		feedVersionLookup: feedVersionLookup{unexpectedLookup: unexpectedLookup{t}},
		stopIds:           map[int]string{1: "A", 2: "B", 3: "LOOP", 4: "E"},
		stopIdLookups:     stopIdLookups,
	}, "realtime_trip_updates", msg)
	sts := []*model.StopTime{testStopTime(1, 1), testStopTime(2, 2), testStopTime(3, 3), testStopTime(3, 4), testStopTime(4, 5), testStopTime(5, 6)}
	valueFinder := testFinder(t, feedVersionLookup{unexpectedLookup: unexpectedLookup{t}}, "realtime_trip_updates", msg)
	stops := []StopRef{{1, "A"}, {2, "B"}, {3, "LOOP"}, {4, "LOOP"}, {5, "E"}, {6, ""}}
	matchByValue := func(day int) []*model.RTStopTimeUpdate {
		ru := valueFinder.FindRunUpdate(ctx, Run{FeedVersionID: 1, TripID: "loop", Date: testDate(day)})
		if ru == nil {
			return make([]*model.RTStopTimeUpdate, len(stops))
		}
		return ru.MatchStops(ctx, stops)
	}
	tcs := []struct {
		name      string
		found     bool  // matched to the trip update
		stu       int   // index of the matched stop time update, or -1 for none
		lastDelay int32 // zero for none
	}{
		{"by stop_sequence", true, 0, 0},
		{"by stop_id", true, 1, 120},
		{"first visit to a stop named twice", false, -1, 0},
		{"second visit to a stop named twice", false, -1, 0},
		{"stop not in the update", true, -1, 240},
		{"stop without a known stop_id", false, -1, 0},
	}
	for _, path := range []struct {
		name  string
		match func(day int) []*model.RTStopTimeUpdate
	}{
		{"FindStopTimeUpdates", func(day int) []*model.RTStopTimeUpdate {
			return lookupFinder.FindStopTimeUpdates(ctx, testRun(1, "loop", day), sts)
		}},
		{"MatchStops", matchByValue},
	} {
		t.Run(path.name, func(t *testing.T) {
			got := path.match(31)
			if !assert.Len(t, got, len(tcs)) {
				return
			}
			for i, tc := range tcs {
				t.Run(tc.name, func(t *testing.T) {
					ste := got[i]
					if !tc.found {
						assert.Nil(t, ste)
						return
					}
					if !assert.NotNil(t, ste) {
						return
					}
					assert.Equal(t, "loop", ste.TripUpdate.GetTrip().GetTripId())
					if tc.stu < 0 {
						assert.Nil(t, ste.StopTimeUpdate)
					} else {
						assert.True(t, proto.Equal(stus[tc.stu], ste.StopTimeUpdate), "stop time update %d", tc.stu)
					}
					if tc.lastDelay == 0 {
						assert.Nil(t, ste.LastDelay)
					} else if assert.NotNil(t, ste.LastDelay) {
						assert.Equal(t, tc.lastDelay, *ste.LastDelay)
					}
				})
			}
			// A run without an update matches none of its stop times.
			for i, ste := range path.match(30) {
				assert.Nil(t, ste, "stop time %d", i)
			}
		})
	}
	// Matching the trip's stop times looked up the stop_id of the five that no
	// update names by stop_sequence, and none for the run without an update.
	assert.EqualValues(t, 5, stopIdLookups.Load())
}

// payloadAgeLookup answers the lookups that TestFinder_PayloadAge's matching
// paths make: feed version 1's realtime feeds, its route R, and agency 5's
// routes.
type payloadAgeLookup struct {
	feedVersionLookup
}

func (l payloadAgeLookup) GetFeedVersionRoute(_ context.Context, fvid int, routeId string) (GtfsRoute, bool) {
	return GtfsRoute{ID: 10, RouteID: "R", AgencyID: "AG"}, fvid == 1 && routeId == "R"
}

func (l payloadAgeLookup) GetAgencyRouteIDs(_ context.Context, id int) (map[string]bool, bool) {
	return map[string]bool{"R": true}, id == 5
}

// Matching trip updates and vehicle positions ignores a payload whose header
// timestamp is more than an hour from now, behind or ahead. Alerts and
// GetMessage still read the stale payload, and GetMessage reports the age.
func TestFinder_PayloadAge(t *testing.T) {
	ctx := context.Background()
	fetched := time.Date(2018, 5, 31, 12, 0, 0, 0, time.UTC)
	td := &pb.TripDescriptor{TripId: proto.String("T"), RouteId: proto.String("R"), StartDate: proto.String("20180531")}
	added := &pb.TripDescriptor{TripId: proto.String("A"), RouteId: proto.String("R"), ScheduleRelationship: pb.TripDescriptor_ADDED.Enum()}
	payloads := map[string]*pb.FeedMessage{
		"realtime_trip_updates": {Entity: []*pb.FeedEntity{
			{Id: proto.String("tu"), TripUpdate: &pb.TripUpdate{Trip: td, StopTimeUpdate: []*pb.TripUpdate_StopTimeUpdate{{StopSequence: proto.Uint32(1)}}}},
			{Id: proto.String("added"), TripUpdate: &pb.TripUpdate{Trip: added, StopTimeUpdate: []*pb.TripUpdate_StopTimeUpdate{{StopId: proto.String("S1")}}}},
		}},
		vehiclePositionTopicKey: {Entity: []*pb.FeedEntity{{Id: proto.String("vp"), Vehicle: &pb.VehiclePosition{Trip: td}}}},
		"realtime_alerts": {Entity: []*pb.FeedEntity{{Id: proto.String("alert"), Alert: &pb.Alert{InformedEntity: []*pb.EntitySelector{{
			Trip: &pb.TripDescriptor{TripId: proto.String("T"), StartDate: proto.String("20180531")},
		}}}}}},
	}
	// A Finder an age after the payloads were fetched, their headers stamped
	// with timestamp, or with none.
	finderAfter := func(t *testing.T, timestamp *uint64, age time.Duration) *Finder {
		f := NewFinder(kvcache.NewMemoryStore(), payloadAgeLookup{feedVersionLookup{unexpectedLookup: unexpectedLookup{t}}})
		t.Cleanup(func() { f.Close() })
		for kind, msg := range payloads {
			msg = proto.Clone(msg).(*pb.FeedMessage)
			msg.Header = &pb.FeedHeader{GtfsRealtimeVersion: proto.String("2.0"), Timestamp: timestamp}
			data, err := proto.Marshal(msg)
			if err != nil {
				t.Fatal(err)
			}
			if err := f.AddData(ctx, getTopicKey("BA", kind), data); err != nil {
				t.Fatal(err)
			}
		}
		f.Clock = &clock.Mock{T: fetched.Add(age)}
		return f
	}
	stamped := proto.Uint64(uint64(fetched.Unix()))
	tcs := []struct {
		name      string
		timestamp *uint64
		age       time.Duration
		current   bool
	}{
		{"under an hour old", stamped, 59 * time.Minute, true},
		{"exactly an hour old", stamped, time.Hour, true},
		{"over an hour old", stamped, 61 * time.Minute, false},
		{"over an hour ahead of the clock", stamped, -61 * time.Minute, false},
		{"without a header timestamp", nil, 61 * time.Minute, true},
	}
	trip := testRun(1, "T", 31)
	stop, agency, route := &model.Stop{}, &model.Agency{}, &model.Route{}
	stop.FeedVersionID = 1
	stop.StopID.Set("S1")
	agency.ID, agency.FeedVersionID = 5, 1
	route.FeedVersionID = 1
	route.RouteID.Set("R")
	count := func(current bool) int {
		if current {
			return 1
		}
		return 0
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			f := finderAfter(t, tc.timestamp, tc.age)
			// Trip updates.
			assert.Equal(t, tc.current, f.FindRunUpdate(ctx, Run{FeedVersionID: 1, TripID: "T", Date: testDate(31)}) != nil, "FindRunUpdate")
			assert.Len(t, f.GetAddedTripsForStop(ctx, stop), count(tc.current), "GetAddedTripsForStop")
			// Vehicle positions.
			assert.Equal(t, tc.current, f.FindVehiclePositionForTrip(ctx, trip, nil) != nil, "FindVehiclePositionForTrip")
			assert.Len(t, f.FindVehiclePositionsForAgency(ctx, agency, nil, nil), count(tc.current), "FindVehiclePositionsForAgency")
			assert.Len(t, f.FindVehiclePositionsForRoute(ctx, route, nil, nil), count(tc.current), "FindVehiclePositionsForRoute")
			// Read whatever the age.
			assert.Len(t, f.FindAlertsForTrip(ctx, trip, nil, nil), 1, "FindAlertsForTrip")
			msg, stale, ok := f.GetMessage(ctx, "BA", "realtime_trip_updates")
			if assert.True(t, ok, "GetMessage") {
				assert.NotNil(t, msg)
				assert.Equal(t, !tc.current, stale, "GetMessage reports the payload stale")
			}
		})
	}
}

// A stop time update matches by stop_sequence only when it names one.
func TestFinder_MatchStops_StopSequence(t *testing.T) {
	departs := func(delay int32) *pb.TripUpdate_StopTimeEvent {
		return &pb.TripUpdate_StopTimeEvent{Delay: proto.Int32(delay)}
	}
	ctx := context.Background()
	tcs := []struct {
		name  string
		stus  []*pb.TripUpdate_StopTimeUpdate
		stops []StopRef
		match []int // index of each stop's matched stop time update, or -1 for none
	}{
		{
			name: "an update without a stop_sequence doesn't match stop_sequence 0",
			stus: []*pb.TripUpdate_StopTimeUpdate{
				{StopId: proto.String("C"), Departure: departs(60)},
				{StopId: proto.String("D"), Departure: departs(90)},
			},
			stops: []StopRef{{StopSequence: 0, StopID: "A"}, {StopSequence: 1, StopID: "C"}},
			match: []int{-1, 0},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			msg := &pb.FeedMessage{
				Header: &pb.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
				Entity: []*pb.FeedEntity{{Id: proto.String("t"), TripUpdate: &pb.TripUpdate{
					Trip:           &pb.TripDescriptor{TripId: proto.String("T"), StartDate: proto.String("20180531")},
					StopTimeUpdate: tc.stus,
				}}},
			}
			f := testFinder(t, feedVersionLookup{unexpectedLookup: unexpectedLookup{t}}, "realtime_trip_updates", msg)
			ru := f.FindRunUpdate(ctx, Run{FeedVersionID: 1, TripID: "T", Date: testDate(31)})
			if !assert.NotNil(t, ru) {
				return
			}
			got := ru.MatchStops(ctx, tc.stops)
			for i, want := range tc.match {
				if !assert.NotNil(t, got[i], "stop %d", i) {
					continue
				}
				if want < 0 {
					assert.Nil(t, got[i].StopTimeUpdate, "stop %d", i)
				} else {
					assert.True(t, proto.Equal(tc.stus[want], got[i].StopTimeUpdate), "stop %d", i)
				}
			}
		})
	}
}

// A copy of a trip, sent as DUPLICATED or as ADDED naming the scheduled trip,
// leaves the scheduled trip's own trip update and vehicle in place, and a trip
// added in real time takes only ADDED data.
func TestFinder_CopiesOfATrip(t *testing.T) {
	ctx := context.Background()
	lookup := feedVersionLookup{unexpectedLookup: unexpectedLookup{t}}
	td := func(rel pb.TripDescriptor_ScheduleRelationship) *pb.TripDescriptor {
		return &pb.TripDescriptor{TripId: proto.String("T"), StartDate: proto.String("20180531"), ScheduleRelationship: rel.Enum()}
	}
	header := &pb.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")}
	for _, rel := range []pb.TripDescriptor_ScheduleRelationship{pb.TripDescriptor_DUPLICATED, pb.TripDescriptor_ADDED} {
		t.Run(rel.String(), func(t *testing.T) {
			// The copy comes last, and its vehicle reported last.
			updates := testFinder(t, lookup, "realtime_trip_updates", &pb.FeedMessage{Header: header, Entity: []*pb.FeedEntity{
				{Id: proto.String("own"), TripUpdate: &pb.TripUpdate{Trip: td(pb.TripDescriptor_SCHEDULED)}},
				{Id: proto.String("copy"), TripUpdate: &pb.TripUpdate{Trip: td(rel)}},
			}})
			vehicles := testFinder(t, lookup, vehiclePositionTopicKey, &pb.FeedMessage{Header: header, Entity: []*pb.FeedEntity{
				{Id: proto.String("own"), Vehicle: &pb.VehiclePosition{Trip: td(pb.TripDescriptor_SCHEDULED), Timestamp: proto.Uint64(100)}},
				{Id: proto.String("copy"), Vehicle: &pb.VehiclePosition{Trip: td(rel), Timestamp: proto.Uint64(200)}},
			}})
			if ru := updates.FindRunUpdate(ctx, Run{FeedVersionID: 1, TripID: "T", Date: testDate(31)}); assert.NotNil(t, ru) {
				assert.Equal(t, pb.TripDescriptor_SCHEDULED, ru.TripUpdate().GetTrip().GetScheduleRelationship(), "trip update")
			}
			if vp := vehicles.FindVehiclePositionForTrip(ctx, testRun(1, "T", 31), nil); assert.NotNil(t, vp) {
				assert.Equal(t, "own", vp.ID, "vehicle")
			}
		})
	}
	t.Run("trip added in real time", func(t *testing.T) {
		// An added trip sharing its trip_id with scheduled trip T: T's update
		// comes last.
		f := testFinder(t, lookup, "realtime_trip_updates", &pb.FeedMessage{Header: header, Entity: []*pb.FeedEntity{
			{Id: proto.String("added"), TripUpdate: &pb.TripUpdate{Trip: td(pb.TripDescriptor_ADDED)}},
			{Id: proto.String("own"), TripUpdate: &pb.TripUpdate{Trip: td(pb.TripDescriptor_SCHEDULED)}},
		}})
		added := testRun(0, "T", 31)
		added.AddedInRealtime = true
		if tu := f.FindTrip(ctx, added); assert.NotNil(t, tu) {
			assert.Equal(t, pb.TripDescriptor_ADDED, tu.GetTrip().GetScheduleRelationship())
		}
	})
}

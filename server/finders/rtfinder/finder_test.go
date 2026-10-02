package rtfinder

import (
	"context"
	"sync"
	"testing"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
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
			route := func() (gtfsRoute, bool) {
				looked = true
				return gtfsRoute{RouteID: "05", RouteType: 1, AgencyID: "BART"}, true
			}
			assert.Equal(t, tc.expect, tripAgrees(tc.s, route))
			assert.Equal(t, tc.lookup, looked, "route lookup")
		})
	}
	// Without the trip's route, a selector giving an agency, route or mode can't agree.
	assert.False(t, tripAgrees(&pb.EntitySelector{AgencyId: proto.String("BART"), Trip: testTrip("T05", "")}, func() (gtfsRoute, bool) { return gtfsRoute{}, false }))
}

func TestStopAgrees(t *testing.T) {
	fv := feedVersionRefs{
		agencyIds:  func() ([]string, bool) { return []string{"BART"}, true },
		routeTypes: func() ([]int, bool) { return []int{1}, true },
		route: func(routeId string) (gtfsRoute, bool) {
			if routeId == "05" {
				return gtfsRoute{RouteID: "05", RouteType: 1, AgencyID: "BART"}, true
			}
			return gtfsRoute{}, false
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
		f := testFinder(t, db, "realtime_alerts", msg)
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
		f := testFinder(t, db, "realtime_alerts", msg)
		assert.Equal(t, []string{"Route 05 by trip descriptor"}, headers(f.FindAlertsForRoute(ctx, routes["05"], nil, nil, false, false)))
		assert.EqualValues(t, 0, db.tripQueries.Load())
	})
	t.Run("failing lookup", func(t *testing.T) {
		// Route 05's call reaches three trips named by trip_id alone, which
		// share one lookup per call; a canceled call starts none.
		db := &countingDB{Ext: raw}
		db.fail.Store(true)
		f := testFinder(t, db, "realtime_alerts", msg)
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
	f := testFinder(t, db, "realtime_alerts", msg)
	found := headers(f.FindAlertsForStop(ctx, stop, nil, nil))
	assert.Contains(t, found, "Route 05 at Fruitvale")
	assert.NotContains(t, found, "BART at Fruitvale")
	assert.EqualValues(t, 1, db.agencyQueries.Load(), "failing")
	db.fail.Store(false)
	assert.Contains(t, headers(f.FindAlertsForStop(ctx, stop, nil, nil)), "BART at Fruitvale")
	assert.EqualValues(t, 2, db.agencyQueries.Load(), "recovered")
}

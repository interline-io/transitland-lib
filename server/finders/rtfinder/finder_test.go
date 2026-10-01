package rtfinder

import (
	"context"
	"sync"
	"testing"

	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/caches/kvcache"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/jmoiron/sqlx"
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
	got := makeAlert(a).ActivePeriod
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

func TestSelectorNamesRoute(t *testing.T) {
	tcs := []struct {
		name   string
		s      *pb.EntitySelector
		expect bool
		lookup bool
	}{
		{"route", &pb.EntitySelector{RouteId: proto.String("05")}, true, false},
		{"other route", &pb.EntitySelector{RouteId: proto.String("03")}, false, false},
		{"route at stop", &pb.EntitySelector{RouteId: proto.String("05"), StopId: proto.String("FTVL")}, true, false},
		{"other route, trip of this route", &pb.EntitySelector{RouteId: proto.String("03"), Trip: testTrip("T05", "")}, false, false},
		{"trip route", &pb.EntitySelector{Trip: testTrip("", "05")}, true, false},
		{"other trip route, trip of this route", &pb.EntitySelector{Trip: testTrip("T05", "03")}, false, false},
		{"trip of this route", &pb.EntitySelector{Trip: testTrip("T05", "")}, true, true},
		{"trip of other route", &pb.EntitySelector{Trip: testTrip("T03", "")}, false, true},
		{"trip of this route at stop", &pb.EntitySelector{StopId: proto.String("FTVL"), Trip: testTrip("T05", "")}, true, true},
		{"empty trip", &pb.EntitySelector{Trip: &pb.TripDescriptor{}}, false, false},
		{"stop", &pb.EntitySelector{StopId: proto.String("FTVL")}, false, false},
		{"route type", &pb.EntitySelector{RouteType: proto.Int32(1)}, false, false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			looked := false
			tripRoutes := func() map[string]string {
				looked = true
				return map[string]string{"T05": "05", "T03": "03"}
			}
			assert.Equal(t, tc.expect, selectorNamesRoute(tc.s, "05", tripRoutes))
			assert.Equal(t, tc.lookup, looked, "trip lookup")
		})
	}
	// A trip that resolves to no route matches none, not even a route without a route_id.
	assert.False(t, tripOnRoute(testTrip("T99", ""), "", func() map[string]string { return nil }))
}

func TestMatchesRouteType(t *testing.T) {
	tcs := []struct {
		name   string
		s      *pb.EntitySelector
		expect bool
	}{
		{"mode", &pb.EntitySelector{RouteType: proto.Int32(1)}, true},
		{"mode of agency", &pb.EntitySelector{AgencyId: proto.String("BART"), RouteType: proto.Int32(1)}, true},
		{"extended type of mode", &pb.EntitySelector{RouteType: proto.Int32(401)}, true},
		{"other mode", &pb.EntitySelector{RouteType: proto.Int32(3)}, false},
		{"mode on route", &pb.EntitySelector{RouteType: proto.Int32(1), RouteId: proto.String("05")}, false},
		{"mode at stop", &pb.EntitySelector{RouteType: proto.Int32(1), StopId: proto.String("FTVL")}, false},
		{"mode on trip", &pb.EntitySelector{RouteType: proto.Int32(1), Trip: testTrip("T05", "")}, false},
		{"mode on trip route", &pb.EntitySelector{RouteType: proto.Int32(1), Trip: testTrip("", "05")}, false},
		{"mode with empty trip", &pb.EntitySelector{RouteType: proto.Int32(1), Trip: &pb.TripDescriptor{}}, true},
		{"agency", &pb.EntitySelector{AgencyId: proto.String("BART")}, false},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expect, matchesRouteType(tc.s, 1))
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
	fvid := testBartFeedVersion(t, raw)
	var routes []*model.Route
	q := `select id, feed_version_id, route_id, agency_id, route_type from gtfs_routes where feed_version_id = $1`
	if err := sqlx.Select(raw, &routes, q, fvid); err != nil {
		t.Fatal(err)
	}
	db := &countingDB{Ext: raw}
	f := NewFinder(kvcache.NewMemoryStore(), db)
	defer f.Close()
	ctx := context.Background()
	msg, err := rt.ReadFile(testdata.Path("server", "rt", "BA-alerts-trips.json"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.AddData(ctx, getTopicKey("BA", "realtime_alerts"), data); err != nil {
		t.Fatal(err)
	}
	headers := map[string][]string{}
	var lock sync.Mutex
	run := func() {
		var wg sync.WaitGroup
		for _, r := range routes {
			for range 10 {
				wg.Go(func() {
					for _, a := range f.FindAlertsForRoute(ctx, r, nil, nil) {
						lock.Lock()
						headers[r.RouteID.Val] = append(headers[r.RouteID.Val], a.HeaderText[0].Text)
						lock.Unlock()
					}
				})
			}
		}
		wg.Wait()
	}
	run()
	assert.EqualValues(t, 1, db.tripQueries.Load(), "cold")
	run()
	assert.EqualValues(t, 1, db.tripQueries.Load(), "warm")
	assert.Contains(t, headers["05"], "Trip 1031527WKDY")
	assert.Contains(t, headers["03"], "Trip 2211533WKDY")
	assert.NotContains(t, headers["05"], "Trip 2211533WKDY")
}

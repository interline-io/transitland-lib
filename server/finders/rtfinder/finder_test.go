package rtfinder

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/set"
	"github.com/interline-io/transitland-lib/rt/pb"
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
			tripIds := func() set.Set[string] {
				looked = true
				return set.New("T05")
			}
			assert.Equal(t, tc.expect, selectorNamesRoute(tc.s, "05", tripIds))
			assert.Equal(t, tc.lookup, looked, "trip lookup")
		})
	}
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

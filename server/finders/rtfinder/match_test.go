package rtfinder

import (
	"context"
	"testing"

	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

// Test stop times: a run starting at testBase whose stops are ten minutes
// apart, dwelling a minute at each.
const testBase int64 = 1_000_000

func testStopRef(seq int, stopId string) StopRef {
	return StopRef{StopSequence: seq, StopID: stopId, Arrival: tt.NewSeconds(seq*600 - 60), Departure: tt.NewSeconds(seq * 600)}
}

func delayOf(d int32) *pb.TripUpdate_StopTimeEvent {
	return &pb.TripUpdate_StopTimeEvent{Delay: proto.Int32(d)}
}

func timeOf(t int64) *pb.TripUpdate_StopTimeEvent {
	return &pb.TripUpdate_StopTimeEvent{Time: proto.Int64(t)}
}

// stu returns a stop time update naming a stop_sequence, when seq is
// positive, and a stop_id, when given.
func stu(seq int, stopId string) *pb.TripUpdate_StopTimeUpdate {
	ret := &pb.TripUpdate_StopTimeUpdate{}
	if seq > 0 {
		ret.StopSequence = proto.Uint32(uint32(seq))
	}
	if stopId != "" {
		ret.StopId = proto.String(stopId)
	}
	return ret
}

func withDeparture(s *pb.TripUpdate_StopTimeUpdate, e *pb.TripUpdate_StopTimeEvent) *pb.TripUpdate_StopTimeUpdate {
	s.Departure = e
	return s
}

func withArrival(s *pb.TripUpdate_StopTimeUpdate, e *pb.TripUpdate_StopTimeEvent) *pb.TripUpdate_StopTimeUpdate {
	s.Arrival = e
	return s
}

func withRelationship(s *pb.TripUpdate_StopTimeUpdate, rel pb.TripUpdate_StopTimeUpdate_ScheduleRelationship) *pb.TripUpdate_StopTimeUpdate {
	s.ScheduleRelationship = rel.Enum()
	return s
}

// Stop time updates are placed on a run's stop times in order along the trip,
// by stop_sequence or by stop_id.
func TestMatchRun_Placement(t *testing.T) {
	// A loop visiting B twice, with a last stop whose stop_id isn't known.
	stops := []StopRef{testStopRef(1, "A"), testStopRef(2, "B"), testStopRef(3, "C"), testStopRef(4, "B"), testStopRef(5, "D"), testStopRef(6, "")}
	tcs := []struct {
		name   string
		stus   []*pb.TripUpdate_StopTimeUpdate
		placed []int // the index of the stop time each update is placed on, or -1
	}{
		{"by stop_sequence", []*pb.TripUpdate_StopTimeUpdate{stu(2, ""), stu(5, "")}, []int{1, 4}},
		{"by stop_sequence and stop_id", []*pb.TripUpdate_StopTimeUpdate{stu(2, "B"), stu(5, "D")}, []int{1, 4}},
		{"by stop_id, each visit in order", []*pb.TripUpdate_StopTimeUpdate{stu(0, "B"), stu(0, "B")}, []int{1, 3}},
		{"by stop_id after an earlier stop time", []*pb.TripUpdate_StopTimeUpdate{stu(3, ""), stu(0, "B")}, []int{2, 3}},
		{"by stop_id when the stop_sequence values are another numbering", []*pb.TripUpdate_StopTimeUpdate{stu(20, "B"), stu(30, "D")}, []int{1, 4}},
		{"by stop_id when the stop_sequence names another stop", []*pb.TripUpdate_StopTimeUpdate{stu(2, "D")}, []int{4}},
		{"one naming no stop time is left out", []*pb.TripUpdate_StopTimeUpdate{stu(9, ""), stu(0, "Z"), stu(3, "")}, []int{-1, -1, 2}},
		{"one before the previous match is left out", []*pb.TripUpdate_StopTimeUpdate{stu(5, ""), stu(2, "")}, []int{4, -1}},
		{"a stop_id doesn't name a stop time without one", []*pb.TripUpdate_StopTimeUpdate{stu(0, "E")}, []int{-1}},
		{"a stop_sequence names a stop time without a stop_id", []*pb.TripUpdate_StopTimeUpdate{stu(6, "E")}, []int{5}},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tu := &pb.TripUpdate{Trip: &pb.TripDescriptor{TripId: proto.String("T")}, StopTimeUpdate: tc.stus}
			got := matchRun(context.Background(), tu, testBase, stops)
			if !assert.Len(t, got, len(stops)) {
				return
			}
			// Each update at its stop time, and nothing elsewhere.
			want := make([]*pb.TripUpdate_StopTimeUpdate, len(stops))
			for i, at := range tc.placed {
				if at >= 0 {
					want[at] = tc.stus[i]
				}
			}
			for i, r := range got {
				assert.Equal(t, tu, r.TripUpdate, "stop time %d trip update", i)
				assert.Same(t, want[i], r.StopTimeUpdate, "stop time %d", i)
			}
		})
	}
	// An update without a stop_sequence doesn't name stop_sequence 0.
	t.Run("stop_sequence 0 by stop_id only", func(t *testing.T) {
		stops := []StopRef{testStopRef(0, "A"), testStopRef(1, "B")}
		tu := &pb.TripUpdate{StopTimeUpdate: []*pb.TripUpdate_StopTimeUpdate{stu(0, "B")}}
		got := matchRun(context.Background(), tu, testBase, stops)
		assert.Nil(t, got[0].StopTimeUpdate)
		assert.Same(t, tu.StopTimeUpdate[0], got[1].StopTimeUpdate)
	})
}

// A stop time's estimates come from its own stop time update, or else from
// the delay carried from earlier stops.
func TestMatchRun_Estimates(t *testing.T) {
	stops := []StopRef{testStopRef(1, "A"), testStopRef(2, "B"), testStopRef(3, "C"), testStopRef(4, "D"), testStopRef(5, "E")}
	// late returns the estimates of stop time i delayed by d, or none for a
	// negative d.
	type est struct{ arr, dep int64 }
	late := func(i int, d int64) est {
		if d < 0 {
			return est{}
		}
		return est{testBase + stops[i].Arrival.Val + d, testBase + stops[i].Departure.Val + d}
	}
	none := est{}
	depA := testBase + stops[0].Departure.Val
	arrA := testBase + stops[0].Arrival.Val
	tcs := []struct {
		name  string
		delay *int32 // the trip's delay
		base  int64
		stus  []*pb.TripUpdate_StopTimeUpdate
		want  []est
	}{
		{
			name: "the nearest earlier delay carries, not the last",
			base: testBase,
			stus: []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(1, ""), delayOf(60)), withDeparture(stu(3, ""), delayOf(300))},
			want: []est{late(0, 60), late(1, 60), late(2, 300), late(3, 300), late(4, 300)},
		},
		{
			name:  "the trip's delay applies until the first update",
			delay: proto.Int32(120),
			base:  testBase,
			stus:  []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(3, ""), delayOf(300))},
			want:  []est{late(0, 120), late(1, 120), late(2, 300), late(3, 300), late(4, 300)},
		},
		{
			name:  "NO_DATA ends the carry",
			delay: proto.Int32(120),
			base:  testBase,
			stus:  []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(2, ""), delayOf(60)), withRelationship(stu(3, ""), pb.TripUpdate_StopTimeUpdate_NO_DATA)},
			want:  []est{late(0, 120), late(1, 60), none, none, none},
		},
		{
			name: "SKIPPED passes the carried delay on",
			base: testBase,
			stus: []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(1, ""), delayOf(60)), withRelationship(stu(2, ""), pb.TripUpdate_StopTimeUpdate_SKIPPED)},
			want: []est{late(0, 60), none, late(2, 60), late(3, 60), late(4, 60)},
		},
		{
			name: "a time carries the delay it implies",
			base: testBase,
			stus: []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(1, ""), timeOf(depA+180))},
			want: []est{{depA + 180, depA + 180}, late(1, 180), late(2, 180), late(3, 180), late(4, 180)},
		},
		{
			name: "an arrival stands in for a missing departure",
			base: testBase,
			stus: []*pb.TripUpdate_StopTimeUpdate{withArrival(stu(1, ""), timeOf(arrA+240))},
			want: []est{{arrA + 240, arrA + 240}, late(1, 240), late(2, 240), late(3, 240), late(4, 240)},
		},
		{
			name: "a departure delay carries over an arrival delay",
			base: testBase,
			stus: []*pb.TripUpdate_StopTimeUpdate{withDeparture(withArrival(stu(1, ""), delayOf(30)), delayOf(90))},
			want: []est{{arrA + 30, depA + 90}, late(1, 90), late(2, 90), late(3, 90), late(4, 90)},
		},
		{
			name: "a time outranks a delay on the same event",
			base: testBase,
			stus: []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(1, ""), &pb.TripUpdate_StopTimeEvent{Time: proto.Int64(depA + 100), Delay: proto.Int32(60)})},
			want: []est{{depA + 100, depA + 100}, late(1, 60), late(2, 60), late(3, 60), late(4, 60)},
		},
		{
			name:  "an update without events takes the carried delay",
			delay: proto.Int32(100),
			base:  testBase,
			stus:  []*pb.TripUpdate_StopTimeUpdate{stu(2, "")},
			want:  []est{late(0, 100), late(1, 100), late(2, 100), late(3, 100), late(4, 100)},
		},
		{
			name: "without a date, times pass through and delays estimate nothing",
			base: 0,
			stus: []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(1, ""), delayOf(60)), withDeparture(stu(2, ""), timeOf(5000))},
			want: []est{none, {5000, 5000}, none, none, none},
		},
		{
			name: "no update, no estimates",
			base: testBase,
			want: []est{none, none, none, none, none},
		},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tu := &pb.TripUpdate{Trip: &pb.TripDescriptor{TripId: proto.String("T")}, Delay: tc.delay, StopTimeUpdate: tc.stus}
			got := matchRun(context.Background(), tu, tc.base, stops)
			if !assert.Len(t, got, len(stops)) {
				return
			}
			for i, r := range got {
				assert.Equal(t, tc.want[i], est{r.EstimatedArrival, r.EstimatedDeparture}, "stop time %d", i)
			}
		})
	}
	// A stop time without a scheduled time gets no estimate from a delay, and
	// the delay carries past it.
	t.Run("a stop time without a scheduled time", func(t *testing.T) {
		untimed := []StopRef{testStopRef(1, "A"), {StopSequence: 2, StopID: "B"}, testStopRef(3, "C")}
		tu := &pb.TripUpdate{StopTimeUpdate: []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(1, ""), delayOf(60))}}
		got := matchRun(context.Background(), tu, testBase, untimed)
		assert.Equal(t, est{}, est{got[1].EstimatedArrival, got[1].EstimatedDeparture}, "untimed stop")
		assert.Equal(t, est{testBase + untimed[2].Arrival.Val + 60, testBase + untimed[2].Departure.Val + 60}, est{got[2].EstimatedArrival, got[2].EstimatedDeparture}, "the stop after")
	})
}

// A RunUpdate matches its first feed's trip update to stop times given by
// value, and FindStopTimeUpdates matches a trip's, asking the Lookup for them
// so that a caller holding only some of them still gets the carried delay.
func TestFinder_MatchStops(t *testing.T) {
	ctx := context.Background()
	loc := testLosAngeles(t)
	msg := &pb.FeedMessage{
		Header: &pb.FeedHeader{GtfsRealtimeVersion: proto.String("2.0")},
		Entity: []*pb.FeedEntity{{
			Id: proto.String("T"),
			TripUpdate: &pb.TripUpdate{
				Trip:           &pb.TripDescriptor{TripId: proto.String("T"), StartDate: proto.String("20180531")},
				StopTimeUpdate: []*pb.TripUpdate_StopTimeUpdate{withDeparture(stu(2, ""), delayOf(60))},
			},
		}},
	}
	stops := testTripStops(10*3600, "A", "B", "C", "D")
	dayStart := tt.ServiceDayStart(testDate(31), loc).Unix()

	t.Run("MatchStops", func(t *testing.T) {
		f := testFinder(t, feedVersionLookup{unexpectedLookup: unexpectedLookup{t}}, "realtime_trip_updates", msg)
		ru := f.FindRunUpdate(ctx, Run{FeedVersionID: 1, TripID: "T", Date: testDate(31)})
		if !assert.NotNil(t, ru) {
			return
		}
		got := ru.MatchStops(ctx, dayStart, stops)
		if assert.Len(t, got, 4) {
			assert.Zero(t, got[0].EstimatedDeparture, "before the first update")
			assert.Equal(t, dayStart+stops[1].Departure.Val+60, got[1].EstimatedDeparture, "the update's stop")
			assert.Equal(t, dayStart+stops[3].Departure.Val+60, got[3].EstimatedDeparture, "carried")
		}
		assert.Nil(t, f.FindRunUpdate(ctx, Run{FeedVersionID: 1, TripID: "T", Date: testDate(30)}), "a run without an update")
	})
	t.Run("FindStopTimeUpdates", func(t *testing.T) {
		lookup := tripUpdateLookup{
			feedVersionLookup: feedVersionLookup{unexpectedLookup: unexpectedLookup{t}, tz: loc},
			stops:             map[int][]StopRef{7: stops},
		}
		f := testFinder(t, lookup, "realtime_trip_updates", msg)
		// Only the last two stop times, as a filtered query holds.
		got := f.FindStopTimeUpdates(ctx, testRun(7, "T", 31), []*model.StopTime{testStopTime(3), testStopTime(4)})
		if assert.Len(t, got, 2) && assert.NotNil(t, got[0]) && assert.NotNil(t, got[1]) {
			assert.Equal(t, dayStart+stops[2].Departure.Val+60, got[0].EstimatedDeparture)
			assert.Equal(t, dayStart+stops[3].Departure.Val+60, got[1].EstimatedDeparture)
		}
		// A stop time the trip doesn't have.
		got = f.FindStopTimeUpdates(ctx, testRun(7, "T", 31), []*model.StopTime{testStopTime(9)})
		if assert.Len(t, got, 1) {
			assert.Nil(t, got[0])
		}
		// A trip whose stop times the Lookup doesn't have isn't matched.
		_, ok := f.FindStopTimeUpdate(ctx, testRun(8, "T", 31), testStopTime(1))
		assert.False(t, ok)
		// A run without an update.
		for _, r := range f.FindStopTimeUpdates(ctx, testRun(7, "T", 30), []*model.StopTime{testStopTime(1)}) {
			assert.Nil(t, r)
		}
	})
}

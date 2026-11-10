package routematch

import (
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
)

var (
	day1 = time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)
	day2 = time.Date(2026, 11, 11, 0, 0, 0, 0, time.UTC)
	// Two ends of a line, about 11 km apart, and a point 30 m from the first.
	endA    = tlxy.Point{Lon: -123.26, Lat: 44.56}
	endA30m = tlxy.Point{Lon: -123.26, Lat: 44.56027}
	endB    = tlxy.Point{Lon: -123.10, Lat: 44.63}
	hms     = func(h, m, s int) int { return h*3600 + m*60 + s }
	trip    = func(d time.Time, start int, first tlxy.Point) TripStart {
		return TripStart{Date: d, Start: start, FirstStop: first, PatternKey: 1}
	}
	opts     = DefaultOptions()
	stopNear = func(id int, lon, lat float64) PatternStop {
		return PatternStop{StopID: id, Point: tlxy.Point{Lon: lon, Lat: lat}}
	}
)

func TestMatchTrips(t *testing.T) {
	t.Run("identical trips match one to one", func(t *testing.T) {
		a := []TripStart{trip(day1, hms(8, 0, 0), endA), trip(day1, hms(9, 0, 0), endA)}
		b := []TripStart{trip(day1, hms(9, 0, 0), endA30m), trip(day1, hms(8, 0, 0), endA30m)}
		assert.Len(t, MatchTrips(a, b, opts), 2)
	})
	t.Run("start within tolerance", func(t *testing.T) {
		a := []TripStart{trip(day1, hms(8, 0, 30), endA)}
		b := []TripStart{trip(day1, hms(8, 0, 0), endA)}
		assert.Len(t, MatchTrips(a, b, opts), 1)
		b = []TripStart{trip(day1, hms(8, 1, 31), endA)}
		assert.Len(t, MatchTrips(a, b, opts), 0)
	})
	t.Run("a trip from the other end of the line at the same time does not match", func(t *testing.T) {
		a := []TripStart{trip(day1, hms(8, 0, 0), endA)}
		b := []TripStart{trip(day1, hms(8, 0, 0), endB)}
		assert.Len(t, MatchTrips(a, b, opts), 0)
	})
	t.Run("one trip of b matches at most one trip of a", func(t *testing.T) {
		a := []TripStart{trip(day1, hms(8, 0, 0), endA), trip(day1, hms(8, 0, 50), endA)}
		b := []TripStart{trip(day1, hms(8, 0, 20), endA)}
		pairs := MatchTrips(a, b, opts)
		assert.Len(t, pairs, 1)
	})
	t.Run("the closest start is chosen", func(t *testing.T) {
		a := []TripStart{trip(day1, hms(8, 0, 0), endA)}
		b := []TripStart{trip(day1, hms(8, 0, 50), endA), trip(day1, hms(8, 0, 10), endA)}
		pairs := MatchTrips(a, b, opts)
		if assert.Len(t, pairs, 1) {
			assert.Equal(t, hms(8, 0, 10), pairs[0].B.Start)
		}
	})
	t.Run("after midnight on one date matches early morning on the next", func(t *testing.T) {
		a := []TripStart{trip(day1, hms(24, 15, 0), endA)}
		b := []TripStart{trip(day2, hms(0, 15, 0), endA)}
		assert.Len(t, MatchTrips(a, b, opts), 1)
	})
	t.Run("same time on different dates does not match", func(t *testing.T) {
		a := []TripStart{trip(day1, hms(8, 0, 0), endA)}
		b := []TripStart{trip(day2, hms(8, 0, 0), endA)}
		assert.Len(t, MatchTrips(a, b, opts), 0)
	})
}

func TestRelate(t *testing.T) {
	tcs := []struct {
		trips, other, matched int
		want                  Relationship
	}{
		{0, 0, 0, NoTrips},
		{0, 5, 0, NoTrips},
		{5, 0, 0, OtherNoTrips},
		{5, 5, 0, Disjoint},
		{10, 10, 10, SameTrips},
		// Two editions of one timetable a day apart: 572 of 579 and of 595.
		{579, 595, 572, SameTrips},
		// Coast to Valley Express: 6 trips, all among the other's 11.
		{6, 11, 6, Subset},
		{11, 6, 6, Superset},
		{10, 10, 5, Partial},
	}
	for _, tc := range tcs {
		assert.Equal(t, tc.want, Relate(tc.trips, tc.other, tc.matched, opts), "%d/%d/%d", tc.trips, tc.other, tc.matched)
	}
}

func TestStopCorrespondences(t *testing.T) {
	t.Run("pairs stops with different ids by place, counting matched trips", func(t *testing.T) {
		// Two publishers' copies of one route: different stop ids, stops a few
		// meters to 40 m apart.
		pa := []PatternStop{stopNear(1, -123.100, 44.630), stopNear(2, -123.200, 44.600), stopNear(3, -123.262, 44.5646)}
		pb := []PatternStop{stopNear(11, -123.1004, 44.6302), stopNear(12, -123.2001, 44.6001), stopNear(13, -123.2621, 44.5647)}
		pairs := []Pair{
			{A: TripStart{PatternKey: 1}, B: TripStart{PatternKey: 2}},
			{A: TripStart{PatternKey: 1}, B: TripStart{PatternKey: 2}},
		}
		got := StopCorrespondences(pairs, map[int][]PatternStop{1: pa}, map[int][]PatternStop{2: pb}, opts)
		if assert.Len(t, got, 3) {
			for _, c := range got {
				assert.Equal(t, 2, c.MatchedTrips)
				assert.Equal(t, c.StopID+10, c.OtherStopID)
				assert.Less(t, c.Distance, 50.0)
			}
		}
	})
	t.Run("a stop only one pattern serves pairs with nothing, and later stops still pair", func(t *testing.T) {
		// b has an extra stop 200 m past a's first stop, before the shared second stop.
		pa := []PatternStop{stopNear(1, -123.1000, 44.63), stopNear(2, -123.1050, 44.63)}
		pb := []PatternStop{stopNear(11, -123.1000, 44.63), stopNear(19, -123.1025, 44.63), stopNear(12, -123.1050, 44.63)}
		pairs := []Pair{{A: TripStart{PatternKey: 1}, B: TripStart{PatternKey: 2}}}
		got := StopCorrespondences(pairs, map[int][]PatternStop{1: pa}, map[int][]PatternStop{2: pb}, opts)
		ids := map[int]int{}
		for _, c := range got {
			ids[c.StopID] = c.OtherStopID
		}
		assert.Equal(t, map[int]int{1: 11, 2: 12}, ids)
	})
	t.Run("a loop pairs each visit to a stop with its own counterpart, counting the trip once", func(t *testing.T) {
		pa := []PatternStop{stopNear(1, -123.10, 44.63), stopNear(2, -123.11, 44.63), stopNear(1, -123.10, 44.63)}
		pb := []PatternStop{stopNear(11, -123.10, 44.63), stopNear(12, -123.11, 44.63), stopNear(11, -123.10, 44.63)}
		pairs := []Pair{{A: TripStart{PatternKey: 1}, B: TripStart{PatternKey: 2}}}
		got := StopCorrespondences(pairs, map[int][]PatternStop{1: pa}, map[int][]PatternStop{2: pb}, opts)
		counts := map[[2]int]int{}
		for _, c := range got {
			counts[[2]int{c.StopID, c.OtherStopID}] = c.MatchedTrips
		}
		assert.Equal(t, map[[2]int]int{{1, 11}: 1, {2, 12}: 1}, counts)
	})
	t.Run("stops too far apart do not pair", func(t *testing.T) {
		pa := []PatternStop{stopNear(1, -123.10, 44.63)}
		pb := []PatternStop{stopNear(11, -123.11, 44.63)}
		pairs := []Pair{{A: TripStart{PatternKey: 1}, B: TripStart{PatternKey: 2}}}
		assert.Empty(t, StopCorrespondences(pairs, map[int][]PatternStop{1: pa}, map[int][]PatternStop{2: pb}, opts))
	})
}

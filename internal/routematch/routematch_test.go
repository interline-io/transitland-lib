package routematch

import (
	"testing"

	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/stretchr/testify/assert"
)

var (
	// Two ends of a line, about 14 km apart, and a point 30 m from the first.
	endA    = tlxy.Point{Lon: -123.26, Lat: 44.56}
	endA30m = tlxy.Point{Lon: -123.26, Lat: 44.56027}
	endB    = tlxy.Point{Lon: -123.10, Lat: 44.63}
	opts    = DefaultOptions()
)

// A run starting at h:m:s on an arbitrary day; MatchTrips sees only instants.
func at(h, m, s int, first tlxy.Point) TripStart {
	return TripStart{Instant: int64(1_700_000_000 + h*3600 + m*60 + s), FirstStop: first, PatternKey: 1}
}

func stopNear(id int, lon, lat float64) PatternStop {
	return PatternStop{StopID: id, Point: tlxy.Point{Lon: lon, Lat: lat}}
}

func TestMatchTrips(t *testing.T) {
	t.Run("identical trips match one to one", func(t *testing.T) {
		a := []TripStart{at(8, 0, 0, endA), at(9, 0, 0, endA)}
		b := []TripStart{at(9, 0, 0, endA30m), at(8, 0, 0, endA30m)}
		assert.Len(t, MatchTrips(a, b, opts), 2)
	})
	t.Run("start within tolerance", func(t *testing.T) {
		a := []TripStart{at(8, 0, 30, endA)}
		assert.Len(t, MatchTrips(a, []TripStart{at(8, 0, 0, endA)}, opts), 1)
		assert.Len(t, MatchTrips(a, []TripStart{at(8, 1, 31, endA)}, opts), 0)
	})
	t.Run("a trip from the other end of the line at the same time does not match", func(t *testing.T) {
		assert.Len(t, MatchTrips([]TripStart{at(8, 0, 0, endA)}, []TripStart{at(8, 0, 0, endB)}, opts), 0)
	})
	t.Run("one trip of b matches at most one trip of a", func(t *testing.T) {
		a := []TripStart{at(8, 0, 0, endA), at(8, 0, 50, endA)}
		assert.Len(t, MatchTrips(a, []TripStart{at(8, 0, 20, endA)}, opts), 1)
	})
	t.Run("pairs as many trips as possible, the same count either way", func(t *testing.T) {
		// Pairing 08:00:00 with 08:00:30, the closer start, would leave
		// 08:01:00 without a partner.
		a := []TripStart{at(8, 0, 0, endA), at(8, 1, 0, endA)}
		b := []TripStart{at(8, 0, 30, endA), at(7, 59, 10, endA)}
		assert.Len(t, MatchTrips(a, b, opts), 2)
		assert.Len(t, MatchTrips(b, a, opts), 2)
	})
	t.Run("headways under the tolerance pair in order", func(t *testing.T) {
		var a, b []TripStart
		for i := 0; i < 30; i++ {
			a = append(a, at(7, 0, i*90, endA))
			b = append(b, at(7, 0, i*90+40, endA30m))
		}
		assert.Len(t, MatchTrips(a, b, opts), 30)
		assert.Len(t, MatchTrips(b, a, opts), 30)
	})
}

func TestRelate(t *testing.T) {
	tcs := []struct {
		trips, other, matched, otherMatched int
		want                                Relationship
	}{
		{0, 0, 0, 0, NoTrips},
		{0, 5, 0, 0, NoTrips},
		{5, 0, 0, 0, OtherNoTrips},
		{5, 5, 0, 0, Disjoint},
		{10, 10, 10, 10, SameTrips},
		// Two editions of one timetable a day apart: 572 of 579 and of 595.
		{579, 595, 572, 572, SameTrips},
		// Coast to Valley Express: 6 trips, all among the other's 11.
		{6, 11, 6, 6, Subset},
		{11, 6, 6, 6, Superset},
		{10, 10, 5, 5, Partial},
		// After-midnight trips matched across the compared dates' edges can
		// leave the two counts apart.
		{10, 10, 10, 9, SameTrips},
		{10, 10, 10, 8, Subset},
	}
	for _, tc := range tcs {
		assert.Equal(t, tc.want, Relate(tc.trips, tc.other, tc.matched, tc.otherMatched, opts), "%+v", tc)
	}
}

func TestStopCorrespondences(t *testing.T) {
	one := func(pa, pb []PatternStop, n int) []Correspondence {
		var pairs []Pair
		for i := 0; i < n; i++ {
			pairs = append(pairs, Pair{A: TripStart{PatternKey: 1}, B: TripStart{PatternKey: 2}})
		}
		return StopCorrespondences(pairs, map[int][]PatternStop{1: pa}, map[int][]PatternStop{2: pb}, opts)
	}
	t.Run("pairs stops with different ids by place, counting matched trips", func(t *testing.T) {
		// Two publishers' copies of one route: different stop ids, a few meters
		// to 40 m apart.
		pa := []PatternStop{stopNear(1, -123.100, 44.630), stopNear(2, -123.200, 44.600), stopNear(3, -123.262, 44.5646)}
		pb := []PatternStop{stopNear(11, -123.1004, 44.6302), stopNear(12, -123.2001, 44.6001), stopNear(13, -123.2621, 44.5647)}
		got := one(pa, pb, 2)
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
		ids := map[int]int{}
		for _, c := range one(pa, pb, 1) {
			ids[c.StopID] = c.OtherStopID
		}
		assert.Equal(t, map[int]int{1: 11, 2: 12}, ids)
	})
	t.Run("more pairs within reach beat fewer closer ones", func(t *testing.T) {
		// a at 0, 290 and 580 m along a line; b at -290, 0 and 290 m. Three
		// pairs, each within 300 m, beat two exact ones.
		const mPerDeg = 79_000.0 // meters per degree of longitude near 44.6 N, roughly
		lon := func(m float64) float64 { return -123.1 + m/mPerDeg }
		pa := []PatternStop{stopNear(1, lon(0), 44.6), stopNear(2, lon(290), 44.6), stopNear(3, lon(580), 44.6)}
		pb := []PatternStop{stopNear(11, lon(-290), 44.6), stopNear(12, lon(0), 44.6), stopNear(13, lon(290), 44.6)}
		assert.Len(t, one(pa, pb, 1), 3)
	})
	t.Run("a loop pairs each visit to a stop with its own counterpart, counting the trip once", func(t *testing.T) {
		pa := []PatternStop{stopNear(1, -123.10, 44.63), stopNear(2, -123.11, 44.63), stopNear(1, -123.10, 44.63)}
		pb := []PatternStop{stopNear(11, -123.10, 44.63), stopNear(12, -123.11, 44.63), stopNear(11, -123.10, 44.63)}
		counts := map[[2]int]int{}
		for _, c := range one(pa, pb, 1) {
			counts[[2]int{c.StopID, c.OtherStopID}] = c.MatchedTrips
		}
		assert.Equal(t, map[[2]int]int{{1, 11}: 1, {2, 12}: 1}, counts)
	})
	t.Run("stops too far apart do not pair", func(t *testing.T) {
		assert.Empty(t, one([]PatternStop{stopNear(1, -123.10, 44.63)}, []PatternStop{stopNear(11, -123.11, 44.63)}, 1))
	})
}

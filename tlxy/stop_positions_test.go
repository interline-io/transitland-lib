package tlxy

import (
	"math"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stopTestOrigin = Point{Lon: -122.4, Lat: 37.8}

// metersLine's meters match the haversine measure to within a few
// centimeters across these fixtures.
const fixtureDelta = 0.05

// metersLine is a line through points given in meters east and north of
// stopTestOrigin.
func metersLine(pts ...[2]float64) []Point {
	ap := NewApprox(stopTestOrigin)
	ret := make([]Point, len(pts))
	for i, p := range pts {
		ret[i] = Point{Lon: stopTestOrigin.Lon + p[0]/ap.LonMeters(), Lat: stopTestOrigin.Lat + p[1]/ap.LatMeters()}
	}
	return ret
}

// square is a 100 m square loop with its lower-left corner at (x0, y0), closed
// back to the start.
func square(x0, y0 float64) [][2]float64 {
	return [][2]float64{{x0, y0}, {x0 + 100, y0}, {x0 + 100, y0 + 100}, {x0, y0 + 100}, {x0, y0}}
}

// twoLaps is a loop route drawn twice, the second lap shifted 2 m east and
// north, so the second pass of the bottom edge sits at y=2.
func twoLaps() []Point {
	return metersLine(append(square(0, 0), square(2, 2)...)...)
}

// boulevard runs out along y=0 and back along y=30, with a stop every 100 m
// each way and dists in meters along the shape.
//
// The stop at x=500 serves the trip out but sits 25 m from the way out and 5 m
// from the way back, farther apart than the lap tolerance.
func boulevard() (shape []Point, stops []Point, dists []float64) {
	shape = metersLine([2]float64{0, 0}, [2]float64{1000, 0}, [2]float64{1000, 30}, [2]float64{0, 30})
	for x := 0.0; x <= 1000; x += 100 {
		y := 0.0
		if x == 500 {
			y = 25
		}
		stops = append(stops, metersLine([2]float64{x, y})...)
		dists = append(dists, x)
	}
	for x := 900.0; x >= 0; x -= 100 {
		stops = append(stops, metersLine([2]float64{x, 30})...)
		dists = append(dists, 2030-x)
	}
	return shape, stops, dists
}

// straightLine is a 1 km line along y=0 with a stop every 100 m, and dists in
// meters along the line.
func straightLine() (shape []Point, stops []Point, dists []float64) {
	shape = metersLine([2]float64{0, 0}, [2]float64{1000, 0})
	for x := 0.0; x <= 1000; x += 100 {
		stops = append(stops, metersLine([2]float64{x, 0})...)
		dists = append(dists, x)
	}
	return shape, stops, dists
}

// assertInOrder checks that each placed stop is at or after the one before.
func assertInOrder(t *testing.T, got []StopPosition) {
	t.Helper()
	prev := 0.0
	for i, p := range got {
		if !p.Placed {
			continue
		}
		assert.GreaterOrEqual(t, p.Position, prev, "stop %d", i)
		prev = p.Position
	}
}

// assertPositions checks that each stop sits at want, and that the hint placed
// every stop but those in unhinted.
func assertPositions(t *testing.T, got []StopPosition, want []float64, unhinted ...int) {
	t.Helper()
	require.Len(t, got, len(want))
	for i := range got {
		assert.InDelta(t, want[i], got[i].Position, fixtureDelta, "stop %d", i)
		assert.Equal(t, !slices.Contains(unhinted, i), got[i].Hinted, "stop %d", i)
	}
}

func TestStopPositions_Laps(t *testing.T) {
	stops := metersLine([2]float64{10, 3}, [2]float64{50, 3}, [2]float64{100, 50}, [2]float64{50, 103}, [2]float64{0, 50}, [2]float64{50, 3}, [2]float64{100, 50})
	got := StopPositions(twoLaps(), stops, nil)
	require.Len(t, got, len(stops))
	assertInOrder(t, got)
	// The second stop is 3 m from the first pass and 1 m from the second, and
	// stays on the first lap; its repeat lands on the second.
	assert.InDelta(t, 50, got[1].Position, fixtureDelta)
	assert.InDelta(t, 3, got[1].Distance, fixtureDelta)
	assert.Greater(t, got[5].Position, 400.0)
	// The last stop is 2 m from the second lap's right edge, and stays there
	// rather than on the top edge 52 m away.
	assert.InDelta(t, 2, got[6].Distance, fixtureDelta)
}

func TestStopPositions_FirstStop(t *testing.T) {
	// A loop whose shape starts 20 m from its terminal but ends 2 m from it.
	shape := metersLine([2]float64{20, 0}, [2]float64{100, 0}, [2]float64{100, 100}, [2]float64{0, 100}, [2]float64{0, 2})
	stops := metersLine([2]float64{0, 0}, [2]float64{100, 50}, [2]float64{50, 100}, [2]float64{0, 0})
	got := StopPositions(shape, stops, nil)
	assert.InDelta(t, 0, got[0].Position, fixtureDelta)
	assert.InDelta(t, 20, got[0].Distance, fixtureDelta)
	assert.InDelta(t, 378, got[3].Position, fixtureDelta)
	assertInOrder(t, got)
}

func TestStopPositions_LastStop(t *testing.T) {
	// The shape passes its last stop 11 m away, runs around a loop, and ends
	// 6 m from the stop. The hint agrees, and its scale doesn't come from the
	// first pass.
	shape := metersLine([2]float64{0, 0}, [2]float64{1100, 0}, [2]float64{1100, 200}, [2]float64{994, 200}, [2]float64{994, 11})
	stops := metersLine([2]float64{0, 0}, [2]float64{500, 0}, [2]float64{1000, 11})
	end := ShapeDistances(shape)[4]
	for _, dists := range [][]float64{nil, {0, 500, end}} {
		got := StopPositions(shape, stops, dists)
		assert.InDelta(t, end, got[2].Position, fixtureDelta)
		assert.InDelta(t, 6, got[2].Distance, fixtureDelta)
		assert.Equal(t, dists != nil, got[2].Hinted)
	}
}

func TestStopPositions_LastStopLayover(t *testing.T) {
	// The shape passes its last stop 11.5 m away, then runs on to a layover
	// and ends 38 m from the stop, farther than the lap tolerance.
	shape := metersLine([2]float64{0, 0}, [2]float64{1000, 0}, [2]float64{1000, 60}, [2]float64{900, 60}, [2]float64{900, 49.5})
	stops := metersLine([2]float64{0, 0}, [2]float64{500, 0}, [2]float64{900, 11.5})
	got := StopPositions(shape, stops, nil)
	assert.InDelta(t, 900, got[2].Position, fixtureDelta)
	assert.InDelta(t, 11.5, got[2].Distance, fixtureDelta)
}

func TestStopPositions_FirstStopStart(t *testing.T) {
	// The shape starts 12.3 m from its first stop, bends away, and comes back
	// within 10.8 m of the stop 31 m along. The shape's start stays a choice,
	// and is the earliest within the search radius of the nearest.
	shape := metersLine([2]float64{0, 12.3}, [2]float64{10, 20}, [2]float64{-5, 9.6}, [2]float64{-205, 9.6})
	stops := metersLine([2]float64{0, 0}, [2]float64{-105, 9.6}, [2]float64{-205, 9.6})
	sd := ShapeDistances(shape)
	for _, dists := range [][]float64{nil, {0, sd[2] + 100, sd[3]}} {
		got := StopPositions(shape, stops, dists)
		assert.InDelta(t, 0, got[0].Position, fixtureDelta)
		assert.InDelta(t, 12.3, got[0].Distance, fixtureDelta)
	}
}

func TestStopPositions_SmallLoop(t *testing.T) {
	// The shape passes the middle stop 9 m away, runs around a 20 m loop, and
	// passes it again 0.4 m away. The loop never takes the shape 30 m farther
	// from the stop, so both are one pass, at its nearest point.
	shape := metersLine([2]float64{-50, 9}, [2]float64{10, 9}, [2]float64{10, 25}, [2]float64{-10, 25}, [2]float64{-10, 0.4}, [2]float64{50, 0.4})
	stops := metersLine([2]float64{-50, 9}, [2]float64{0, 0}, [2]float64{50, 0.4})
	got := StopPositions(shape, stops, nil)
	assert.InDelta(t, ShapeDistances(shape)[4]+10, got[1].Position, fixtureDelta)
	assert.InDelta(t, 0.4, got[1].Distance, fixtureDelta)
}

func TestStopPositions_TurnBack(t *testing.T) {
	// The shape runs out 1 km and turns back at a single point, converging on
	// the way out: the stop at x=300 sits 3 m from the way out and 2.6 m from
	// the way back, and has a pass on each.
	shape := metersLine([2]float64{0, 0}, [2]float64{1000, 0}, [2]float64{0, 8})
	var stops []Point
	var want []float64
	for x := 0.0; x <= 1000; x += 100 {
		y := 0.0
		if x == 300 {
			y = 3
		}
		stops = append(stops, metersLine([2]float64{x, y})...)
		want = append(want, x)
	}
	for x := 900.0; x >= 0; x -= 100 {
		stops = append(stops, metersLine([2]float64{x, 8 * (1000 - x) / 1000})...)
		want = append(want, 2000-x)
	}
	got := StopPositions(shape, stops, nil)
	for i := range got {
		assert.InDelta(t, want[i], got[i].Position, 0.1, "stop %d", i)
	}
}

func TestStopPositions_RepeatedPoint(t *testing.T) {
	// A repeated shape point is no pass: the stop's nearest point is on the
	// segment after the repeat, 17 m away, not at the repeat, 22.7 m away.
	shape := metersLine([2]float64{0, 0}, [2]float64{100, 0}, [2]float64{100, 0}, [2]float64{200, 0})
	stops := metersLine([2]float64{0, 0}, [2]float64{115, 17}, [2]float64{200, 0})
	got := StopPositions(shape, stops, nil)
	assert.InDelta(t, 115, got[1].Position, fixtureDelta)
	assert.InDelta(t, 17, got[1].Distance, fixtureDelta)
}

func TestStopPositions_Hint(t *testing.T) {
	shape, stops, dists := boulevard()
	// Geometry alone puts the stop at x=500 on the way back, and drags the
	// rest of the way out after that stop.
	geo := StopPositions(shape, stops, nil)
	assert.InDelta(t, 1530, geo[5].Position, fixtureDelta)
	assertInOrder(t, geo)

	got := StopPositions(shape, stops, dists)
	assertPositions(t, got, dists)
	assert.InDelta(t, 25, got[5].Distance, fixtureDelta)
}

func TestStopPositions_HintFirstStop(t *testing.T) {
	// The shape leads in east, then turns back west past the first stop, which
	// is 35 m from the shape's start but 5 m from the way back. Stops follow
	// every 100 m for 2 km.
	shape := metersLine([2]float64{0, 0}, [2]float64{100, 0}, [2]float64{100, 40}, [2]float64{-2100, 40})
	stops := metersLine([2]float64{0, 35})
	dists := []float64{240}
	for x := -100.0; x >= -2000; x -= 100 {
		stops = append(stops, metersLine([2]float64{x, 40})...)
		dists = append(dists, 240-x)
	}
	geo := StopPositions(shape, stops, nil)
	assert.InDelta(t, 0, geo[0].Position, fixtureDelta)
	got := StopPositions(shape, stops, dists)
	assertPositions(t, got, dists)
	assert.InDelta(t, 5, got[0].Distance, fixtureDelta)
}

func TestStopPositions_HintSpike(t *testing.T) {
	// The shape runs out to a 41 m spike at x=500 and back before going on.
	// The feed puts the second stop at the spike's tip, 41 m from the stop,
	// but its own pass is 0.6 m away.
	shape := metersLine([2]float64{0, 0}, [2]float64{500, 0}, [2]float64{540, 10}, [2]float64{500, 0}, [2]float64{1000, 0})
	stops := metersLine([2]float64{0, 0}, [2]float64{580, 0.6}, [2]float64{1000, 0})
	sd := ShapeDistances(shape)
	dists := []float64{0, sd[2], sd[4]}
	got := StopPositions(shape, stops, dists)
	assert.InDelta(t, sd[3]+80, got[1].Position, fixtureDelta)
	assert.InDelta(t, 0.6, got[1].Distance, fixtureDelta)
	assert.True(t, got[1].Hinted)
}

func TestStopPositions_HintScale(t *testing.T) {
	// The shape runs out 1 km, then around a 10.7 km loop that ends 2 m from
	// the stop at x=300, which sits 40 m from the way out. Geometry alone puts that
	// stop at the shape's end and drags every stop after it there, but the
	// hint's scale comes from each end stop placed on its own.
	shape, stops, dists := straightLine()
	shape = append(shape, metersLine([2]float64{1000, 5000}, [2]float64{300, 5000}, [2]float64{300, 42})...)
	stops[3] = metersLine([2]float64{300, 40})[0]
	geo := StopPositions(shape, stops, nil)
	assert.Greater(t, geo[3].Position, 10000.0)
	assertPositions(t, StopPositions(shape, stops, dists), dists)
}

func TestStopPositions_HintUnits(t *testing.T) {
	shape, stops, dists := boulevard()
	feet := make([]float64, len(dists))
	for i, d := range dists {
		feet[i] = d * 3.28084
	}
	assert.Equal(t, StopPositions(shape, stops, dists), StopPositions(shape, stops, feet))
}

func TestStopPositions_UnusableHint(t *testing.T) {
	shape, stops, dists := boulevard()
	geo := StopPositions(shape, stops, nil)
	falling := slices.Clone(dists)
	falling[7] = falling[6] - 1
	assert.Equal(t, geo, StopPositions(shape, stops, falling))
	assert.Equal(t, geo, StopPositions(shape, stops, make([]float64, len(stops))))
	assert.Equal(t, geo, StopPositions(shape, stops, dists[1:]))
	nan := slices.Clone(dists)
	nan[3] = math.NaN()
	assert.Equal(t, geo, StopPositions(shape, stops, nan))
}

func TestStopPositions_HintFallsBack(t *testing.T) {
	// The feed puts the sixth stop 1 m past the fifth rather than 100 m, so no
	// pass is near where it says; that stop falls back to geometry alone.
	shape, stops, want := straightLine()
	dists := slices.Clone(want)
	dists[5] = dists[4] + 1
	assertPositions(t, StopPositions(shape, stops, dists), want, 5)
}

func TestStopPositions_HintFallbackBetween(t *testing.T) {
	// As in the boulevard, with the stop at x=500 put 1 m past the one before.
	// Geometry alone would take its pass on the way back, but the stop stays
	// short of where the hint finds the next one, on the way out.
	shape, stops, want := boulevard()
	dists := slices.Clone(want)
	dists[5] = dists[4] + 1
	assertPositions(t, StopPositions(shape, stops, dists), want, 5)
}

func TestStopPositions_HintFallbackHolds(t *testing.T) {
	// The shape runs out along y=0 and back along y=10. The feed's values for
	// the third and fourth stops are wrong: the hint can't place the third, at
	// the turn, and expects the fourth, 5 m from both ways, on the way out.
	// Geometry alone places the third at the turn, which holds the fourth to
	// the way back.
	shape := metersLine([2]float64{0, 0}, [2]float64{1000, 0}, [2]float64{1000, 10}, [2]float64{0, 10})
	stops := metersLine([2]float64{0, 0}, [2]float64{400, 0}, [2]float64{1005, 5}, [2]float64{800, 5}, [2]float64{0, 10})
	got := StopPositions(shape, stops, []float64{0, 400, 550, 800, 2010})
	assert.InDelta(t, 1005, got[2].Position, 5+fixtureDelta)
	assert.False(t, got[2].Hinted)
	assert.InDelta(t, 1210, got[3].Position, fixtureDelta)
	assert.InDelta(t, 2010, got[4].Position, fixtureDelta)
}

func TestStopPositions_PastShapeEnd(t *testing.T) {
	// The shape ends at the third stop, and the last two lie 80 m north of
	// its end and 20 m past it. The hint can't place the fourth stop, and the
	// stretch where it finds the fifth runs past the shape's end.
	shape := metersLine([2]float64{0, 0}, [2]float64{1000, 0})
	stops := metersLine([2]float64{0, 0}, [2]float64{500, 0}, [2]float64{1000, 0}, [2]float64{1000, 80}, [2]float64{1020, 0})
	for _, dists := range [][]float64{nil, {0, 500, 1000, 1010, 1020}} {
		got := StopPositions(shape, stops, dists)
		require.Len(t, got, len(stops))
		for i, want := range []float64{0, 500, 1000, 1000, 1000} {
			assert.True(t, got[i].Placed, "stop %d", i)
			assert.InDelta(t, want, got[i].Position, fixtureDelta, "stop %d", i)
		}
	}
}

func TestStopPositions_FarStop(t *testing.T) {
	// The third stop is 2 km off the line, near its far end. It has no
	// position and doesn't hold back the stops after it, with or without the
	// hint.
	shape, stops, dists := straightLine()
	stops[2] = metersLine([2]float64{950, 2000})[0]
	for _, d := range [][]float64{nil, dists} {
		got := StopPositions(shape, stops, d)
		assert.False(t, got[2].Placed)
		for i := range got {
			if i != 2 {
				assert.True(t, got[i].Placed, "stop %d", i)
				assert.InDelta(t, float64(i)*100, got[i].Position, fixtureDelta, "stop %d", i)
			}
		}
	}
}

func TestStopPositions_NotFinite(t *testing.T) {
	shape, stops, dists := straightLine()
	for _, v := range []float64{math.NaN(), math.Inf(1)} {
		bad := slices.Clone(shape)
		bad[1].Lat = v
		assert.Nil(t, StopPositions(bad, stops, dists))
		assert.Nil(t, ShapeDistances(bad))
		badStops := slices.Clone(stops)
		badStops[3].Lon = v
		got := StopPositions(shape, badStops, dists)
		assert.False(t, got[3].Placed)
		assert.True(t, got[4].Placed)
		assert.InDelta(t, 400, got[4].Position, fixtureDelta)
	}
}

func TestStopPositions_ShortShape(t *testing.T) {
	stops := metersLine([2]float64{0, 0})
	assert.Nil(t, StopPositions(nil, stops, nil))
	assert.Nil(t, StopPositions(metersLine([2]float64{0, 0}), stops, nil))
	assert.Nil(t, StopPositions(metersLine([2]float64{0, 0}, [2]float64{0, 0}), stops, nil))
	assert.Empty(t, StopPositions(twoLaps(), nil, nil))
}

func TestStopPositions_Reverse(t *testing.T) {
	// A shape 190 km long, run each way: both measure the same, and each stop
	// sits as far from one end as from the other end of the reverse. The stops
	// are 11 m north of the shape's ends and the middle of each segment.
	shape := []Point{{-122.4, 37.8}, {-121.9, 38.3}, {-121.5, 38.6}, {-120.8, 38.9}}
	stops := []Point{shape[0]}
	for i := 1; i < len(shape); i++ {
		stops = append(stops, Point{Lon: (shape[i-1].Lon + shape[i].Lon) / 2, Lat: (shape[i-1].Lat + shape[i].Lat) / 2})
	}
	stops = append(stops, shape[len(shape)-1])
	for i := range stops {
		stops[i].Lat += 0.0001
	}
	rshape, rstops := slices.Clone(shape), slices.Clone(stops)
	slices.Reverse(rshape)
	slices.Reverse(rstops)
	fwd, rev := StopPositions(shape, stops, nil), StopPositions(rshape, rstops, nil)
	length := LengthHaversine(shape)
	assert.InDelta(t, length, ShapeDistances(rshape)[len(rshape)-1], 1e-6)
	for i := range fwd {
		j := len(rev) - 1 - i
		assert.InDelta(t, length, fwd[i].Position+rev[j].Position, 0.5, "stop %d", i)
		assert.InDelta(t, fwd[i].Distance, rev[j].Distance, 0.5, "stop %d", i)
	}
}

func TestStopPositions_Antimeridian(t *testing.T) {
	// A 2 km line across 180 degrees stays 2 km long.
	shape := []Point{{179.99, -17}, {-179.99, -17}}
	stops := []Point{{179.99, -17}, {180, -17.0001}, {-179.99, -17}}
	length := LengthHaversine(shape)
	assert.InDelta(t, 2127, length, 1)
	got := StopPositions(shape, stops, nil)
	assert.InDelta(t, 0, got[0].Position, fixtureDelta)
	assert.InDelta(t, length/2, got[1].Position, 1)
	assert.InDelta(t, 11.1, got[1].Distance, 0.1)
	assert.InDelta(t, length, got[2].Position, fixtureDelta)
}

func TestShapeDistances(t *testing.T) {
	shape := twoLaps()
	got := ShapeDistances(shape)
	require.Len(t, got, len(shape))
	for i, want := range []float64{0, 100, 200, 300, 400} {
		assert.InDelta(t, want, got[i], fixtureDelta, "point %d", i)
	}
	assert.Equal(t, LengthHaversine(shape), got[len(got)-1])
	// A stop on a point of a shape sits at that point's distance.
	bend := metersLine([2]float64{0, 0}, [2]float64{100, 0}, [2]float64{100, 100})
	pos := StopPositions(bend, metersLine([2]float64{100, 0}, [2]float64{100, 100}), nil)
	dists := ShapeDistances(bend)
	assert.InDelta(t, dists[1], pos[0].Position, 1e-6)
	assert.InDelta(t, dists[2], pos[1].Position, 1e-6)
	assert.Nil(t, ShapeDistances(nil))
}

package tlxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var stopTestOrigin = Point{Lon: -122.4, Lat: 37.8}

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

// twoLaps is a loop route drawn twice, the second lap 2 m inside the first,
// so the second pass of the bottom edge sits at y=2.
func twoLaps() []Point {
	return metersLine(append(square(0, 0), square(2, 2)...)...)
}

// boulevard runs out along y=0 and back along y=30, with a stop every 100 m
// each way and dists in meters along it. The stop at x=500 serves the trip
// out but sits 25 m from the way out and 5 m from the way back, farther apart
// than the lap tolerance.
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

func assertInOrder(t *testing.T, got []StopPosition) {
	t.Helper()
	for i := 1; i < len(got); i++ {
		assert.GreaterOrEqual(t, got[i].Position, got[i-1].Position, "stop %d", i)
	}
}

func TestStopPositions_Laps(t *testing.T) {
	stops := metersLine([2]float64{10, 3}, [2]float64{50, 3}, [2]float64{100, 50}, [2]float64{50, 103}, [2]float64{0, 50}, [2]float64{50, 3}, [2]float64{100, 50})
	got := StopPositions(twoLaps(), stops, nil)
	require.Len(t, got, len(stops))
	assertInOrder(t, got)
	// The second stop is 3 m from the first pass and 1 m from the second, and
	// stays on the first lap; its repeat lands on the second.
	assert.InDelta(t, 50, got[1].Position, 1e-6)
	assert.InDelta(t, 3, got[1].Distance, 1e-6)
	assert.Greater(t, got[5].Position, 400.0)
}

func TestStopPositions_FirstStop(t *testing.T) {
	// A loop whose shape starts 20 m from its terminal but ends 2 m from it.
	shape := metersLine([2]float64{20, 0}, [2]float64{100, 0}, [2]float64{100, 100}, [2]float64{0, 100}, [2]float64{0, 2})
	stops := metersLine([2]float64{0, 0}, [2]float64{100, 50}, [2]float64{50, 100}, [2]float64{0, 0})
	got := StopPositions(shape, stops, nil)
	assert.InDelta(t, 0, got[0].Position, 1e-6)
	assert.InDelta(t, 20, got[0].Distance, 1e-6)
	assert.InDelta(t, 378, got[3].Position, 1e-6)
	assertInOrder(t, got)
}

func TestStopPositions_Hint(t *testing.T) {
	shape, stops, dists := boulevard()
	// Geometry alone puts the stop at x=500 on the way back, and drags the
	// rest of the way out after it.
	geo := StopPositions(shape, stops, nil)
	assert.InDelta(t, 1530, geo[5].Position, 1e-6)
	assertInOrder(t, geo)

	got := StopPositions(shape, stops, dists)
	require.Len(t, got, len(stops))
	for i := range got {
		assert.InDelta(t, dists[i], got[i].Position, 1e-6, "stop %d", i)
		assert.True(t, got[i].Hinted, "stop %d", i)
	}
	assert.InDelta(t, 25, got[5].Distance, 1e-6)
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
	assert.InDelta(t, 0, geo[0].Position, 1e-6)
	got := StopPositions(shape, stops, dists)
	for i := range got {
		assert.InDelta(t, dists[i], got[i].Position, 1e-6, "stop %d", i)
		assert.True(t, got[i].Hinted, "stop %d", i)
	}
	assert.InDelta(t, 5, got[0].Distance, 1e-6)
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
	falling := append([]float64{}, dists...)
	falling[7] = falling[6] - 1
	assert.Equal(t, geo, StopPositions(shape, stops, falling))
	assert.Equal(t, geo, StopPositions(shape, stops, make([]float64, len(stops))))
	assert.Equal(t, geo, StopPositions(shape, stops, dists[1:]))
}

func TestStopPositions_HintFallsBack(t *testing.T) {
	// The feed puts the sixth stop 1 m past the fifth rather than 100 m, so no
	// pass is near where it says; that stop falls back to geometry alone.
	shape := metersLine([2]float64{0, 0}, [2]float64{1000, 0})
	var stops []Point
	var dists []float64
	for x := 0.0; x <= 1000; x += 100 {
		stops = append(stops, metersLine([2]float64{x, 0})...)
		dists = append(dists, x)
	}
	dists[5] = dists[4] + 1
	got := StopPositions(shape, stops, dists)
	for i := range got {
		assert.InDelta(t, float64(i)*100, got[i].Position, 1e-6, "stop %d", i)
	}
	assert.True(t, got[4].Hinted)
	assert.False(t, got[5].Hinted)
	assert.True(t, got[6].Hinted)
}

func TestStopPositions_ShortShape(t *testing.T) {
	stops := metersLine([2]float64{0, 0})
	assert.Nil(t, StopPositions(nil, stops, nil))
	assert.Nil(t, StopPositions(metersLine([2]float64{0, 0}), stops, nil))
	assert.Empty(t, StopPositions(twoLaps(), nil, nil))
}

func TestShapeDistances(t *testing.T) {
	shape := twoLaps()
	got := ShapeDistances(shape)
	require.Len(t, got, len(shape))
	for i, want := range []float64{0, 100, 200, 300, 400} {
		assert.InDelta(t, want, got[i], 1e-6, "point %d", i)
	}
	// A stop on a point of a shape sits at that point's distance.
	bend := metersLine([2]float64{0, 0}, [2]float64{100, 0}, [2]float64{100, 100})
	pos := StopPositions(bend, metersLine([2]float64{100, 0}, [2]float64{100, 100}), nil)
	dists := ShapeDistances(bend)
	assert.InDelta(t, dists[1], pos[0].Position, 1e-6)
	assert.InDelta(t, dists[2], pos[1].Position, 1e-6)
	assert.Nil(t, ShapeDistances(nil))
}

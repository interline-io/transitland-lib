package copier

import (
	"testing"
)

////////////////////

func Benchmark_segmentClosestPoint(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	q := unflattenCoordinates(l.FlatCoords())
	// get the midpoint
	sa := q[0]
	sb := q[1]
	p := [2]float64{sa[0] + (sb[0]-sa[0])/2, sa[1] + (sb[1]-sa[1])/2}
	for n := 0; n < b.N; n++ {
		segmentClosestPoint(sa, sb, p)
	}
}

func Benchmark_lineClosestPoint(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	q := unflattenCoordinates(l.FlatCoords())
	// get the midpoint of the two middle points
	sa := q[len(q)/2]
	sb := q[len(q)/2+1]
	p := [2]float64{sa[0] + (sb[0]-sa[0])/2, sa[1] + (sb[1]-sa[1])/2}
	for n := 0; n < b.N; n++ {
		lineClosestPoint(q, p)
	}
}

func Benchmark_linePositions(b *testing.B) {
	line, points := decodeGeojson(testPositions[0].Geojson)
	lc := unflattenCoordinates(line.FlatCoords())
	pp := [][2]float64{}
	for _, p := range points {
		pp = append(pp, [2]float64{p.FlatCoords()[0], p.FlatCoords()[1]})
	}
	var r []float64
	for n := 0; n < b.N; n++ {
		r = linePositions(lc, pp)
	}
	_ = r
}

func Benchmark_linePositionsFallback(b *testing.B) {
	_, points := decodeGeojson(testPositions[0].Geojson)
	pp := [][2]float64{}
	for _, p := range points {
		pp = append(pp, [2]float64{p.FlatCoords()[0], p.FlatCoords()[1]})
	}
	var r []float64
	for n := 0; n < b.N; n++ {
		r = linePositionsFallback(pp)
	}
	_ = r
}

func Benchmark_distance2d(b *testing.B) {
	dp := testDistancePoints[0]
	var r float64
	for n := 0; n < b.N; n++ {
		r = distance2d(dp.orig, dp.dest)
	}
	_ = r
}

func Benchmark_distanceHaversine(b *testing.B) {
	dp := testDistancePoints[0]
	var r float64
	for n := 0; n < b.N; n++ {
		r = distanceHaversine(dp.orig, dp.dest)
	}
	_ = r
}

func Benchmark_length2d(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	line := unflattenCoordinates(l.FlatCoords())
	var r float64
	for n := 0; n < b.N; n++ {
		r = length2d(line)
	}
	_ = r
}

func Benchmark_lengthHaversine(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	line := unflattenCoordinates(l.FlatCoords())
	var r float64
	for n := 0; n < b.N; n++ {
		r = lengthHaversine(line)
	}
	_ = r
}

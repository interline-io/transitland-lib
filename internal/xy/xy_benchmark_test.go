package xy

import (
	"testing"
)

////////////////////

func BenchmarkSegmentClosestPoint(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	q := unflattenCoordinates(l.FlatCoords())
	// get the midpoint
	sa := q[0]
	sb := q[1]
	p := [2]float64{sa[0] + (sb[0]-sa[0])/2, sa[1] + (sb[1]-sa[1])/2}
	for n := 0; n < b.N; n++ {
		SegmentClosestPoint(sa, sb, p)
	}
}

func BenchmarkLineClosestPoint(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	q := unflattenCoordinates(l.FlatCoords())
	// get the midpoint of the two middle points
	sa := q[len(q)/2]
	sb := q[len(q)/2+1]
	p := [2]float64{sa[0] + (sb[0]-sa[0])/2, sa[1] + (sb[1]-sa[1])/2}
	for n := 0; n < b.N; n++ {
		LineClosestPoint(q, p)
	}
}

func BenchmarkLinePositions(b *testing.B) {
	line, points := decodeGeojson(testPositions[0].Geojson)
	lc := unflattenCoordinates(line.FlatCoords())
	pp := [][2]float64{}
	for _, p := range points {
		pp = append(pp, [2]float64{p.FlatCoords()[0], p.FlatCoords()[1]})
	}
	var r []float64
	for n := 0; n < b.N; n++ {
		r = LinePositions(lc, pp)
	}
	_ = r
}

func BenchmarkLinePositionsFallback(b *testing.B) {
	_, points := decodeGeojson(testPositions[0].Geojson)
	pp := [][2]float64{}
	for _, p := range points {
		pp = append(pp, [2]float64{p.FlatCoords()[0], p.FlatCoords()[1]})
	}
	var r []float64
	for n := 0; n < b.N; n++ {
		r = LinePositionsFallback(pp)
	}
	_ = r
}

func BenchmarkDistance2d(b *testing.B) {
	dp := testDistancePoints[0]
	var r float64
	for n := 0; n < b.N; n++ {
		r = Distance2d(dp.orig, dp.dest)
	}
	_ = r
}

func BenchmarkDistanceHaversine(b *testing.B) {
	dp := testDistancePoints[0]
	var r float64
	for n := 0; n < b.N; n++ {
		r = DistanceHaversine(dp.orig[0], dp.orig[1], dp.dest[0], dp.dest[1])
	}
	_ = r
}

func BenchmarkLength2d(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	line := unflattenCoordinates(l.FlatCoords())
	var r float64
	for n := 0; n < b.N; n++ {
		r = Length2d(line)
	}
	_ = r
}

func BenchmarkLengthHaversine(b *testing.B) {
	l, _ := decodeGeojson(testLines[0].Geojson)
	line := unflattenCoordinates(l.FlatCoords())
	var r float64
	for n := 0; n < b.N; n++ {
		r = LengthHaversine(line)
	}
	_ = r
}

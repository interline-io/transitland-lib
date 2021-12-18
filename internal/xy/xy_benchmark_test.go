package xy

import (
	"testing"
)

////////////////////

func BenchmarkSegmentClosestPoint(b *testing.B) {
	l, _, err := decodeGeojson(testLines[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	q := unflattenCoordinates(l.FlatCoords())
	// get the midpoint
	sa := q[0]
	sb := q[1]
	p := Point{sa.Lon + (sb.Lon-sa.Lon)/2, sa.Lat + (sb.Lat-sa.Lat)/2}
	for n := 0; n < b.N; n++ {
		SegmentClosestPoint(sa, sb, p)
	}
}

func BenchmarkLineClosestPoint(b *testing.B) {
	l, _, err := decodeGeojson(testLines[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	q := unflattenCoordinates(l.FlatCoords())
	// get the midpoint of the two middle points
	sa := q[len(q)/2]
	sb := q[len(q)/2+1]
	p := Point{sa.Lon + (sb.Lon-sa.Lon)/2, sa.Lat + (sb.Lat-sa.Lat)/2}
	for n := 0; n < b.N; n++ {
		LineClosestPoint(q, p)
	}
}

func BenchmarkLinePositions(b *testing.B) {
	line, points, err := decodeGeojson(testPositions[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	lc := unflattenCoordinates(line.FlatCoords())
	pp := []Point{}
	for _, p := range points {
		pp = append(pp, Point{p.FlatCoords()[0], p.FlatCoords()[1]})
	}
	var r []float64
	for n := 0; n < b.N; n++ {
		r = LinePositions(lc, pp)
	}
	_ = r
}

func BenchmarkLinePositionsFallback(b *testing.B) {
	_, points, err := decodeGeojson(testPositions[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	pp := []Point{}
	for _, p := range points {
		pp = append(pp, Point{p.FlatCoords()[0], p.FlatCoords()[1]})
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
		r = DistanceHaversine(dp.orig.Lon, dp.orig.Lat, dp.dest.Lon, dp.dest.Lat)
	}
	_ = r
}

func BenchmarkLength2d(b *testing.B) {
	l, _, err := decodeGeojson(testLines[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	line := unflattenCoordinates(l.FlatCoords())
	var r float64
	for n := 0; n < b.N; n++ {
		r = Length2d(line)
	}
	_ = r
}

func BenchmarkLengthHaversine(b *testing.B) {
	l, _, err := decodeGeojson(testLines[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	line := unflattenCoordinates(l.FlatCoords())
	var r float64
	for n := 0; n < b.N; n++ {
		r = LengthHaversine(line)
	}
	_ = r
}

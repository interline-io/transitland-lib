package tlxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type pt = Point

func TestSegmentClosestPoint(t *testing.T) {
	tcs := []struct {
		p      pt
		expect pt
		a1     pt
		a2     pt
	}{
		{pt{0, 0}, pt{0, 0}, pt{0, 0}, pt{1, 1}},
		{pt{0.5, 0.5}, pt{0, 0.5}, pt{0, 0}, pt{0, 1}},
		{pt{0.5, 0.5}, pt{0.5, 0.5}, pt{0, 0}, pt{1, 1}},
		{pt{20, 20}, pt{20, 20}, pt{10, 10}, pt{30, 30}},
		{pt{20, 20}, pt{0, 20}, pt{0, 0}, pt{0, 30}},
		{pt{-20, -20}, pt{-20, -10}, pt{-100, -10}, pt{-1, -10}},
	}
	for _, tc := range tcs {
		t.Run("", func(t *testing.T) {
			cp, d := SegmentClosestPoint(tc.a1, tc.a2, tc.p)
			_ = d
			assert.InDelta(t, tc.expect.Lon, cp.Lon, 0.001, "lon")
			assert.InDelta(t, tc.expect.Lat, cp.Lat, 0.001, "lat")
		})
	}
}

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

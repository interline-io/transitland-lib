package tlxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLineRelativePositions(t *testing.T) {
	for _, dp := range testPositions {
		line, points, err := decodeGeojson(dp.Geojson)
		if err != nil {
			t.Fatal(err)
		}
		lc := unflattenCoordinates(line.FlatCoords())
		pp := []Point{}
		for _, p := range points {
			pp = append(pp, Point{p.FlatCoords()[0], p.FlatCoords()[1]})
		}
		pos := LineRelativePositions(lc, pp)
		if len(pos) != len(dp.Positions) {
			t.Errorf("expect %d positions, got %d", len(dp.Positions), len(pos))
			continue
		}
		for i := 0; i < len(pos); i++ {
			testApproxEqual(t, pos[i], dp.Positions[i])
		}
	}
}

func TestLineRelativePositionsFallback(t *testing.T) {
	for _, dp := range testPositions {
		_, points, err := decodeGeojson(dp.Geojson)
		if err != nil {
			t.Fatal(err)
		}
		pp := []Point{}
		for _, p := range points {
			pp = append(pp, Point{p.FlatCoords()[0], p.FlatCoords()[1]})
		}
		pos := LineRelativePositionsFallback(pp)
		if len(pos) != len(dp.FallbackPositions) {
			t.Errorf("expect %d positions, got %d", len(dp.FallbackPositions), len(pos))
			continue
		}
		for i := 0; i < len(pos); i++ {
			testApproxEqual(t, pos[i], dp.FallbackPositions[i])
		}
	}
}

func TestContains(t *testing.T) {
	testcases := []struct {
		name   string
		a      []Point
		b      []Point
		expect bool
	}{
		{
			"basic",
			[]Point{{0, 1}, {0, 2}},
			[]Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
			true,
		},
		{
			"one point",
			[]Point{{0, 1}},
			[]Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
			true,
		},
		{
			"equal",
			[]Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
			[]Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
			true,
		},
		{
			"not quite equal",
			[]Point{{0, 0}, {0, 2}, {0, 2}, {0, 3}},
			[]Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
			false,
		},
		{
			"longer",
			[]Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
			[]Point{{0, 0}, {0, 1}, {0, 2}},
			false,
		},
		{
			"does not contain",
			[]Point{{0, 1}, {0, 4}},
			[]Point{{0, 0}, {0, 1}, {0, 2}, {0, 3}},
			false,
		},
		{
			"false start",
			[]Point{{0, 1}, {0, 2}},
			[]Point{{0, 0}, {0, 1}, {0, 0}, {0, 2}, {0, 3}},
			false,
		},
		{
			"false start 2",
			[]Point{{0, 1}, {0, 2}},
			[]Point{{0, 0}, {0, 1}, {0, 0}, {0, 1}, {0, 2}, {0, 3}},
			true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			if LineContains(tc.a, tc.b) != tc.expect {
				t.Errorf("expected %t", tc.expect)
			}
		})
	}
}

func TestDecodePolyline(t *testing.T) {
	check := "yfttIf{jR?B@BBDD?@ANDH@XHrAj@t@Z"
	expect := []Point{
		{Lon: -3.1738, Lat: 55.97821},
		{Lon: -3.17382, Lat: 55.97821},
		{Lon: -3.17384, Lat: 55.978199},
		{Lon: -3.17387, Lat: 55.978179},
		{Lon: -3.17387, Lat: 55.978149},
		{Lon: -3.17386, Lat: 55.978139},
		{Lon: -3.17389, Lat: 55.978059},
		{Lon: -3.17390, Lat: 55.978009},
		{Lon: -3.17395, Lat: 55.977879},
		{Lon: -3.17417, Lat: 55.977459},
		{Lon: -3.17431, Lat: 55.977189},
	}
	p, err := DecodePolyline(check)
	if err != nil {
		t.Fatal(err)
	}
	if len(expect) != len(p) {
		t.Fatal("unequal length")
	}
	for i := range expect {
		assert.InDelta(t, expect[i].Lon, p[i].Lon, 0.001)
		assert.InDelta(t, expect[i].Lat, p[i].Lat, 0.001)
	}
}

func TestEncodePolyline(t *testing.T) {
	expect := "yfttIf{jR?B@BBDD?@ANDH@XHrAj@t@Z"
	check := []Point{
		{Lon: -3.1738, Lat: 55.97821},
		{Lon: -3.17382, Lat: 55.97821},
		{Lon: -3.17384, Lat: 55.978199},
		{Lon: -3.17387, Lat: 55.978179},
		{Lon: -3.17387, Lat: 55.978149},
		{Lon: -3.17386, Lat: 55.978139},
		{Lon: -3.17389, Lat: 55.978059},
		{Lon: -3.17390, Lat: 55.978009},
		{Lon: -3.17395, Lat: 55.977879},
		{Lon: -3.17417, Lat: 55.977459},
		{Lon: -3.17431, Lat: 55.977189},
	}
	p := EncodePolyline(check)
	assert.Equal(t, expect, string(p))
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
		r = LineRelativePositions(lc, pp)
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
		r = LineRelativePositionsFallback(pp)
	}
	_ = r
}

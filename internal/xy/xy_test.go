package xy

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

func testApproxEqual(t *testing.T, result float64, expect float64) {
	if math.Abs(result-expect) > 1e-6 {
		t.Errorf("got %f, expect %f", result, expect)
	}
}

var testDistancePoints = []struct {
	orig              Point
	dest              Point
	Distance2d        float64
	distanceHaversine float64
}{
	{Point{-122.2772554, 37.8039604}, Point{-122.274464, 37.802963}, 0.0029642403276459884, 269.15621622898107},
	{Point{-122.2767695, 37.7770346}, Point{-122.2768192, 37.7748926}, 0.0021425765073847646, 238.21988351245543},
	{Point{-122.2226131, 37.7839461}, Point{-122.2220745, 37.7853226}, 0.00147812117568097, 160.2113624659609},
	{Point{-122.2173998, 37.7970237}, Point{-122.2163427, 37.8000987}, 0.0032516281168069854, 354.31523902465915},
}

var testLines = []struct {
	Geojson         string
	Length2d        float64
	lengthHaversine float64
}{
	{`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"type":"LineString","coordinates":[[-122.50717163085938,37.77722770873696],[-122.45155334472656,37.78156937014928],[-122.39593505859376,37.790794553924414],[-122.32040405273438,37.80544394934271],[-122.26959228515624,37.80761398306056],[-122.26478576660156,37.84124135065978],[-122.22015380859374,37.851543444173984],[-122.19955444335938,37.86618078529668],[-122.17208862304686,37.89219554724437],[-122.07595825195312,37.899239630600185],[-122.05467224121094,37.938782346134424],[-122.03681945800783,38.005902055387054],[-121.97158813476561,38.023754217706944],[-121.8816375732422,38.01726302540855],[-121.81915283203126,37.99832709721297],[-121.75048828124999,37.98858671553364]]}}]}`, 0.886044, 82069.771981},
}

var testPositions = []struct {
	Geojson           string
	Positions         []float64
	FallbackPositions []float64
}{
	{`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"type":"LineString","coordinates":[[-122.2665023803711,37.87431138542283],[-122.26581573486328,37.853712122567565],[-122.26444244384766,37.83961457275219],[-122.26821899414061,37.82551432799189],[-122.26341247558594,37.819548028632376],[-122.27130889892578,37.803273851858656],[-122.26959228515624,37.80001858607365],[-122.24555969238281,37.788352705583755],[-122.22564697265625,37.77641361883315],[-122.19577789306639,37.75225820732335],[-122.16487884521483,37.72673718477409]]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[-122.27062225341797,37.8724143256462]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[-122.26289749145506,37.84354589127591]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[-122.26427078247069,37.81507298760665]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[-122.27027893066405,37.80544394934271]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[-122.2258186340332,37.77695634643178]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[-122.19697952270508,37.75347973770911]}},{"type":"Feature","properties":{},"geometry":{"type":"Point","coordinates":[-122.16487884521483,37.72782336496339]}}]}`, []float64{0.008487181237797688, 0.1496538990811318, 0.2964636787237469, 0.3509258016789892, 0.6191751524042424, 0.7983912644738984, 0.9966620810270027}, []float64{0, 0.14880713711146062, 0.2907521492606507, 0.3472678477073867, 0.6102041614615478, 0.7953741712658748, 1.0}},
}

func decodeGeojson(data string) (*geom.LineString, []*geom.Point, error) {
	fc := geojson.FeatureCollection{}
	err := fc.UnmarshalJSON([]byte(data))
	if err != nil {
		return nil, nil, err
	}
	var line *geom.LineString
	points := []*geom.Point{}
	for _, g := range fc.Features {
		if v, ok := g.Geometry.(*geom.LineString); ok {
			line = v
		}
		if v, ok := g.Geometry.(*geom.Point); ok {
			points = append(points, v)
		}
	}
	return line, points, nil
}

func unflattenCoordinates(coords []float64) []Point {
	ret := []Point{}
	for i := 0; i < len(coords); i += 2 {
		ret = append(ret, Point{coords[i], coords[i+1]})
	}
	return ret
}

func TestDistance2d(t *testing.T) {
	for _, dp := range testDistancePoints {
		d := Distance2d(dp.orig, dp.dest)
		testApproxEqual(t, dp.Distance2d, d)
	}
}

func TestDistanceHaversine(t *testing.T) {
	for _, dp := range testDistancePoints {
		d := DistanceHaversine(dp.orig.Lon, dp.orig.Lat, dp.dest.Lon, dp.dest.Lat)
		testApproxEqual(t, dp.distanceHaversine, d)
	}
}

func TestLinePositions(t *testing.T) {
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
		pos := LinePositions(lc, pp)
		if len(pos) != len(dp.Positions) {
			t.Errorf("expect %d positions, got %d", len(dp.Positions), len(pos))
			continue
		}
		for i := 0; i < len(pos); i++ {
			testApproxEqual(t, pos[i], dp.Positions[i])
		}
	}
}

func TestLinePositionsFallback(t *testing.T) {
	for _, dp := range testPositions {
		_, points, err := decodeGeojson(dp.Geojson)
		if err != nil {
			t.Fatal(err)
		}
		pp := []Point{}
		for _, p := range points {
			pp = append(pp, Point{p.FlatCoords()[0], p.FlatCoords()[1]})
		}
		pos := LinePositionsFallback(pp)
		if len(pos) != len(dp.FallbackPositions) {
			t.Errorf("expect %d positions, got %d", len(dp.FallbackPositions), len(pos))
			continue
		}
		for i := 0; i < len(pos); i++ {
			testApproxEqual(t, pos[i], dp.FallbackPositions[i])
		}
	}
}

func TestLength2d(t *testing.T) {
	for _, line := range testLines {
		l, _, err := decodeGeojson(line.Geojson)
		if err != nil {
			t.Fatal(err)
		}
		coords := unflattenCoordinates(l.FlatCoords())
		d := Length2d(coords)
		testApproxEqual(t, line.Length2d, d)
	}
}

func TestLengthHaversine(t *testing.T) {
	for _, line := range testLines {
		l, _, err := decodeGeojson(line.Geojson)
		if err != nil {
			t.Fatal(err)
		}
		coords := unflattenCoordinates(l.FlatCoords())
		d := LengthHaversine(coords)
		testApproxEqual(t, line.lengthHaversine, d)
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
			if PointSliceContains(tc.a, tc.b) != tc.expect {
				t.Errorf("expected %t", tc.expect)
			}
		})
	}
}

func TestLineSliceShapeDistTraveled(t *testing.T) {
	simpleLine := []Point{{0, 0}, {1, 0}, {2, 0}, {3, 0}}
	simpleDist := []float64{0, 1, 2, 3}
	diagLine := []Point{{0, 0}, {2, 1}, {4, 2}, {6, 3}}
	diagDist := []float64{0, 2.23606797749979, 4.47213595499958, 6.708203932499369}
	testcases := []struct {
		name   string
		line   []Point
		dists  []float64
		a      float64
		b      float64
		expect []Point
	}{
		{
			name:   "basic",
			line:   simpleLine,
			dists:  simpleDist,
			a:      0.5,
			b:      2.5,
			expect: []Point{{0.5, 0}, {1, 0}, {2, 0}, {2.5, 0}},
		},
		{
			name:   "oob 1",
			line:   simpleLine,
			dists:  simpleDist,
			a:      -1.0,
			b:      -0.5,
			expect: nil,
		},
		{
			name:   "oob 2",
			line:   simpleLine,
			dists:  simpleDist,
			a:      5.0,
			b:      6.0,
			expect: nil,
		},
		{
			name:   "oob 3",
			line:   simpleLine,
			dists:  simpleDist,
			a:      -0.5,
			b:      2.0,
			expect: nil,
		},
		{
			name:   "oob 4",
			line:   simpleLine,
			dists:  simpleDist,
			a:      0.5,
			b:      3.5,
			expect: nil,
		},
		{
			name:   "eq 1",
			line:   simpleLine,
			dists:  simpleDist,
			a:      0.0,
			b:      2.5,
			expect: []Point{{0, 0}, {1, 0}, {2, 0}, {2.5, 0}},
		},
		{
			name:   "eq 0",
			line:   simpleLine,
			dists:  simpleDist,
			a:      0.0,
			b:      3.0,
			expect: []Point{{0, 0}, {1, 0}, {2, 0}, {3.0, 0}},
		},
		{
			name:   "eq 2",
			line:   simpleLine,
			dists:  simpleDist,
			a:      0.5,
			b:      3.0,
			expect: []Point{{0.5, 0}, {1, 0}, {2, 0}, {3.0, 0}},
		},
		{
			name:   "diag 1",
			line:   diagLine,
			dists:  diagDist,
			a:      1,
			b:      4,
			expect: []Point{{0.8944271909999159, 0.4472135954999579}, {Lon: 2, Lat: 1}, {3.5777087639996634, 1.7888543819998317}},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ret := LineSliceShapeDistTraveled(tc.line, tc.dists, tc.a, tc.b)
			// assert.Equal(t, tc.expect, ret)
			if len(ret) != len(tc.expect) {
				t.Error("expected len", len(tc.expect), "got len", len(ret))
			} else {
				for i := 0; i < len(tc.expect); i++ {
					assert.InDelta(t, 0, ret[i].Lon-tc.expect[i].Lon, 0.001, "expected to be within 0.001: %f - %f", ret[i].Lon, tc.expect[i].Lon)
					assert.InDelta(t, 0, ret[i].Lat-tc.expect[i].Lat, 0.001, "expected to be within 0.001: %f - %f", ret[i].Lat, tc.expect[i].Lat)
				}
			}
		})
	}
}

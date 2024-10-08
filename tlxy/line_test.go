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
		lc := unflattenCoordinates(line.Stride(), line.FlatCoords())
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

func TestLineSimilarity(t *testing.T) {
	// TODO
	type testcase struct {
		name string
		a    []Point
		b    []Point
	}
	testcases := []testcase{
		{
			name: "basic",
			a:    []Point{{0, 1}, {0, 2}},
			b:    []Point{{0.1, 1.1}, {0.1, 2.1}},
		},
	}
	grandLine, _, _ := decodeGeojsonToLine(`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"coordinates":[[-122.26246114293795,37.81090280922756],[-122.26259163497477,37.810481842020764],[-122.26273300134832,37.8100265074422],[-122.26284174471239,37.809648491696365],[-122.2629613624132,37.80928765667909],[-122.26308098011364,37.808969776750956],[-122.26328759250578,37.808686261065205],[-122.26354857657977,37.80842851858853],[-122.26384218366286,37.80828246411912],[-122.26426628278314,37.80817936667252],[-122.26470125624016,37.80813640936084],[-122.26522322438814,37.80826528122144],[-122.26571256952701,37.80838556142176],[-122.2661366686473,37.80848865858053],[-122.26650639608572,37.80849725000387],[-122.26709807343619,37.80862354963777],[-122.26768017239762,37.80882552453851],[-122.2681834614729,37.809025151400334]],"type":"LineString"}}]}`)
	grandLineCheck0, _, _ := decodeGeojsonToLine(`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"coordinates":[[-122.26246114293795,37.81090280922756],[-122.26256769221949,37.81044873935899],[-122.26276891548123,37.81000286253837],[-122.26285371609002,37.80961065965698],[-122.2629613624132,37.80930657278323],[-122.26308098011364,37.808969776750956],[-122.26319178368219,37.80881123600107],[-122.26333757498934,37.80865763134196],[-122.26357534083209,37.80840249335971],[-122.26384218366286,37.80828246411912],[-122.26406864474349,37.8082195293292],[-122.26426628278314,37.80817936667252],[-122.26472802049275,37.80813640936084],[-122.26523763590863,37.80828480017972],[-122.26572286347022,37.808383934844386],[-122.2661612820695,37.80848865858053],[-122.26650639608572,37.80849725000387],[-122.26680838811649,37.808540953654],[-122.26709807343619,37.80862354963777],[-122.26738141412697,37.80873275105169],[-122.26768017239762,37.80882552453851],[-122.2681793412269,37.80900236475469]],"type":"LineString"}}]}`)
	grandLineCheck1, _, _ := decodeGeojsonToLine(`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"coordinates":[[-122.26240861442449,37.81075679440447],[-122.26272470126467,37.81031410545066],[-122.2627294904592,37.809803307205755],[-122.26287795549025,37.80923574945669],[-122.26307910166142,37.808811970158885],[-122.26355323192172,37.80852818801935],[-122.26408962171155,37.80816494528973],[-122.26480800089409,37.80808926949648],[-122.2658999372515,37.80836548576781],[-122.2663884350954,37.80854332309423],[-122.26696313844145,37.808717376233574],[-122.26738458756193,37.80879683513906],[-122.2680694423828,37.80893305020638],[-122.26859625378322,37.80903899508513],[-122.26865851331243,37.80933034271747]],"type":"LineString"}}]}`)
	grandLineCheck2, _, _ := decodeGeojsonToLine(`{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"coordinates":[[-122.26299045647606,37.81084857568382],[-122.26338924709547,37.80994987412993],[-122.26396455159525,37.8090821519056],[-122.26459869405592,37.80878257877056],[-122.2654878010107,37.8088393944584],[-122.2663311451073,37.809149297442104],[-122.26705681328392,37.80920611284779],[-122.26810935901688,37.80949018922179],[-122.2688594115069,37.809690375064875],[-122.26955482996993,37.81012076705166],[-122.27019900185734,37.81024777058492]],"type":"LineString"}}]}`)
	testcases = append(testcases, testcase{
		name: "grand",
		a:    grandLineCheck0,
		b:    grandLine,
	})
	testcases = append(testcases, testcase{
		name: "grand",
		a:    grandLineCheck1,
		b:    grandLine,
	})
	testcases = append(testcases, testcase{
		name: "grand",
		a:    grandLineCheck2,
		b:    grandLine,
	})

	for _, tc := range testcases {
		LineSimilarity(tc.a, tc.b)
	}
}

func BenchmarkLinePositions(b *testing.B) {
	line, points, err := decodeGeojson(testPositions[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	lc := unflattenCoordinates(line.Stride(), line.FlatCoords())
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

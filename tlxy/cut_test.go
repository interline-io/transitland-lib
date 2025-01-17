package tlxy

import (
	"fmt"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// cutBetweenPositions is similar to CutBetweenPoints but takes absolute positions.
func cutBetweenPositionsDebug(t *testing.T, line []Point, dists []float64, startDist float64, endDist float64, extraPts ...Point) []Point {
	spt, ept, sidx, eidx, ok := cutBetweenPositions(line, dists, startDist, endDist)
	if !ok {
		return nil
	}
	var ret []Point
	ret = append(ret, spt)
	ret = append(ret, line[sidx:eidx]...)
	ret = append(ret, ept)

	i := sidx
	j := eidx

	// DEBUG - Trace log a geojson feature with visualization of result
	var fs []*geojson.Feature
	var baseLine []float64
	for _, pt := range ret {
		baseLine = append(baseLine, pt.Lon, pt.Lat)
	}
	var rawLine []float64
	for _, pt := range line {
		rawLine = append(rawLine, pt.Lon, pt.Lat)
	}
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "input line", "stroke": "#ff00ff", "stroke-width": 1, "stroke-opacity": 0.5},
		Geometry:   geom.NewLineStringFlat(geom.XY, rawLine),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "return line", "stroke": "#aaaaaa", "stroke-width": 20, "stroke-opacity": 0.5},
		Geometry:   geom.NewLineStringFlat(geom.XY, baseLine),
	})
	for _, extraPt := range extraPts {
		fs = append(fs, &geojson.Feature{
			Properties: map[string]any{"name": "extraPt", "marker-color": "#999999"},
			Geometry:   geom.NewPointFlat(geom.XY, []float64{extraPt.Lon, extraPt.Lat}),
		})
	}
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "start matched segment", "stroke": "#00ffff", "stroke-width": 5, "stroke-opacity": 0.2},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			line[i-1].Lon, line[i-1].Lat,
			line[i].Lon, line[i].Lat,
		}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "end matched segment", "stroke": "#ff00ff", "stroke-width": 5, "stroke-opacity": 0.2},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			line[j-1].Lon, line[j-1].Lat,
			line[j].Lon, line[j].Lat,
		}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "start point", "marker-color": "#00ff00"},
		Geometry:   geom.NewPointFlat(geom.XY, []float64{spt.Lon, spt.Lat}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "start point to line[i]", "stroke": "#00ff00"},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			spt.Lon, spt.Lat,
			line[i].Lon, line[i].Lat,
		}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "ept", "marker-color": "#ff0000"},
		Geometry:   geom.NewPointFlat(geom.XY, []float64{ept.Lon, ept.Lat}),
	})
	fs = append(fs, &geojson.Feature{
		Properties: map[string]any{"name": "end point to line[j-1]", "stroke": "#ff0000"},
		Geometry: geom.NewLineStringFlat(geom.XY, []float64{
			line[j-1].Lon, line[j-1].Lat,
			ept.Lon, ept.Lat,
		}),
	})
	fc := geojson.FeatureCollection{Features: fs}
	d, _ := fc.MarshalJSON()
	t.Logf("LineBetweenPositions: %s", string(d))
	return ret
}

func TestSegmentClosestPoint(t *testing.T) {
	tcs := []struct {
		p      Point
		expect Point
		a1     Point
		a2     Point
	}{
		{Point{0, 0}, Point{0, 0}, Point{0, 0}, Point{1, 1}},
		{Point{0.5, 0.5}, Point{0, 0.5}, Point{0, 0}, Point{0, 1}},
		{Point{0.5, 0.5}, Point{0.5, 0.5}, Point{0, 0}, Point{1, 1}},
		{Point{20, 20}, Point{20, 20}, Point{10, 10}, Point{30, 30}},
		{Point{20, 20}, Point{0, 20}, Point{0, 0}, Point{0, 30}},
		{Point{-20, -20}, Point{-20, -10}, Point{-100, -10}, Point{-1, -10}},
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

func testCutPositionsDebug(t *testing.T, tc lineTestCase) {
	var extraPoints []Point
	tcDist := 0.0
	if tc.from.Lon != 0.0 && tc.to.Lon != 0.0 {
		extraPoints = append(extraPoints, tc.from)
		extraPoints = append(extraPoints, tc.to)
		tcDist = DistanceHaversine(tc.from, tc.to)
	}
	ret := cutBetweenPositionsDebug(t, tc.line, tc.dists, tc.a, tc.b, extraPoints...)
	if tc.debugOnly {
		return
	}
	if len(tc.expect) > 0 && len(ret) != len(tc.expect) {
		t.Error("expected len", len(tc.expect), "got len", len(ret))
		return
	}
	for i := 0; i < len(tc.expect); i++ {
		assert.InDelta(t, 0, ret[i].Lon-tc.expect[i].Lon, 0.001, "expected to be within 0.001: %f - %f", ret[i].Lon, tc.expect[i].Lon)
		assert.InDelta(t, 0, ret[i].Lat-tc.expect[i].Lat, 0.001, "expected to be within 0.001: %f - %f", ret[i].Lat, tc.expect[i].Lat)
	}
	if tcDist > 0 {
		retLength := LengthHaversine(ret)
		assert.LessOrEqual(t, retLength, tcDist*3, "expected shape length %f to be less than 3 times the stop to stop dist %f", retLength, tcDist)
	}
}

func testCutDecode(drawShapeText string) ([]Point, []Point, []float64, []float64) {
	drawLine, drawPoints, _ := decodeGeojsonToLine(drawShapeText)
	drawLineDists := make([]float64, len(drawLine))
	for i := 1; i < len(drawLine); i++ {
		drawLineDists[i] = drawLineDists[i-1] + DistanceHaversine(drawLine[i-1], drawLine[i])
	}
	drawLineLength := drawLineDists[len(drawLineDists)-1]
	drawPositions := LineRelativePositions(drawLine, drawPoints)
	for i := 0; i < len(drawPositions); i++ {
		drawPositions[i] = drawPositions[i] * drawLineLength
	}
	return drawLine, drawPoints, drawLineDists, drawPositions
}

type lineTestCase struct {
	name      string
	line      []Point
	dists     []float64
	from      Point
	to        Point
	a         float64
	b         float64
	expect    []Point
	debugOnly bool
}

func TestCutBetweenPositions_Simple(t *testing.T) {
	// Simple examples
	simpleLine := []Point{{0, 0}, {1, 0}, {2, 0}, {3, 0}}
	simpleDist := []float64{0, 1, 2, 3}
	diagLine := []Point{{0, 0}, {2, 1}, {4, 2}, {6, 3}}
	diagDist := []float64{0, 2.23606797749979, 4.47213595499958, 6.708203932499369}

	testcases := []lineTestCase{
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
			testCutPositionsDebug(t, tc)
		})
	}
}

func TestCutBetweenPositions_Complex(t *testing.T) {
	// A slightly more complex example
	drawShapeText := `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"coordinates":[[-122.26163702039102,37.8127865537068],[-122.26179579336255,37.81246400961611],[-122.26203017251086,37.81197421774691],[-122.26215870301151,37.81165764322462],[-122.26230235474746,37.81133509420221],[-122.2624233246307,37.81098267791967],[-122.26256697636666,37.81061831355312],[-122.26268794624964,37.81026589384862],[-122.26278623427967,37.809925418977244],[-122.2629147647803,37.8095311829545],[-122.26307353775184,37.80920265132744],[-122.26318694701718,37.80893385163593],[-122.26335328060634,37.80863518416545],[-122.26355741728378,37.80845000972671],[-122.26382203890299,37.80831262193969],[-122.26422275164049,37.80818718069344],[-122.26460078252475,37.80813939349625],[-122.26500905588016,37.80818718069344],[-122.26540220799976,37.808288728385534],[-122.26571975394279,37.80837235579121],[-122.26615070915089,37.80847987660296],[-122.2665514218884,37.80848584997675],[-122.26698993771441,37.808599343987055],[-122.26740577168721,37.8087486779462],[-122.26770819639495,37.80884425152185],[-122.26806354542626,37.80897566498531],[-122.26818451530924,37.80901747831179]],"type":"LineString"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26197724818701,37.81252971315594],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26232503660067,37.810964758401994],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26301305281034,37.80952520966528],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26348937172463,37.80836638240825],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26469150993711,37.8082409412533],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26590120876725,37.80836040902477],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26717895315656,37.80876659800094],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26822231839778,37.80889801160333],"type":"Point"}}]}`
	drawLine, drawPoints, drawLineDists, drawPositions := testCutDecode(drawShapeText)
	var testcases []lineTestCase
	for i := 1; i < len(drawPoints); i++ {
		testcases = append(testcases, lineTestCase{
			name:  "",
			line:  drawLine,
			dists: drawLineDists,
			from:  drawPoints[i-1],
			to:    drawPoints[i],
			a:     drawPositions[i-1],
			b:     drawPositions[i],
		})
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testCutPositionsDebug(t, tc)
		})
	}
}

func TestCutBetweenPositions_Loop(t *testing.T) {
	// TODO: replace with a real loop example
	drawShapeText := `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"coordinates":[[-122.2639336869992,37.81548994289855],[-122.26345780859032,37.81533045351995],[-122.2632559207807,37.8150684345068],[-122.26308287408636,37.81477223798103],[-122.26290982739229,37.81440768677987],[-122.26267909846688,37.81408870300197],[-122.26244836954143,37.813746933139214],[-122.26220322005798,37.81348490850672],[-122.26197249113255,37.81316592074181],[-122.26171292109134,37.81293807149483],[-122.2614821921659,37.81272161405886],[-122.2612370426827,37.812562118699745],[-122.26086210817876,37.812186163989566],[-122.26068906148468,37.811889955900924],[-122.2606169586953,37.81155956855328],[-122.26057369702173,37.81111525220287],[-122.26047275311703,37.810830432623916],[-122.26065210677301,37.81066494290765],[-122.26099905034185,37.810642241199545],[-122.26141008937677,37.81071650448423],[-122.26195807057479,37.810807647009455],[-122.26229721936576,37.810898537542755],[-122.2623185845207,37.81102411005739],[-122.26223206117379,37.811183608738915],[-122.26208785559528,37.8115367831646],[-122.26192922945899,37.81181020736575],[-122.26178523642588,37.81212903307083],[-122.26162639774442,37.81241401555771],[-122.2615110332817,37.81266465146969],[-122.2613668277032,37.81290389404725],[-122.26113609877777,37.81315452829621],[-122.26090536985234,37.8133937692856],[-122.26074674371604,37.81363300950052],[-122.26055927646395,37.8138608566028],[-122.26040065032765,37.81407731069875],[-122.26018434196001,37.814270979615856],[-122.25995361303458,37.81452160922393]],"type":"LineString"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26406355420818,37.81564654845421],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26273948282167,37.814295217184],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26203092570142,37.813317041459655],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26180189713733,37.81293255265584],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26135099715178,37.812695073276885],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26074979717097,37.812123988785004],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26064959717412,37.81073301203713],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26188062570614,37.81084044613468],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.26183768285057,37.81186388943418],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.2611434400155,37.813203956725616],"type":"Point"}},{"type":"Feature","properties":{},"geometry":{"coordinates":[-122.2605279257495,37.81377503286322],"type":"Point"}}]}`
	drawLine, drawPoints, drawLineDists, drawPositions := testCutDecode(drawShapeText)
	var testcases []lineTestCase
	for i := 1; i < len(drawPoints); i++ {
		testcases = append(testcases, lineTestCase{
			name:  "",
			line:  drawLine,
			dists: drawLineDists,
			from:  drawPoints[i-1],
			to:    drawPoints[i],
			a:     drawPositions[i-1],
			b:     drawPositions[i],
		})
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testCutPositionsDebug(t, tc)
		})
	}
}

func TestCutBetweenPositions_IgnoreDists(t *testing.T) {
	// Ignore included shape_dist_traveled values
	data, err := os.ReadFile(testpath.RelPath("testdata/tlxy/ac.geojson"))
	if err != nil {
		t.Fatal(err)
	}
	drawShapeText := string(data)
	drawLine, drawPoints, drawLineDists, drawPositions := testCutDecode(drawShapeText)
	var testcases []lineTestCase
	for i := 1; i < len(drawPoints); i++ {
		testcases = append(testcases, lineTestCase{
			name:  "",
			line:  drawLine,
			dists: drawLineDists,
			from:  drawPoints[i-1],
			to:    drawPoints[i],
			a:     drawPositions[i-1],
			b:     drawPositions[i],
		})
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testCutPositionsDebug(t, tc)
		})
	}
}

func TestCutBetweenPositions_RealShape(t *testing.T) {
	// In reality, actual shapes and trips do NOT have reliable shape_dist_traveled values.
	// You can enable trace logging to see this example from AC Transit.
	testcases := []lineTestCase{}
	// AC Transit test shape and stops
	acData, err := os.ReadFile(testpath.RelPath("testdata/tlxy/ac.geojson"))
	if err != nil {
		t.Fatal(err)
	}
	acPositions := []float64{0, 254.67, 406.49, 520.41, 813.52, 2845.34, 3251.69, 3462.98, 3683.35, 3868.08, 4072.65, 4332.02, 4630.75, 4981.81, 5383.27, 5872.14, 5976.15, 6253.08, 6529.77, 6942.44, 7068.47, 7245.02, 7383.74, 7782.98, 7908.37, 8303.25, 8584.35, 8964.04, 9327.79, 9485.07, 9576.65, 9867.66, 10416.48, 10592.08, 11492.2, 11997.84, 12318.98, 12515.77, 12703.26, 12844.1, 13118.72, 13304.31, 13495.01, 13714.05, 13945.41, 14093.58, 14826.86}

	// Load
	dcLine, dcPoints, err := decodeGeojson(string(acData))
	if err != nil {
		t.Fatal(err)
	}
	var lcPoints []Point
	var lcDists []float64
	for _, pt := range dcLine.Coords() {
		lcPoints = append(lcPoints, Point{Lon: pt[0], Lat: pt[1]})
		lcDists = append(lcDists, pt[2])
	}
	var stopPoints []Point
	for _, dcPoint := range dcPoints {
		stopPoints = append(stopPoints, Point{Lon: dcPoint.Coords()[0], Lat: dcPoint.Coords()[1]})
	}

	// Create tests
	positions := LineRelativePositions(lcPoints, stopPoints)
	lcLength := lcDists[len(lcDists)-1]
	for i := 1; i < len(acPositions); i++ {
		testcases = append(testcases, lineTestCase{
			name:      fmt.Sprintf("testPosition-%d", i),
			line:      lcPoints,
			dists:     lcDists,
			from:      stopPoints[i-1],
			to:        stopPoints[i],
			a:         positions[i-1] * lcLength,
			b:         positions[i] * lcLength,
			debugOnly: true,
		})
	}

	// Run for each stop location
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			testCutPositionsDebug(t, tc)
		})
	}
}

func BenchmarkSegmentClosestPoint(b *testing.B) {
	l, _, err := decodeGeojson(testLines[0].Geojson)
	if err != nil {
		b.Fatal(err)
	}
	q := unflattenCoordinates(l.Stride(), l.FlatCoords())
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
	q := unflattenCoordinates(l.Stride(), l.FlatCoords())
	// get the midpoint of the two middle points
	sa := q[len(q)/2]
	sb := q[len(q)/2+1]
	p := Point{sa.Lon + (sb.Lon-sa.Lon)/2, sa.Lat + (sb.Lat-sa.Lat)/2}
	for n := 0; n < b.N; n++ {
		LineClosestPoint(q, p)
	}
}

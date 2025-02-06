package tlxy

import (
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

func TestPointInPolygon(t *testing.T) {
	tests := []struct {
		name    string
		polygon [][]geom.Coord
		point   geom.Coord
		wantHit bool
	}{
		// Basic cases
		{
			"point inside simple polygon",
			[][]geom.Coord{{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}},
			geom.Coord{0.5, 0.5},
			true,
		},
		{
			"point outside simple polygon",
			[][]geom.Coord{{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}},
			geom.Coord{1.5, 1.5},
			false,
		},

		// Edge cases
		{
			"point on vertex",
			[][]geom.Coord{{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}},
			geom.Coord{0, 0},
			true,
		},
		{
			"point on edge",
			[][]geom.Coord{{{0, 0}, {0, 1}, {1, 1}, {1, 0}, {0, 0}}},
			geom.Coord{0, 0.5},
			true,
		},

		// Holes
		{
			"point in hole",
			[][]geom.Coord{
				{{0, 0}, {0, 2}, {2, 2}, {2, 0}, {0, 0}},
				{{0.5, 0.5}, {0.5, 1.5}, {1.5, 1.5}, {1.5, 0.5}, {0.5, 0.5}},
			},
			geom.Coord{1.0, 1.0},
			false,
		},
		{
			"point on hole boundary",
			[][]geom.Coord{
				{{0, 0}, {0, 2}, {2, 2}, {2, 0}, {0, 0}},
				{{0.5, 0.5}, {0.5, 1.5}, {1.5, 1.5}, {1.5, 0.5}, {0.5, 0.5}},
			},
			geom.Coord{0.5, 1.0},
			false,
		},

		// Complex cases
		{
			"point between multiple holes",
			[][]geom.Coord{
				{{0, 0}, {0, 3}, {3, 3}, {3, 0}, {0, 0}},
				{{0.5, 0.5}, {0.5, 1}, {1, 1}, {1, 0.5}, {0.5, 0.5}},
				{{2, 2}, {2, 2.5}, {2.5, 2.5}, {2.5, 2}, {2, 2}},
			},
			geom.Coord{1.5, 1.5},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poly := geom.NewPolygon(geom.XY).MustSetCoords(tt.polygon)
			point := geom.NewPoint(geom.XY).MustSetCoords(tt.point)
			if got := pointInPolygon(poly, point); got != tt.wantHit {
				t.Errorf("pointInPolygon() = %v, want %v", got, tt.wantHit)
			}
		})
	}
}

func TestPolygonIndex_Errors(t *testing.T) {
	tests := []struct {
		name    string
		fc      geojson.FeatureCollection
		wantErr bool
	}{
		{
			"empty feature collection",
			geojson.FeatureCollection{Features: []*geojson.Feature{}},
			false,
		},
		{
			"unsupported geometry type",
			geojson.FeatureCollection{Features: []*geojson.Feature{{
				ID:       "test",
				Geometry: geom.NewPoint(geom.XY),
			}}},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewPolygonIndex(tt.fc)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPolygonIndex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPolygonIndex_EmptyQueries(t *testing.T) {
	idx, _ := NewPolygonIndex(geojson.FeatureCollection{})

	if feat, found := idx.NearestFeature(Point{0, 0}); feat != nil || found > 0 {
		t.Errorf("FeatureAt() on empty index = %v, %v; want nil, false", feat, found)
	}
}

// Test with a simple GeoJSON feature collection containing two polygons
func TestPolygonIndex_SanFrancisco(t *testing.T) {
	// A very minimal GeoJSON feature for San Francisco :laugh:
	testFeatures := `{"type":"FeatureCollection","features":[{"type":"Feature","id":"san_francisco","properties":{},"geometry":{"type":"MultiPolygon","coordinates":[[[[-122.51791,37.708131],[-122.5048,37.708131],[-122.4038,37.708131],[-122.3774,37.708131],[-122.3774,37.816239],[-122.51791,37.816239],[-122.51791,37.708131]]],[[[-122.375,37.815],[-122.365,37.815],[-122.365,37.832],[-122.375,37.832],[-122.375,37.815]]]]}},{"type":"Feature","id":"berkeley","properties":{},"geometry":{"coordinates":[[[-122.31846628706155,37.895115355095655],[-122.31846628706155,37.845986688374026],[-122.22559180139987,37.845986688374026],[-122.22559180139987,37.895115355095655],[-122.31846628706155,37.895115355095655]]],"type":"Polygon"}}]}`
	fc := geojson.FeatureCollection{}
	if err := fc.UnmarshalJSON([]byte(testFeatures)); err != nil {
		t.Fatal(err)
	}

	// Create PolygonIndex with SF feature
	idx, err := NewPolygonIndex(fc)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name           string
		point          Point
		withinFeature  string
		nearestFeature string
	}{
		{
			name:           "Mission District",
			point:          Point{-122.4194, 37.7601},
			withinFeature:  "san_francisco",
			nearestFeature: "san_francisco",
		},
		{
			name:           "Financial District",
			point:          Point{-122.4001, 37.7890},
			withinFeature:  "san_francisco",
			nearestFeature: "san_francisco",
		},
		{
			name:           "Golden Gate Park",
			point:          Point{-122.4862, 37.7694},
			withinFeature:  "san_francisco",
			nearestFeature: "san_francisco",
		},
		{
			name:           "Berkeley",
			point:          Point{-122.2729, 37.8715},
			withinFeature:  "berkeley",
			nearestFeature: "berkeley",
		},
		{
			name:           "Pacific Ocean",
			point:          Point{-122.6000, 37.7500},
			withinFeature:  "",
			nearestFeature: "san_francisco",
		},
		{
			name:           "San Jose",
			point:          Point{-121.8863, 37.3382},
			withinFeature:  "",
			nearestFeature: "",
		},
		{
			name:           "Treasure Island",
			point:          Point{-122.3704, 37.8235},
			withinFeature:  "san_francisco",
			nearestFeature: "san_francisco",
		},
		{
			name:           "Alcatraz",
			point:          Point{-122.4229, 37.8267},
			withinFeature:  "",
			nearestFeature: "san_francisco",
		},
		{
			name:           "Walnut Creek",
			point:          Point{-122.0545550, 37.90998},
			withinFeature:  "",
			nearestFeature: "berkeley",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withinFeat, _ := idx.WithinFeature(tt.point)
			nearestFeat, _ := idx.NearestFeature(tt.point)
			assert.Equal(t, tt.withinFeature, checkFeatId(withinFeat), "WithinFeature")
			assert.Equal(t, tt.nearestFeature, checkFeatId(nearestFeat), "NearestFeature")
		})
	}
}

// Test with polyline encoded features loaded from a file
func TestPolygonIndex_Timezones(t *testing.T) {
	type testCase struct {
		name           string
		point          Point
		withinFeature  string
		nearestFeature string
	}

	tcs := []testCase{
		{
			name:           "new york",
			withinFeature:  "America/New_York",
			nearestFeature: "America/New_York",
			point:          Point{Lon: -74.132285, Lat: 40.625665},
		},
		{
			name:           "california",
			withinFeature:  "America/Los_Angeles",
			nearestFeature: "America/Los_Angeles",
			point:          Point{Lon: -122.431297, Lat: 37.773972},
		},
		{
			// Point is in the Pacific Ocean, closest to California
			name:           "just off california coast",
			withinFeature:  "",
			nearestFeature: "America/Los_Angeles",
			point:          Point{Lon: -122.8121, Lat: 37.5116},
		},
		{
			name:           "utah",
			withinFeature:  "America/Denver",
			nearestFeature: "America/Denver",
			point:          Point{Lon: -109.056664, Lat: 40.996479},
		},
		{
			name:           "colorado",
			withinFeature:  "America/Denver",
			nearestFeature: "America/Denver",
			point:          Point{Lon: -109.045685, Lat: 40.997833},
		},
		{
			name:           "wyoming",
			withinFeature:  "America/Denver",
			nearestFeature: "America/Denver",
			point:          Point{Lon: -109.050133, Lat: 41.002209},
		},
		{
			name:           "north dakota",
			withinFeature:  "America/Chicago",
			nearestFeature: "America/Chicago",
			point:          Point{Lon: -100.964531, Lat: 45.946934},
		},
		{
			name:           "georgia",
			withinFeature:  "America/New_York",
			nearestFeature: "America/New_York",
			point:          Point{Lon: -82.066697, Lat: 30.370054},
		},
		{
			name:           "florida",
			withinFeature:  "America/New_York",
			nearestFeature: "America/New_York",
			point:          Point{Lon: -82.046522, Lat: 30.360419},
		},
		{
			name:           "saskatchewan",
			withinFeature:  "America/Mexico_City",
			nearestFeature: "America/Mexico_City",
			point:          Point{Lon: -102.007904, Lat: 58.269615},
		},
		{
			name:           "manitoba",
			withinFeature:  "America/Chicago",
			nearestFeature: "America/Chicago",
			point:          Point{Lon: -101.982025, Lat: 58.269245},
		},
		{
			name:           "texas",
			withinFeature:  "America/Chicago",
			nearestFeature: "America/Chicago",
			point:          Point{Lon: -94.794261, Lat: 29.289210},
		},
		{
			name:           "texas water 1",
			withinFeature:  "America/Chicago",
			nearestFeature: "America/Chicago",
			point:          Point{Lon: -94.784667, Lat: 29.286234},
		},
		{
			name:  "texas water 2",
			point: Point{Lon: -94.237, Lat: 26.874},
		},
		{
			name:           "texas water 3",
			withinFeature:  "",
			nearestFeature: "America/Chicago",
			point:          Point{Lon: -95.10091, Lat: 28.75702},
		},
		{
			name:           "canada maidstone 1",
			withinFeature:  "America/Denver",
			nearestFeature: "America/Denver",
			point:          Point{Lon: -108.96735, Lat: 53.01851},
		},
		{
			name:           "canada maidstone 2",
			withinFeature:  "America/Mexico_City",
			nearestFeature: "America/Mexico_City",
			point:          Point{Lon: -108.86594, Lat: 52.99610},
		},
		{
			name:           "canada halifax",
			withinFeature:  "America/Halifax",
			nearestFeature: "America/Halifax",
			point:          Point{Lon: -68.90401, Lat: 47.26115},
		},
		{
			name:           "phoenix exclave",
			withinFeature:  "America/Phoenix",
			nearestFeature: "America/Phoenix",
			point:          Point{Lon: -110.7767, Lat: 35.6494},
		},
		{
			name:           "phoenix exclave inclave",
			withinFeature:  "America/Denver",
			nearestFeature: "America/Denver",
			point:          Point{Lon: -110.1514, Lat: 35.7432},
		},
	}
	fn := testpath.RelPath("testdata/tlxy/tz-example.polyline")
	r, err := os.Open(fn)
	if err != nil {
		t.Fatal(err)
	}
	fc, err := PolylinesToGeojson(r)
	if err != nil {
		t.Fatal(err)
	}
	tzWorld, err := NewPolygonIndex(fc)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			withinFeature, _ := tzWorld.WithinFeature(tc.point)
			nearestFeature, _ := tzWorld.NearestFeature(tc.point)
			assert.Equal(t, tc.withinFeature, checkFeatId(withinFeature), "WithinFeature")
			assert.Equal(t, tc.nearestFeature, checkFeatId(nearestFeature), "NearestFeature")

		})
	}
}
func checkFeatId(g *geojson.Feature) string {
	if g == nil {
		return ""
	}
	return g.ID
}

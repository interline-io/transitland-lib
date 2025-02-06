package tlxy

import (
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
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

	if feat, found := idx.FeatureAt(Point{0, 0}); found || feat != nil {
		t.Errorf("FeatureAt() on empty index = %v, %v; want nil, false", feat, found)
	}

	if name, found := idx.FeatureNameAt(Point{0, 0}); found || name != "" {
		t.Errorf("FeatureNameAt() on empty index = %v, %v; want \"\", false", name, found)
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
		name        string
		point       Point
		wantFeature string
		wantFound   bool
	}{
		{"Mission District", Point{-122.4194, 37.7601}, "san_francisco", true},
		{"Financial District", Point{-122.4001, 37.7890}, "san_francisco", true},
		{"Golden Gate Park", Point{-122.4862, 37.7694}, "san_francisco", true},
		{"Berkeley", Point{-122.2729, 37.8715}, "berkeley", true},
		{"Pacific Ocean", Point{-122.6000, 37.7500}, "", false},
		{"San Jose", Point{-121.8863, 37.3382}, "", false},
		{"Treasure Island", Point{-122.3704, 37.8235}, "san_francisco", true},
		{"Alcatraz", Point{-122.4229, 37.8267}, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feat, found := idx.FeatureAt(tt.point)
			if found != tt.wantFound {
				t.Errorf("FeatureAt(%v) found = %v, want %v", tt.point, found, tt.wantFound)
			}
			if tt.wantFound {
				if feat == nil {
					t.Errorf("FeatureAt(%v) feature = nil, want non-nil", tt.point)
				} else if feat.ID != tt.wantFeature {
					t.Errorf("FeatureAt(%v) feature ID = %v, want %v", tt.point, feat.ID, tt.wantFeature)
				}
			}
		})
	}
}

// Test with polyline encoded features loaded from a file
func TestPolygonIndex_Timezones(t *testing.T) {
	type testCase struct {
		name          string
		point         Point
		expectName    string
		expectMissing bool
	}

	tcs := []testCase{
		{
			name:       "new york",
			expectName: "America/New_York",
			point:      Point{Lon: -74.132285, Lat: 40.625665},
		},
		{
			name:       "california",
			expectName: "America/Los_Angeles",
			point:      Point{Lon: -122.431297, Lat: 37.773972},
		},
		{
			name:       "utah",
			expectName: "America/Denver",
			point:      Point{Lon: -109.056664, Lat: 40.996479},
		},
		{
			name:       "colorado",
			expectName: "America/Denver",
			point:      Point{Lon: -109.045685, Lat: 40.997833},
		},
		{
			name:       "wyoming",
			expectName: "America/Denver",
			point:      Point{Lon: -109.050133, Lat: 41.002209},
		},
		{
			name:       "north dakota",
			expectName: "America/Chicago",
			point:      Point{Lon: -100.964531, Lat: 45.946934},
		},
		{
			name:       "georgia",
			expectName: "America/New_York",
			point:      Point{Lon: -82.066697, Lat: 30.370054},
		},
		{
			name:       "florida",
			expectName: "America/New_York",
			point:      Point{Lon: -82.046522, Lat: 30.360419},
		},
		{
			name:       "saskatchewan",
			expectName: "America/Mexico_City",
			point:      Point{Lon: -102.007904, Lat: 58.269615},
		},
		{
			name:       "manitoba",
			expectName: "America/Chicago",
			point:      Point{Lon: -101.982025, Lat: 58.269245},
		},
		{
			name:       "texas",
			expectName: "America/Chicago",
			point:      Point{Lon: -94.794261, Lat: 29.289210},
		},
		{
			name:       "texas water 1",
			expectName: "America/Chicago",
			point:      Point{Lon: -94.784667, Lat: 29.286234},
		},
		{
			name:          "texas water 2",
			expectMissing: true,
			point:         Point{Lon: -94.237, Lat: 26.874},
		},
		{
			name:       "canada maidstone 1",
			expectName: "America/Denver",
			point:      Point{Lon: -108.96735, Lat: 53.01851},
		},
		{
			name:       "canada maidstone 2",
			expectName: "America/Mexico_City",
			point:      Point{Lon: -108.86594, Lat: 52.99610},
		},
		{
			name:       "canada halifax",
			expectName: "America/Halifax",
			point:      Point{Lon: -68.90401, Lat: 47.26115},
		},
		{
			name:       "phoenix exclave",
			expectName: "America/Phoenix",
			point:      Point{Lon: -110.7767, Lat: 35.6494},
		},
		{
			name:       "phoenix exclave inclave",
			expectName: "America/Denver",
			point:      Point{Lon: -110.1514, Lat: 35.7432},
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
			feat, ok := tzWorld.FeatureAt(tc.point)
			switch {
			case tc.expectMissing:
				if ok {
					t.Errorf("expected missing, got %v", ok)
				}
			case feat == nil || !ok:
				t.Errorf("expected feature, got nil")
			case feat.ID != tc.expectName:
				t.Errorf("expected %s, got %s", tc.expectName, feat.ID)
			case feat.ID == tc.expectName:
				// ok
			default:
				t.Errorf("unexpected test case")
			}
		})
	}
}

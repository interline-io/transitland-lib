package tlxy

import (
	"testing"

	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

func TestSanFranciscoPoints(t *testing.T) {
	// A very minimal GeoJSON feature for San Francisco :laugh:
	sfFeature := &geojson.Feature{
		ID: "san_francisco",
		Geometry: geom.NewMultiPolygon(geom.XY).MustSetCoords([][][]geom.Coord{
			// Main SF peninsula
			{{
				{-122.517910, 37.708131},
				{-122.504800, 37.708131},
				{-122.403800, 37.708131},
				{-122.377400, 37.708131},
				{-122.377400, 37.816239},
				{-122.517910, 37.816239},
				{-122.517910, 37.708131},
			}},
			// Treasure Island
			{{
				{-122.3750, 37.8150},
				{-122.3750, 37.8320},
				{-122.3650, 37.8320},
				{-122.3650, 37.8150},
				{-122.3750, 37.8150},
			}},
		}),
	}
	// Create Berkeley MultiPolygon feature
	berkeleyFeature := &geojson.Feature{
		ID: "berkeley",
		Geometry: geom.NewMultiPolygon(geom.XY).MustSetCoords([][][]geom.Coord{
			{{
				{-122.324439, 37.853842},
				{-122.324439, 37.900420},
				{-122.229052, 37.900420},
				{-122.229052, 37.853842},
				{-122.324439, 37.853842},
			}},
		}),
	}

	// Create PolygonIndex with SF feature
	idx, err := NewPolygonIndex(geojson.FeatureCollection{
		Features: []*geojson.Feature{sfFeature, berkeleyFeature},
	})
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

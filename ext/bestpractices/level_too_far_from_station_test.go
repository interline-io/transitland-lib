package bestpractices

import (
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	geom "github.com/twpayne/go-geom"
)

func testLevelPolygon(lon, lat float64) *geom.Polygon {
	// Create a small polygon centered at the given coordinates
	offset := 0.001 // ~100m
	polygon, _ := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{
		{
			{lon - offset, lat - offset},
			{lon + offset, lat - offset},
			{lon + offset, lat + offset},
			{lon - offset, lat + offset},
			{lon - offset, lat - offset},
		},
	})
	polygon.SetSRID(4326)
	return polygon
}

// mockGeomCache implements tlxy.GeomCache for testing
type mockGeomCache struct {
	stops map[string]tlxy.Point
}

func newMockGeomCache() *mockGeomCache {
	return &mockGeomCache{stops: make(map[string]tlxy.Point)}
}

func (m *mockGeomCache) GetStop(id string) tlxy.Point {
	return m.stops[id]
}

func (m *mockGeomCache) GetShape(id string) []tlxy.Point {
	return nil
}

func (m *mockGeomCache) AddStop(id string, pt tlxy.Point) {
	m.stops[id] = pt
}

func TestLevelTooFarFromStationCheck(t *testing.T) {
	emap := tt.NewEntityMap()

	// Station coordinates (San Francisco)
	stationLon := -122.4194
	stationLat := 37.7749

	tests := []struct {
		name         string
		station      *gtfs.Stop
		platformStop *gtfs.Stop
		level        *gtfs.Level
		expectError  bool
	}{
		{
			name: "Valid: level geometry near station",
			station: &gtfs.Stop{
				StopID:       tt.NewString("station1"),
				StopName:     tt.NewString("Main Station"),
				LocationType: tt.NewInt(1), // Station
				Geometry:     tt.NewPoint(stationLon, stationLat),
			},
			platformStop: &gtfs.Stop{
				StopID:        tt.NewString("platform1"),
				StopName:      tt.NewString("Platform 1"),
				LocationType:  tt.NewInt(0), // Stop/Platform
				ParentStation: tt.NewKey("station1"),
				LevelID:       tt.NewKey("ground_floor"),
				Geometry:      tt.NewPoint(stationLon+0.0001, stationLat),
			},
			level: &gtfs.Level{
				LevelID:    tt.NewString("ground_floor"),
				LevelIndex: tt.NewFloat(0),
				LevelName:  tt.NewString("Ground Floor"),
				// Level geometry centered near the station (~50m away)
				Geometry: tt.NewGeometry(testLevelPolygon(stationLon+0.0005, stationLat)),
			},
			expectError: false,
		},
		{
			name: "Invalid: level geometry too far from station",
			station: &gtfs.Stop{
				StopID:       tt.NewString("station2"),
				StopName:     tt.NewString("Central Station"),
				LocationType: tt.NewInt(1), // Station
				Geometry:     tt.NewPoint(stationLon, stationLat),
			},
			platformStop: &gtfs.Stop{
				StopID:        tt.NewString("platform2"),
				StopName:      tt.NewString("Platform 2"),
				LocationType:  tt.NewInt(0), // Stop/Platform
				ParentStation: tt.NewKey("station2"),
				LevelID:       tt.NewKey("far_level"),
				Geometry:      tt.NewPoint(stationLon+0.0001, stationLat),
			},
			level: &gtfs.Level{
				LevelID:    tt.NewString("far_level"),
				LevelIndex: tt.NewFloat(0),
				LevelName:  tt.NewString("Far Level"),
				// Level geometry centered ~1km away from station
				Geometry: tt.NewGeometry(testLevelPolygon(stationLon+0.01, stationLat)),
			},
			expectError: true,
		},
		{
			name: "Valid: level without geometry (no check needed)",
			station: &gtfs.Stop{
				StopID:       tt.NewString("station3"),
				StopName:     tt.NewString("Simple Station"),
				LocationType: tt.NewInt(1),
				Geometry:     tt.NewPoint(stationLon, stationLat),
			},
			platformStop: &gtfs.Stop{
				StopID:        tt.NewString("platform3"),
				StopName:      tt.NewString("Platform 3"),
				LocationType:  tt.NewInt(0),
				ParentStation: tt.NewKey("station3"),
				LevelID:       tt.NewKey("no_geom_level"),
				Geometry:      tt.NewPoint(stationLon+0.0001, stationLat),
			},
			level: &gtfs.Level{
				LevelID:    tt.NewString("no_geom_level"),
				LevelIndex: tt.NewFloat(0),
				LevelName:  tt.NewString("Simple Level"),
				// No geometry
			},
			expectError: false,
		},
		{
			name: "Valid: level with no stops referencing it",
			station: &gtfs.Stop{
				StopID:       tt.NewString("station4"),
				StopName:     tt.NewString("Another Station"),
				LocationType: tt.NewInt(1),
				Geometry:     tt.NewPoint(stationLon, stationLat),
			},
			platformStop: nil, // No platform stop references this level
			level: &gtfs.Level{
				LevelID:    tt.NewString("orphan_level"),
				LevelIndex: tt.NewFloat(0),
				LevelName:  tt.NewString("Orphan Level"),
				Geometry:   tt.NewGeometry(testLevelPolygon(stationLon+0.01, stationLat)), // Far away, but no stops reference it
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock geom cache and add station coordinates
			cache := newMockGeomCache()
			coords := tc.station.Coordinates()
			cache.AddStop(tc.station.StopID.Val, tlxy.Point{Lon: coords[0], Lat: coords[1]})

			check := &LevelTooFarFromStationCheck{}
			check.SetGeomCache(cache)

			// Filter the station stop (doesn't cache coordinates anymore, just for consistency)
			check.Filter(tc.station, emap)

			// Filter the platform stop (if present) - this caches the level->station mapping
			if tc.platformStop != nil {
				check.Filter(tc.platformStop, emap)
			}

			// Validate the level
			errs := check.Validate(tc.level)

			if tc.expectError {
				assert.NotEmpty(t, errs, "Expected validation error")
				if len(errs) > 0 {
					_, ok := errs[0].(*LevelTooFarFromStationError)
					assert.True(t, ok, "Expected LevelTooFarFromStationError")
				}
			} else {
				assert.Empty(t, errs, "Expected no validation errors")
			}
		})
	}
}

func TestPolygonCentroid(t *testing.T) {
	tests := []struct {
		name      string
		geometry  tt.Geometry
		expectLon float64
		expectLat float64
		expectOK  bool
	}{
		{
			name:      "Simple polygon",
			geometry:  tt.NewGeometry(testLevelPolygon(-122.4, 37.7)),
			expectLon: -122.4,
			expectLat: 37.7,
			expectOK:  true,
		},
		{
			name:     "Invalid geometry",
			geometry: tt.Geometry{},
			expectOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			centroid, ok := polygonCentroid(tc.geometry)
			assert.Equal(t, tc.expectOK, ok)
			if tc.expectOK {
				// Allow small floating point differences
				assert.InDelta(t, tc.expectLon, centroid.Lon, 0.001)
				assert.InDelta(t, tc.expectLat, centroid.Lat, 0.001)
			}
		})
	}
}

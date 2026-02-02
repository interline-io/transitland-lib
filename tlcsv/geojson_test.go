package tlcsv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	geom "github.com/twpayne/go-geom"
)

// Helper to create a test polygon geometry
func testPolygon(t *testing.T) *geom.Polygon {
	polygon, err := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{
		{{-122.4194, 37.7749}, {-122.4094, 37.7749}, {-122.4094, 37.7649}, {-122.4194, 37.7649}, {-122.4194, 37.7749}},
	})
	require.NoError(t, err)
	polygon.SetSRID(4326)
	return polygon
}

// Helper to create a test multipolygon geometry
func testMultiPolygon(t *testing.T) *geom.MultiPolygon {
	multipolygon, err := geom.NewMultiPolygon(geom.XY).SetCoords([][][]geom.Coord{
		{{{-122.5, 37.8}, {-122.4, 37.8}, {-122.4, 37.7}, {-122.5, 37.7}, {-122.5, 37.8}}},
		{{{-122.3, 37.9}, {-122.2, 37.9}, {-122.2, 37.8}, {-122.3, 37.8}, {-122.3, 37.9}}},
	})
	require.NoError(t, err)
	multipolygon.SetSRID(4326)
	return multipolygon
}

// Helper to write test files and create a reader
func setupTestReader(t *testing.T, files map[string]string) *Reader {
	tmpDir := t.TempDir()
	for name, content := range files {
		require.NoError(t, os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644))
	}
	reader, err := NewReader(tmpDir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	t.Cleanup(func() { reader.Close() })
	return reader
}

//
// locations.geojson tests
//

func TestReader_Locations_GeoJSON(t *testing.T) {
	geojsonData := `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "zone_downtown",
      "properties": {
        "stop_name": "Downtown Flex Zone",
        "stop_desc": "On-demand service area",
        "zone_id": "zone1",
        "stop_url": "https://example.com/flex"
      },
      "geometry": {
        "type": "Polygon",
        "coordinates": [[[-122.4194, 37.7749], [-122.4094, 37.7749], [-122.4094, 37.7649], [-122.4194, 37.7649], [-122.4194, 37.7749]]]
      }
    },
    {
      "type": "Feature",
      "id": "zone_midtown",
      "properties": {"stop_name": "Midtown Flex Zone"},
      "geometry": {
        "type": "MultiPolygon",
        "coordinates": [[[[-122.5, 37.8], [-122.4, 37.8], [-122.4, 37.7], [-122.5, 37.7], [-122.5, 37.8]]]]
      }
    }
  ]
}`
	reader := setupTestReader(t, map[string]string{
		"agency.txt":         "agency_id,agency_name,agency_url,agency_timezone\n1,Demo,http://example.com,America/Los_Angeles",
		"locations.geojson":  geojsonData,
	})

	locMap := make(map[string]*gtfs.Location)
	for loc := range reader.Locations() {
		l := loc
		locMap[loc.LocationID.Val] = &l
	}

	require.Len(t, locMap, 2)

	// Check downtown zone
	downtown := locMap["zone_downtown"]
	require.NotNil(t, downtown)
	assert.Equal(t, "Downtown Flex Zone", downtown.StopName.Val)
	assert.Equal(t, "On-demand service area", downtown.StopDesc.Val)
	assert.Equal(t, "zone1", downtown.ZoneID.Val)
	assert.True(t, downtown.Geometry.Valid)

	// Check midtown zone
	midtown := locMap["zone_midtown"]
	require.NotNil(t, midtown)
	assert.Equal(t, "Midtown Flex Zone", midtown.StopName.Val)
	assert.True(t, midtown.Geometry.Valid)
}

func TestReader_Locations_NoGeoJSON(t *testing.T) {
	reader := setupTestReader(t, map[string]string{
		"agency.txt": "agency_id,agency_name,agency_url,agency_timezone\n1,Demo,http://example.com,America/Los_Angeles",
	})

	count := 0
	for range reader.Locations() {
		count++
	}
	assert.Equal(t, 0, count, "Should return no locations when file doesn't exist")
}

func TestWriter_Locations_GeoJSON(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewWriter(tmpDir)
	require.NoError(t, err)
	require.NoError(t, writer.Create())

	loc := &gtfs.Location{
		LocationID: tt.NewString("zone_test"),
		StopName:   tt.NewString("Test Zone"),
		Geometry:   tt.NewGeometry(testPolygon(t)),
	}

	_, err = writer.AddEntity(loc)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Read back and verify
	reader, err := NewReader(tmpDir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	locations := reader.Locations()
	count := 0
	for loc := range locations {
		count++
		assert.Equal(t, "zone_test", loc.LocationID.Val)
		assert.True(t, loc.Geometry.Valid)
	}
	assert.Equal(t, 1, count)
}

//
// levels.geojson tests
//

func TestReader_Levels_WithGeoJSON(t *testing.T) {
	levelsCSV := `level_id,level_index,level_name
ground_floor,0,Ground Floor
basement,-1,Basement
mezzanine,0.5,Mezzanine`

	levelsGeoJSON := `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "ground_floor",
      "properties": {"level_id": "ground_floor", "level_index": 0, "level_name": "Ground Floor"},
      "geometry": {
        "type": "Polygon",
        "coordinates": [[[-122.4194, 37.7749], [-122.4094, 37.7749], [-122.4094, 37.7649], [-122.4194, 37.7649], [-122.4194, 37.7749]]]
      }
    },
    {
      "type": "Feature",
      "id": "basement",
      "properties": {"level_id": "basement"},
      "geometry": {
        "type": "MultiPolygon",
        "coordinates": [[[[-122.5, 37.8], [-122.4, 37.8], [-122.4, 37.7], [-122.5, 37.7], [-122.5, 37.8]]]]
      }
    }
  ]
}`

	reader := setupTestReader(t, map[string]string{
		"agency.txt":      "agency_id,agency_name,agency_url,agency_timezone\n1,Demo,http://example.com,America/Los_Angeles",
		"levels.txt":      levelsCSV,
		"levels.geojson":  levelsGeoJSON,
	})

	levelMap := make(map[string]*gtfs.Level)
	for level := range reader.Levels() {
		l := level
		levelMap[level.LevelID.Val] = &l
	}

	require.Len(t, levelMap, 3)

	// ground_floor - has geometry from geojson
	groundFloor := levelMap["ground_floor"]
	require.NotNil(t, groundFloor)
	assert.Equal(t, "Ground Floor", groundFloor.LevelName.Val)
	assert.Equal(t, float64(0), groundFloor.LevelIndex.Val)
	assert.True(t, groundFloor.Geometry.Valid, "ground_floor should have geometry")

	// basement - has geometry from geojson
	basement := levelMap["basement"]
	require.NotNil(t, basement)
	assert.Equal(t, float64(-1), basement.LevelIndex.Val)
	assert.True(t, basement.Geometry.Valid, "basement should have geometry")

	// mezzanine - NOT in geojson, should have no geometry
	mezzanine := levelMap["mezzanine"]
	require.NotNil(t, mezzanine)
	assert.Equal(t, "Mezzanine", mezzanine.LevelName.Val)
	assert.False(t, mezzanine.Geometry.Valid, "mezzanine should not have geometry")
}

func TestReader_Levels_NoGeoJSON(t *testing.T) {
	reader := setupTestReader(t, map[string]string{
		"agency.txt": "agency_id,agency_name,agency_url,agency_timezone\n1,Demo,http://example.com,America/Los_Angeles",
		"levels.txt": "level_id,level_index,level_name\nL1,0,Level 1\nL2,1,Level 2",
	})

	count := 0
	for level := range reader.Levels() {
		count++
		assert.False(t, level.Geometry.Valid, "Level should not have geometry")
	}
	assert.Equal(t, 2, count)
}

func TestWriter_Levels_WithGeoJSON(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewWriter(tmpDir)
	require.NoError(t, err)
	require.NoError(t, writer.Create())

	// Level with polygon geometry
	level1 := &gtfs.Level{
		LevelID:    tt.NewString("ground_floor"),
		LevelIndex: tt.NewFloat(0),
		LevelName:  tt.NewString("Ground Floor"),
		Geometry:   tt.NewGeometry(testPolygon(t)),
	}

	// Level with multipolygon geometry
	level2 := &gtfs.Level{
		LevelID:    tt.NewString("basement"),
		LevelIndex: tt.NewFloat(-1),
		LevelName:  tt.NewString("Basement"),
		Geometry:   tt.NewGeometry(testMultiPolygon(t)),
	}

	// Level without geometry
	level3 := &gtfs.Level{
		LevelID:    tt.NewString("mezzanine"),
		LevelIndex: tt.NewFloat(0.5),
		LevelName:  tt.NewString("Mezzanine"),
	}

	_, err = writer.AddEntity(level1)
	require.NoError(t, err)
	_, err = writer.AddEntity(level2)
	require.NoError(t, err)
	_, err = writer.AddEntity(level3)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Read back and verify
	reader, err := NewReader(tmpDir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	levelMap := make(map[string]*gtfs.Level)
	for level := range reader.Levels() {
		l := level
		levelMap[level.LevelID.Val] = &l
	}

	require.Len(t, levelMap, 3)

	// Verify round-trip preserves geometry
	assert.True(t, levelMap["ground_floor"].Geometry.Valid)
	assert.True(t, levelMap["basement"].Geometry.Valid)
	assert.False(t, levelMap["mezzanine"].Geometry.Valid)
}

func TestWriter_Levels_GeoJSON_Zip(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.zip")

	writer, err := NewWriter(tmpFile)
	require.NoError(t, err)
	require.NoError(t, writer.Create())

	level := &gtfs.Level{
		LevelID:    tt.NewString("platform"),
		LevelIndex: tt.NewFloat(1),
		LevelName:  tt.NewString("Platform Level"),
		Geometry:   tt.NewGeometry(testPolygon(t)),
	}

	_, err = writer.AddEntity(level)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Read back from zip
	reader, err := NewReader(tmpFile)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	count := 0
	for level := range reader.Levels() {
		count++
		assert.Equal(t, "platform", level.LevelID.Val)
		assert.True(t, level.Geometry.Valid)
	}
	assert.Equal(t, 1, count)
}

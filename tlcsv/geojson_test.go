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

func TestReader_Locations_GeoJSON(t *testing.T) {
	// Create a temporary GTFS feed with locations.geojson
	tmpDir := t.TempDir()

	// Create minimal required GTFS files
	agencyCSV := `agency_id,agency_name,agency_url,agency_timezone
1,Demo Transit,http://example.com,America/Los_Angeles`

	stopsCSV := `stop_id,stop_name,stop_lat,stop_lon
stop1,Stop 1,37.7749,-122.4194`

	routesCSV := `route_id,route_short_name,route_long_name,route_type,agency_id
route1,1,Main Street,3,1`

	// Create locations.geojson with GTFS-Flex zones
	geojsonData := `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "zone_downtown",
      "properties": {
        "stop_name": "Downtown Flex Zone",
        "stop_desc": "On-demand service area in downtown",
        "zone_id": "zone1",
        "stop_url": "https://example.com/flex/downtown"
      },
      "geometry": {
        "type": "Polygon",
        "coordinates": [[
          [-122.4194, 37.7749],
          [-122.4094, 37.7749],
          [-122.4094, 37.7649],
          [-122.4194, 37.7649],
          [-122.4194, 37.7749]
        ]]
      }
    },
    {
      "type": "Feature",
      "id": "zone_midtown",
      "properties": {
        "stop_name": "Midtown Flex Zone"
      },
      "geometry": {
        "type": "MultiPolygon",
        "coordinates": [
          [[
            [-122.5, 37.8],
            [-122.4, 37.8],
            [-122.4, 37.7],
            [-122.5, 37.7],
            [-122.5, 37.8]
          ]],
          [[
            [-122.3, 37.9],
            [-122.2, 37.9],
            [-122.2, 37.8],
            [-122.3, 37.8],
            [-122.3, 37.9]
          ]]
        ]
      }
    }
  ]
}`

	// Write files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agency.txt"), []byte(agencyCSV), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "stops.txt"), []byte(stopsCSV), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "routes.txt"), []byte(routesCSV), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "locations.geojson"), []byte(geojsonData), 0644))

	// Create reader
	reader, err := NewReader(tmpDir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	// Read locations from GeoJSON
	locations := reader.Locations()

	locMap := make(map[string]bool)
	count := 0
	for loc := range locations {
		count++
		locMap[loc.LocationID.Val] = true

		if loc.LocationID.Val == "zone_downtown" {
			assert.Equal(t, "Downtown Flex Zone", loc.StopName.Val)
			assert.Equal(t, "On-demand service area in downtown", loc.StopDesc.Val)
			assert.Equal(t, "zone1", loc.ZoneID.Val)
			assert.Equal(t, "https://example.com/flex/downtown", loc.StopURL.Val)
			assert.True(t, loc.Geometry.Valid, "Geometry should be valid")
		}

		if loc.LocationID.Val == "zone_midtown" {
			assert.Equal(t, "Midtown Flex Zone", loc.StopName.Val)
			assert.True(t, loc.Geometry.Valid, "MultiPolygon geometry should be valid")
		}
	}

	assert.Equal(t, 2, count)
	assert.True(t, locMap["zone_downtown"])
	assert.True(t, locMap["zone_midtown"])
}

func TestReader_Locations_NoGeoJSON(t *testing.T) {
	// Create a feed without locations.geojson
	tmpDir := t.TempDir()

	agencyCSV := `agency_id,agency_name,agency_url,agency_timezone
1,Demo Transit,http://example.com,America/Los_Angeles`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agency.txt"), []byte(agencyCSV), 0644))

	reader, err := NewReader(tmpDir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	// Should return empty channel without error
	locations := reader.Locations()

	count := 0
	for range locations {
		count++
	}

	assert.Equal(t, 0, count, "Should return no locations when file doesn't exist")
}

func TestReader_Locations_AllFormats(t *testing.T) {
	// Verify reader can handle regular stops AND GeoJSON locations together
	tmpDir := t.TempDir()

	agencyCSV := `agency_id,agency_name,agency_url,agency_timezone
1,Demo Transit,http://example.com,America/Los_Angeles`

	stopsCSV := `stop_id,stop_name,stop_lat,stop_lon
stop1,Regular Stop,37.7749,-122.4194`

	routesCSV := `route_id,route_short_name,route_long_name,route_type,agency_id
route1,1,Route 1,3,1`

	geojsonData := `{
  "type": "FeatureCollection",
  "features": [
    {
      "type": "Feature",
      "id": "flex1",
      "properties": {
        "stop_name": "Flex Zone"
      },
      "geometry": {
        "type": "Polygon",
        "coordinates": [[
          [-122.4, 37.7],
          [-122.3, 37.7],
          [-122.3, 37.6],
          [-122.4, 37.6],
          [-122.4, 37.7]
        ]]
      }
    }
  ]
}`

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agency.txt"), []byte(agencyCSV), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "stops.txt"), []byte(stopsCSV), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "routes.txt"), []byte(routesCSV), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "locations.geojson"), []byte(geojsonData), 0644))

	reader, err := NewReader(tmpDir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	// Can read regular stops
	stops := reader.Stops()
	stopCount := 0
	for range stops {
		stopCount++
	}
	assert.Equal(t, 1, stopCount)

	// Can read flex locations
	locations := reader.Locations()
	locCount := 0
	for loc := range locations {
		locCount++
		assert.Equal(t, "flex1", loc.LocationID.Val)
	}
	assert.Equal(t, 1, locCount)
}

func TestWriter_Locations_GeoJSON(t *testing.T) {
	// Test writing locations.geojson
	tmpDir := t.TempDir()

	writer, err := NewWriter(tmpDir)
	require.NoError(t, err)
	require.NoError(t, writer.Create())
	defer writer.Delete()

	// Create Location entities
	loc1 := gtfs.Location{
		LocationID: tt.NewString("zone_downtown"),
		StopName:   tt.NewString("Downtown Flex Zone"),
		StopDesc:   tt.NewString("On-demand service area in downtown"),
		ZoneID:     tt.NewString("zone1"),
		StopURL:    tt.NewUrl("https://example.com/flex/downtown"),
	}

	// Create a polygon geometry
	polygon, err := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{
		{{-122.4194, 37.7749}, {-122.4094, 37.7749}, {-122.4094, 37.7649}, {-122.4194, 37.7649}, {-122.4194, 37.7749}},
	})
	require.NoError(t, err)
	polygon.SetSRID(4326)
	loc1.Geometry = tt.NewGeometry(polygon)

	loc2 := gtfs.Location{
		LocationID: tt.NewString("zone_midtown"),
		StopName:   tt.NewString("Midtown Flex Zone"),
	}

	// Create a MultiPolygon geometry
	multipolygon, err := geom.NewMultiPolygon(geom.XY).SetCoords([][][]geom.Coord{
		{
			{{-122.5, 37.8}, {-122.4, 37.8}, {-122.4, 37.7}, {-122.5, 37.7}, {-122.5, 37.8}},
		},
		{
			{{-122.3, 37.9}, {-122.2, 37.9}, {-122.2, 37.8}, {-122.3, 37.8}, {-122.3, 37.9}},
		},
	})
	require.NoError(t, err)
	multipolygon.SetSRID(4326)
	loc2.Geometry = tt.NewGeometry(multipolygon)

	// Write locations
	_, err = writer.AddEntity(&loc1)
	require.NoError(t, err)

	_, err = writer.AddEntity(&loc2)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	// Read back and verify
	reader, err := NewReader(tmpDir)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	locations := reader.Locations()
	locMap := make(map[string]*gtfs.Location)
	for loc := range locations {
		locMap[loc.LocationID.Val] = &loc
	}

	require.Len(t, locMap, 2)

	// Verify loc1
	downtown, ok := locMap["zone_downtown"]
	require.True(t, ok)
	assert.Equal(t, "Downtown Flex Zone", downtown.StopName.Val)
	assert.Equal(t, "On-demand service area in downtown", downtown.StopDesc.Val)
	assert.Equal(t, "zone1", downtown.ZoneID.Val)
	assert.Equal(t, "https://example.com/flex/downtown", downtown.StopURL.Val)
	assert.True(t, downtown.Geometry.Valid)

	// Verify loc2
	midtown, ok := locMap["zone_midtown"]
	require.True(t, ok)
	assert.Equal(t, "Midtown Flex Zone", midtown.StopName.Val)
	assert.True(t, midtown.Geometry.Valid)
}

func TestWriter_Locations_GeoJSON_Zip(t *testing.T) {
	// Test writing locations.geojson to a zip file
	tmpFile := filepath.Join(t.TempDir(), "test.zip")

	writer, err := NewWriter(tmpFile)
	require.NoError(t, err)
	require.NoError(t, writer.Create())
	defer writer.Delete()

	// Create a Location entity
	loc := gtfs.Location{
		LocationID: tt.NewString("zone_test"),
		StopName:   tt.NewString("Test Zone"),
	}

	// Create a polygon geometry
	polygon, err := geom.NewPolygon(geom.XY).SetCoords([][]geom.Coord{
		{{-122.4194, 37.7749}, {-122.4094, 37.7749}, {-122.4094, 37.7649}, {-122.4194, 37.7649}, {-122.4194, 37.7749}},
	})
	require.NoError(t, err)
	polygon.SetSRID(4326)
	loc.Geometry = tt.NewGeometry(polygon)

	// Write location
	_, err = writer.AddEntity(&loc)
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	// Read back from zip and verify
	reader, err := NewReader(tmpFile)
	require.NoError(t, err)
	require.NoError(t, reader.Open())
	defer reader.Close()

	locations := reader.Locations()
	count := 0
	for loc := range locations {
		count++
		assert.Equal(t, "zone_test", loc.LocationID.Val)
		assert.Equal(t, "Test Zone", loc.StopName.Val)
		assert.True(t, loc.Geometry.Valid)
	}

	assert.Equal(t, 1, count)
}

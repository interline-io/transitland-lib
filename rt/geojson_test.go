package rt

import (
	"encoding/json"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
)

func TestVehiclePositionsToGeoJSON(t *testing.T) {
	// Test with vehicle positions data
	msg, err := ReadFile(testpath.RelPath("testdata/rt/ct-vehicle-positions.pb"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("standard geojson", func(t *testing.T) {
		result, err := VehiclePositionsToGeoJSON(msg, false)
		if err != nil {
			t.Fatal(err)
		}

		// Parse the result to verify it's valid GeoJSON
		var geojson map[string]any
		if err := json.Unmarshal(result, &geojson); err != nil {
			t.Fatal(err)
		}

		// Verify it's a FeatureCollection
		assert.Equal(t, "FeatureCollection", geojson["type"])

		// Verify it has features
		features, ok := geojson["features"].([]any)
		assert.True(t, ok, "features should be an array")
		assert.Greater(t, len(features), 0, "should have at least one feature")

		// Verify first feature structure
		if len(features) > 0 {
			feature := features[0].(map[string]any)
			assert.Equal(t, "Feature", feature["type"])

			// Verify geometry
			geometry, ok := feature["geometry"].(map[string]any)
			assert.True(t, ok, "geometry should be present")
			assert.Equal(t, "Point", geometry["type"])

			coordinates, ok := geometry["coordinates"].([]any)
			assert.True(t, ok, "coordinates should be an array")
			assert.Equal(t, 2, len(coordinates), "coordinates should have 2 elements (lon, lat)")

			// Verify properties
			properties, ok := feature["properties"].(map[string]any)
			assert.True(t, ok, "properties should be present")
			assert.Contains(t, properties, "id", "should have id property")
		}
	})

	t.Run("geojsonl format", func(t *testing.T) {
		result, err := VehiclePositionsToGeoJSON(msg, true)
		if err != nil {
			t.Fatal(err)
		}

		// GeoJSONL should be one JSON object per line
		lines := 0
		for _, b := range result {
			if b == '\n' {
				lines++
			}
		}

		// Should have at least one line (one feature per line)
		assert.Greater(t, lines, 0, "should have at least one line")

		// Verify each line is valid JSON
		var currentLine []byte
		for _, b := range result {
			if b == '\n' {
				if len(currentLine) > 0 {
					var feature map[string]any
					if err := json.Unmarshal(currentLine, &feature); err != nil {
						t.Errorf("invalid JSON in line: %s", string(currentLine))
					}
					assert.Equal(t, "Feature", feature["type"])
				}
				currentLine = nil
			} else {
				currentLine = append(currentLine, b)
			}
		}
	})
}

func TestVehiclePositionsToGeoJSON_EmptyMessage(t *testing.T) {
	// Test with empty message
	msg, err := ReadFile(testpath.RelPath("testdata/rt/hart-vehicle-positions.pb"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := VehiclePositionsToGeoJSON(msg, false)
	if err != nil {
		t.Fatal(err)
	}

	// Should return valid GeoJSON with empty features array
	var geojson map[string]any
	if err := json.Unmarshal(result, &geojson); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "FeatureCollection", geojson["type"])
	features, ok := geojson["features"].([]any)
	assert.True(t, ok, "features should be an array")
	assert.Equal(t, 0, len(features), "should have no features for empty message")
}

func TestVehiclePositionsToGeoJSON_NoVehiclePositions(t *testing.T) {
	// Test with trip updates (no vehicle positions)
	msg, err := ReadFile(testpath.RelPath("testdata/rt/ct-trip-updates.pb"))
	if err != nil {
		t.Fatal(err)
	}

	result, err := VehiclePositionsToGeoJSON(msg, false)
	if err != nil {
		t.Fatal(err)
	}

	// Should return valid GeoJSON with empty features array
	var geojson map[string]any
	if err := json.Unmarshal(result, &geojson); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "FeatureCollection", geojson["type"])
	features, ok := geojson["features"].([]any)
	assert.True(t, ok, "features should be an array")
	assert.Equal(t, 0, len(features), "should have no features for trip updates")
}

func TestVehiclePositionsToGeoJSONLStream(t *testing.T) {
	// Test with vehicle positions data
	msg, err := ReadFile(testpath.RelPath("testdata/rt/ct-vehicle-positions.pb"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("streaming geojsonl", func(t *testing.T) {
		var buf []byte
		w := &testWriter{&buf}

		err := VehiclePositionsToGeoJSONLStream(msg, w)
		if err != nil {
			t.Fatal(err)
		}

		// Should have written some data
		assert.Greater(t, len(buf), 0, "should have written data")

		// Count lines (one feature per line)
		lines := 0
		for _, b := range buf {
			if b == '\n' {
				lines++
			}
		}
		assert.Greater(t, lines, 0, "should have at least one line")

		// Verify each line is valid JSON feature
		var currentLine []byte
		featureCount := 0
		for _, b := range buf {
			if b == '\n' {
				if len(currentLine) > 0 {
					var feature map[string]any
					if err := json.Unmarshal(currentLine, &feature); err != nil {
						t.Errorf("invalid JSON in line: %s", string(currentLine))
					}
					assert.Equal(t, "Feature", feature["type"])

					// Verify feature structure
					geometry, ok := feature["geometry"].(map[string]any)
					assert.True(t, ok, "geometry should be present")
					assert.Equal(t, "Point", geometry["type"])

					properties, ok := feature["properties"].(map[string]any)
					assert.True(t, ok, "properties should be present")
					assert.Contains(t, properties, "id", "should have id property")

					featureCount++
				}
				currentLine = nil
			} else {
				currentLine = append(currentLine, b)
			}
		}

		assert.Greater(t, featureCount, 0, "should have at least one feature")
	})

	t.Run("streaming empty message", func(t *testing.T) {
		// Test with empty message
		msg, err := ReadFile(testpath.RelPath("testdata/rt/hart-vehicle-positions.pb"))
		if err != nil {
			t.Fatal(err)
		}

		var buf []byte
		w := &testWriter{&buf}

		err = VehiclePositionsToGeoJSONLStream(msg, w)
		if err != nil {
			t.Fatal(err)
		}

		// Should have written no data for empty message
		assert.Equal(t, 0, len(buf), "should have no output for empty message")
	})

	t.Run("streaming no vehicle positions", func(t *testing.T) {
		// Test with trip updates (no vehicle positions)
		msg, err := ReadFile(testpath.RelPath("testdata/rt/ct-trip-updates.pb"))
		if err != nil {
			t.Fatal(err)
		}

		var buf []byte
		w := &testWriter{&buf}

		err = VehiclePositionsToGeoJSONLStream(msg, w)
		if err != nil {
			t.Fatal(err)
		}

		// Should have written no data for trip updates
		assert.Equal(t, 0, len(buf), "should have no output for trip updates")
	})
}

// testWriter implements io.Writer for testing
type testWriter struct {
	buf *[]byte
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

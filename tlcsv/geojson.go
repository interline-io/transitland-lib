package tlcsv

import (
	"encoding/json"
	"io"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	geom "github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/encoding/geojson"
)

// GeoJSONFeatureParser is a callback function for parsing a GeoJSON feature
// into a GTFS entity. It receives the feature and should return the parsed
// entity and whether it was successfully parsed.
type GeoJSONFeatureParser[T any] func(*geojson.Feature) (T, bool)

// GeoJSONFeatureWriter is a callback function for converting a GTFS entity
// into a GeoJSON feature. It receives the entity and should return the feature
// and whether it was successfully converted.
type GeoJSONFeatureWriter[T any] func(T) (*geojson.Feature, bool)

// readGeoJSON reads and parses a GeoJSON FeatureCollection file using a
// provided parser function. This provides a generic way to read any GeoJSON
// file format into GTFS entities.
func readGeoJSON[T any](reader *Reader, filename string, parser GeoJSONFeatureParser[T]) ([]T, error) {
	var entities []T
	var parseErr error

	err := reader.Adapter.OpenFile(filename, func(in io.Reader) {
		var fc geojson.FeatureCollection
		if err := json.NewDecoder(in).Decode(&fc); err != nil {
			parseErr = err
			return
		}

		for _, feature := range fc.Features {
			if entity, ok := parser(feature); ok {
				entities = append(entities, entity)
			}
		}
	})

	if err != nil {
		return nil, err
	}
	if parseErr != nil {
		return nil, parseErr
	}

	return entities, nil
}

// parseLocationFeature parses a GeoJSON feature into a gtfs.Location.
// This is used for locations.geojson (GTFS-Flex extension).
func parseLocationFeature(feature *geojson.Feature) (gtfs.Location, bool) {
	loc := gtfs.Location{}

	// The ID is at the feature level, not in properties
	if feature.ID != "" {
		loc.LocationID = tt.NewString(feature.ID)
	}

	// Parse properties
	if feature.Properties != nil {
		if v, ok := feature.Properties["stop_name"].(string); ok {
			loc.StopName = tt.NewString(v)
		}
		if v, ok := feature.Properties["stop_desc"].(string); ok {
			loc.StopDesc = tt.NewString(v)
		}
		if v, ok := feature.Properties["zone_id"].(string); ok {
			loc.ZoneID = tt.NewString(v)
		}
		if v, ok := feature.Properties["stop_url"].(string); ok {
			loc.StopURL = tt.NewUrl(v)
		}
	}

	// Parse geometry - must be Polygon or MultiPolygon for locations
	if feature.Geometry != nil {
		switch g := feature.Geometry.(type) {
		case *geom.Polygon:
			g.SetSRID(4326)
			loc.Geometry = tt.NewGeometry(g)
		case *geom.MultiPolygon:
			g.SetSRID(4326)
			loc.Geometry = tt.NewGeometry(g)
		default:
			// Invalid geometry type for location - skip
			return loc, false
		}
	}

	return loc, true
}

// readLocationsGeoJSON reads and parses locations.geojson from the adapter.
// This is a GTFS-Flex extension that defines zones using GeoJSON Polygon
// or MultiPolygon geometries where riders can request pickups or drop-offs.
func (reader *Reader) readLocationsGeoJSON(filename string) ([]gtfs.Location, error) {
	return readGeoJSON(reader, filename, parseLocationFeature)
}

// writeLocationFeature converts a gtfs.Location entity to a GeoJSON feature.
// This is used for locations.geojson (GTFS-Flex extension).
func writeLocationFeature(loc *gtfs.Location) (*geojson.Feature, bool) {
	if loc == nil {
		return nil, false
	}
	feature := &geojson.Feature{}

	// Set feature ID from LocationID
	if loc.LocationID.Val != "" {
		feature.ID = loc.LocationID.Val
	}

	// Set properties
	properties := make(map[string]any)
	if loc.StopName.Val != "" {
		properties["stop_name"] = loc.StopName.Val
	}
	if loc.StopDesc.Val != "" {
		properties["stop_desc"] = loc.StopDesc.Val
	}
	if loc.ZoneID.Val != "" {
		properties["zone_id"] = loc.ZoneID.Val
	}
	if loc.StopURL.Val != "" {
		properties["stop_url"] = loc.StopURL.Val
	}
	if len(properties) > 0 {
		feature.Properties = properties
	}

	// Set geometry - must be Polygon or MultiPolygon for locations
	if !loc.Geometry.Valid {
		return nil, false
	}

	// Ensure SRID is set
	switch g := loc.Geometry.Val.(type) {
	case *geom.Polygon:
		g.SetSRID(4326)
		feature.Geometry = g
	case *geom.MultiPolygon:
		g.SetSRID(4326)
		feature.Geometry = g
	default:
		// Invalid geometry type for location - skip
		return nil, false
	}

	return feature, true
}

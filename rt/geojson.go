package rt

import (
	"encoding/json"
	"io"

	"github.com/interline-io/transitland-lib/rt/pb"
)

// createVehicleFeature creates a GeoJSON feature from a vehicle entity
func createVehicleFeature(entity *pb.FeedEntity) map[string]any {
	vehicle := entity.Vehicle
	properties := map[string]any{
		"id": entity.Id,
	}

	// Add vehicle descriptor properties
	if vehicle.Vehicle != nil {
		if vehicle.Vehicle.Id != nil {
			properties["vehicle_id"] = *vehicle.Vehicle.Id
		}
		if vehicle.Vehicle.Label != nil {
			properties["vehicle_label"] = *vehicle.Vehicle.Label
		}
		if vehicle.Vehicle.LicensePlate != nil {
			properties["vehicle_license_plate"] = *vehicle.Vehicle.LicensePlate
		}
	}

	// Add trip information
	if vehicle.Trip != nil {
		if vehicle.Trip.TripId != nil {
			properties["trip_id"] = *vehicle.Trip.TripId
		}
		if vehicle.Trip.RouteId != nil {
			properties["route_id"] = *vehicle.Trip.RouteId
		}
		if vehicle.Trip.DirectionId != nil {
			properties["direction_id"] = *vehicle.Trip.DirectionId
		}
	}

	// Add position and timestamp
	if vehicle.Position != nil {
		if vehicle.Position.Latitude != nil {
			properties["latitude"] = *vehicle.Position.Latitude
		}
		if vehicle.Position.Longitude != nil {
			properties["longitude"] = *vehicle.Position.Longitude
		}
	}

	if vehicle.Timestamp != nil {
		properties["timestamp"] = *vehicle.Timestamp
	}

	if vehicle.CurrentStopSequence != nil {
		properties["current_stop_sequence"] = *vehicle.CurrentStopSequence
	}

	if vehicle.StopId != nil {
		properties["stop_id"] = *vehicle.StopId
	}

	if vehicle.CurrentStatus != nil {
		properties["current_status"] = *vehicle.CurrentStatus
	}

	if vehicle.CongestionLevel != nil {
		properties["congestion_level"] = *vehicle.CongestionLevel
	}

	return map[string]any{
		"type":       "Feature",
		"properties": properties,
		"geometry": map[string]any{
			"type": "Point",
			"coordinates": []float64{
				float64(*vehicle.Position.Longitude),
				float64(*vehicle.Position.Latitude),
			},
		},
	}
}

// VehiclePositionsToGeoJSON converts vehicle position protobuf data to GeoJSON format
func VehiclePositionsToGeoJSON(rtMsg *pb.FeedMessage, isGeoJSONL bool) ([]byte, error) {
	features := []map[string]any{}

	for _, entity := range rtMsg.Entity {
		if entity.Vehicle == nil || entity.Vehicle.Position == nil {
			continue
		}

		feature := createVehicleFeature(entity)
		features = append(features, feature)
	}

	if isGeoJSONL {
		// Return GeoJSONL format (one feature per line)
		var result []byte
		for i, feature := range features {
			featureBytes, err := json.Marshal(feature)
			if err != nil {
				return nil, err
			}
			result = append(result, featureBytes...)
			if i < len(features)-1 {
				result = append(result, '\n')
			}
		}
		return result, nil
	} else {
		// Return standard GeoJSON format
		featureCollection := map[string]any{
			"type":     "FeatureCollection",
			"features": features,
		}
		return json.Marshal(featureCollection)
	}
}

// VehiclePositionsToGeoJSONLStream streams vehicle position protobuf data to GeoJSONL format
// This function writes features directly to the provided writer as they are processed,
// reducing memory usage for large datasets.
func VehiclePositionsToGeoJSONLStream(rtMsg *pb.FeedMessage, w io.Writer) error {
	encoder := json.NewEncoder(w)

	for _, entity := range rtMsg.Entity {
		if entity.Vehicle == nil || entity.Vehicle.Position == nil {
			continue
		}

		feature := createVehicleFeature(entity)

		// Encode and write the feature directly to the writer
		if err := encoder.Encode(feature); err != nil {
			return err
		}
	}

	return nil
}

package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FlexGeographyIDDuplicateError reports when a geography ID is duplicated across
// stops.txt, locations.geojson, or location_groups.txt.
type FlexGeographyIDDuplicateError struct {
	GeographyID string
	FirstFile   string
	SecondFile  string
	bc
}

func (e *FlexGeographyIDDuplicateError) Error() string {
	return fmt.Sprintf(
		"geography_id '%s' is duplicated across %s and %s (IDs must be unique across stops.stop_id, locations.geojson id, and location_groups.location_group_id)",
		e.GeographyID,
		e.FirstFile,
		e.SecondFile,
	)
}

// FlexGeographyIDUniqueCheck validates that geography IDs are unique across
// stops.txt, locations.geojson, and location_groups.txt.
//
// Per GTFS-Flex specification, these three ID spaces are merged into a single
// namespace that can be referenced in stop_times.txt via stop_id or location_id.
type FlexGeographyIDUniqueCheck struct {
	geographyIDs map[string]string // geography_id -> filename
}

func (e *FlexGeographyIDUniqueCheck) AfterWrite(eid string, ent tt.Entity, _ *tt.EntityMap) error {
	if e.geographyIDs == nil {
		e.geographyIDs = map[string]string{}
	}

	var geographyID string
	var filename string

	switch entity := ent.(type) {
	case *gtfs.Stop:
		geographyID = entity.StopID.Val
		filename = "stops.txt"
	case *gtfs.Location:
		geographyID = entity.LocationID.Val
		filename = "locations.geojson"
	case *gtfs.LocationGroup:
		geographyID = entity.LocationGroupID.Val
		filename = "location_groups.txt"
	default:
		return nil
	}

	// Store the first occurrence
	if geographyID != "" {
		if _, exists := e.geographyIDs[geographyID]; !exists {
			e.geographyIDs[geographyID] = filename
		}
	}

	return nil
}

func (e *FlexGeographyIDUniqueCheck) Validate(ent tt.Entity) []error {
	var geographyID string
	var filename string

	switch entity := ent.(type) {
	case *gtfs.Stop:
		geographyID = entity.StopID.Val
		filename = "stops.txt"
	case *gtfs.Location:
		geographyID = entity.LocationID.Val
		filename = "locations.geojson"
	case *gtfs.LocationGroup:
		geographyID = entity.LocationGroupID.Val
		filename = "location_groups.txt"
	default:
		return nil
	}

	if geographyID == "" {
		return nil
	}

	// Check if this ID was seen in a different file
	if firstFile, exists := e.geographyIDs[geographyID]; exists && firstFile != filename {
		return []error{&FlexGeographyIDDuplicateError{
			GeographyID: geographyID,
			FirstFile:   firstFile,
			SecondFile:  filename,
		}}
	}

	return nil
}

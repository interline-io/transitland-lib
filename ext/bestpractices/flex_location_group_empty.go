package bestpractices

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FlexLocationGroupEmptyError reports when a location_group has no associated stops.
type FlexLocationGroupEmptyError struct {
	LocationGroupID string
	bc
}

func (e *FlexLocationGroupEmptyError) Error() string {
	return fmt.Sprintf(
		"location_group '%s' has no associated stops in location_group_stops.txt",
		e.LocationGroupID,
	)
}

// FlexLocationGroupEmptyCheck validates that all location_groups have at least one stop.
type FlexLocationGroupEmptyCheck struct {
	locationGroupStops map[string]int // location_group_id -> count of stops
}

func (e *FlexLocationGroupEmptyCheck) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	if e.locationGroupStops == nil {
		e.locationGroupStops = map[string]int{}
	}

	// Count stops for each location_group
	if lgs, ok := ent.(*gtfs.LocationGroupStop); ok {
		e.locationGroupStops[lgs.LocationGroupID.Val]++
	}

	return nil
}

func (e *FlexLocationGroupEmptyCheck) Validate(ent tt.Entity) []error {
	lg, ok := ent.(*gtfs.LocationGroup)
	if !ok {
		return nil
	}

	// Check if this location_group has any stops
	count := e.locationGroupStops[lg.LocationGroupID.Val]
	if count == 0 {
		return []error{&FlexLocationGroupEmptyError{
			LocationGroupID: lg.LocationGroupID.Val,
		}}
	}

	return nil
}


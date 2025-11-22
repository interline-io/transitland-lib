package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FlexStopLocationTypeError reports when a stop referenced in a flex service has invalid location_type.
type FlexStopLocationTypeError struct {
	StopID       string
	LocationType int
	bc
}

func (e *FlexStopLocationTypeError) Error() string {
	return fmt.Sprintf(
		"stop '%s' referenced in flex service has location_type %d, but flex services can only use stops with location_type 0 (stop/platform)",
		e.StopID,
		e.LocationType,
	)
}

// FlexStopLocationTypeCheck validates that stops referenced in flex stop_times have appropriate location_type.
// According to GTFS-Flex, stops used in continuous stopping or location_id/location_group_id references
// must be location_type=0 (stop/platform).
type FlexStopLocationTypeCheck struct {
	locationTypes map[string]int
	flexStops     map[string]bool // stops used in flex services
}

func (e *FlexStopLocationTypeCheck) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	if e.locationTypes == nil {
		e.locationTypes = map[string]int{}
	}
	if e.flexStops == nil {
		e.flexStops = map[string]bool{}
	}

	// Track location_type for all stops
	if stop, ok := ent.(*gtfs.Stop); ok {
		e.locationTypes[eid] = stop.LocationType.Int()
	}

	// Track stops used in flex services
	if st, ok := ent.(*gtfs.StopTime); ok {
		// Check if this stop_time indicates a flex service
		isFlex := (st.ContinuousPickup.Valid && st.ContinuousPickup.Val == 2) ||
			(st.ContinuousDropOff.Valid && st.ContinuousDropOff.Val == 2) ||
			(st.PickupType.Valid && st.PickupType.Val == 2) ||
			(st.DropOffType.Valid && st.DropOffType.Val == 2) ||
			st.StartPickupDropOffWindow.Valid ||
			st.EndPickupDropOffWindow.Valid ||
			st.PickupBookingRuleID.Valid ||
			st.DropOffBookingRuleID.Valid

		if isFlex {
			e.flexStops[st.StopID.Val] = true
		}
	}

	return nil
}

func (e *FlexStopLocationTypeCheck) Validate(ent tt.Entity) []error {
	// Run validation after all entities are processed
	stop, ok := ent.(*gtfs.Stop)
	if !ok {
		return nil
	}

	// Check if this stop is used in flex services
	if !e.flexStops[stop.StopID.Val] {
		return nil
	}

	// Flex stops must be location_type = 0
	locationType := e.locationTypes[stop.StopID.Val]
	if locationType != 0 {
		return []error{&FlexStopLocationTypeError{
			StopID:       stop.StopID.Val,
			LocationType: locationType,
		}}
	}

	return nil
}

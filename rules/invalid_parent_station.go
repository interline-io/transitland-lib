package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// InvalidParentStationError reports when a parent station is not location_type = 1.
type InvalidParentStationError struct {
	bc
}

// NewInvalidParentStationError returns a new InvalidParentStationError
func NewInvalidParentStationError(value string) *InvalidParentStationError {
	return &InvalidParentStationError{bc: bc{Value: value}}
}

func (e *InvalidParentStationError) Error() string {
	return fmt.Sprintf("parent_station '%s' is missing or has invalid location_type", e.Value)
}

// ParentStationLocationTypeCheck checks if a stop's parent_station is of the allowed type.
type ParentStationLocationTypeCheck struct {
	locationTypes map[string]int
}

// Validate .
func (e *ParentStationLocationTypeCheck) Validate(ent tl.Entity) []error {
	// Confirm the parent station location_type is acceptable
	stop, ok := ent.(*tl.Stop)
	if !ok {
		return nil
	}
	if e.locationTypes == nil {
		e.locationTypes = map[string]int{}
	}
	e.locationTypes[stop.StopID] = stop.LocationType
	if stop.ParentStation.Key == "" {
		return nil
	}
	// We need to compare as strings because EntityMap is map[string]string
	var errs []error
	stype := stop.LocationType
	ptype, ok := e.locationTypes[stop.ParentStation.Key]
	if !ok {
		// parent station not found; this is checked during UpdateKeys
	} else if stype == 4 {
		// Boarding areas may only link to type = 0
		if ptype != 0 {
			errs = append(errs, NewInvalidParentStationError(stop.ParentStation.Key))
		}
	} else if ptype != 1 {
		// All other types must have station as parent
		errs = append(errs, NewInvalidParentStationError(stop.ParentStation.Key))
	}
	return errs
}

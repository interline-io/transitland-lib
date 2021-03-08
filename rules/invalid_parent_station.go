package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// InvalidParentStationError reports when a parent_station has a location_type that is not allowed.
type InvalidParentStationError struct {
	StopID            string
	LocationType      int
	ParentStation     string
	ParentStationType int
	bc
}

func (e *InvalidParentStationError) Error() string {
	return fmt.Sprintf(
		"stop '%s' with location_type %d has parent_station '%s' with location_type %d which is not allowed",
		e.StopID,
		e.LocationType,
		e.ParentStation,
		e.ParentStationType,
	)
}

// ParentStationLocationTypeCheck checks for InvalidParentStationErrors.
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
			errs = append(errs, &InvalidParentStationError{
				StopID:            stop.StopID,
				LocationType:      stop.LocationType,
				ParentStation:     stop.ParentStation.Key,
				ParentStationType: ptype,
			})
		}
	} else if ptype != 1 {
		// All other types must have station as parent
		errs = append(errs, &InvalidParentStationError{
			StopID:            stop.StopID,
			LocationType:      stop.LocationType,
			ParentStation:     stop.ParentStation.Key,
			ParentStationType: ptype,
		})
	}
	return errs
}

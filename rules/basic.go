package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// EntityErrorCheck runs the entity's built in Errors() check.
type EntityErrorCheck struct{}

// Validate .
func (e *EntityErrorCheck) Validate(ent tl.Entity) []error {
	return ent.Errors()
}

// EntityWarningCheck runs the entity's built in Warnings() check.
type EntityWarningCheck struct{}

// Validate .
func (e *EntityWarningCheck) Validate(ent tl.Entity) []error {
	return ent.Warnings()
}

///////////////////

// EntityDuplicateCheck determines if a unique entity ID is present more than once in the file.
type EntityDuplicateCheck struct {
	duplicates *tl.EntityMap
}

// Validate .
func (e *EntityDuplicateCheck) Validate(ent tl.Entity) []error {
	if e.duplicates == nil {
		e.duplicates = tl.NewEntityMap()
	}
	eid := ent.EntityID()
	if eid == "" {
		return nil
	}
	var errs []error
	efn := ent.Filename()
	if _, ok := e.duplicates.Get(efn, eid); ok {
		errs = append(errs, causes.NewDuplicateIDError(eid))
	} else {
		e.duplicates.Set(efn, eid, eid)
	}
	return errs
}

///////////////////

// ValidFarezoneCheck checks if fare_rules are referencing zone_id values seen on stops.
type ValidFarezoneCheck struct {
	zones map[string]string
}

// Validate .
func (e *ValidFarezoneCheck) Validate(ent tl.Entity) []error {
	if e.zones == nil {
		e.zones = map[string]string{}
	}
	var errs []error
	switch v := ent.(type) {
	case *tl.Stop:
		e.zones[v.ZoneID] = v.ZoneID
	case *tl.FareRule:
		// TODO: updating values should be handled in UpdateKeys
		// probably shouldn't mutate in validators...
		if fz, ok := e.zones[v.OriginID]; ok {
			v.OriginID = fz
		} else if v.OriginID != "" {
			errs = append(errs, causes.NewInvalidFarezoneError("origin_id", v.OriginID))
		}
		if fz, ok := e.zones[v.DestinationID]; ok {
			v.DestinationID = fz
		} else if v.DestinationID != "" {
			errs = append(errs, causes.NewInvalidFarezoneError("destination_id", v.DestinationID))
		}
		if fz, ok := e.zones[v.ContainsID]; ok {
			v.ContainsID = fz
		} else if v.ContainsID != "" {
			errs = append(errs, causes.NewInvalidFarezoneError("contains_id", v.ContainsID))
		}
	}
	return errs
}

///////////////////

// AgencyIDConditionallyRequiredCheck checks if agency_id is missing when more than one agency is present.
type AgencyIDConditionallyRequiredCheck struct {
	agencyCount int
}

// Validate .
func (e *AgencyIDConditionallyRequiredCheck) Validate(ent tl.Entity) []error {
	var errs []error
	switch v := ent.(type) {
	case *tl.FareAttribute:
		if e.agencyCount > 1 && v.AgencyID.Key == "" {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *tl.Route:
		if v.AgencyID != "" {
			// ok
		} else if e.agencyCount > 1 {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
		// TODO: Move to best practice warning
		// else if e.agencyCount == 1 {
		// 	warns = append(warns, causes.NewConditionallyRequiredFieldError("agency_id"))
		// }
	case *tl.Agency:
		e.agencyCount++
		if e.agencyCount > 1 && v.AgencyID == "" {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	}
	return errs
}

///////////////////

// InconsistentTimezoneCheck checks if agency_timezone doesn't match the first seen agency_timezone
type InconsistentTimezoneCheck struct {
	firstTimeZone string
}

// Validate .
func (e *InconsistentTimezoneCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tl.Agency)
	if !ok {
		return nil
	}
	if e.firstTimeZone == "" {
		e.firstTimeZone = v.AgencyTimezone
	}
	if v.AgencyTimezone != e.firstTimeZone {
		return []error{causes.NewInconsistentTimezoneError(v.AgencyTimezone)}
	}
	return nil
}

///////////////////

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
			errs = append(errs, causes.NewInvalidParentStationError(stop.ParentStation.Key))
		}
	} else if ptype != 1 {
		// All other types must have station as parent
		errs = append(errs, causes.NewInvalidParentStationError(stop.ParentStation.Key))
	}
	return errs
}

///////////////////

// StopTimeSequenceCheck checks that all sequences stop_time sequences in a trip are valid.
// This should be split into multiple validators.
type StopTimeSequenceCheck struct{}

// Validate .
func (e *StopTimeSequenceCheck) Validate(ent tl.Entity) []error {
	trip, ok := ent.(*tl.Trip)
	if !ok {
		return nil
	}
	// Use existing validator.
	var errs = tl.ValidateStopTimes(trip.StopTimes)
	return errs
}

///////////////////

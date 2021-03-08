package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// InvalidFarezoneError reports when a farezone does not exist.
type InvalidFarezoneError struct {
	bc
}

// NewInvalidFarezoneError returns a new InvalidFarezoneError
func NewInvalidFarezoneError(field string, value string) *InvalidFarezoneError {
	return &InvalidFarezoneError{bc: bc{Field: field, Value: value}}
}

func (e *InvalidFarezoneError) Error() string {
	return fmt.Sprintf("%s farezone '%s' is not present on any stops", e.Field, e.Value)
}

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
			errs = append(errs, NewInvalidFarezoneError("origin_id", v.OriginID))
		}
		if fz, ok := e.zones[v.DestinationID]; ok {
			v.DestinationID = fz
		} else if v.DestinationID != "" {
			errs = append(errs, NewInvalidFarezoneError("destination_id", v.DestinationID))
		}
		if fz, ok := e.zones[v.ContainsID]; ok {
			v.ContainsID = fz
		} else if v.ContainsID != "" {
			errs = append(errs, NewInvalidFarezoneError("contains_id", v.ContainsID))
		}
	}
	return errs
}

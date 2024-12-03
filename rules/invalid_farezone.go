package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
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

// ValidFarezoneCheck checks for InvalidFarezoneErrors.
type ValidFarezoneCheck struct {
	zones map[string]string
}

// Validate .
func (e *ValidFarezoneCheck) Validate(ent tt.Entity) []error {
	if e.zones == nil {
		e.zones = map[string]string{}
	}
	var errs []error
	switch v := ent.(type) {
	case *gtfs.Stop:
		e.zones[v.ZoneID.Val] = v.ZoneID.Val
	case *gtfs.FareRule:
		// TODO: updating values should be handled in UpdateKeys
		// probably shouldn't mutate in validators...
		if fz, ok := e.zones[v.OriginID.Val]; ok {
			v.OriginID.Set(fz)
		} else if v.OriginID.Valid {
			errs = append(errs, NewInvalidFarezoneError("origin_id", v.OriginID.Val))
		}
		if fz, ok := e.zones[v.DestinationID.Val]; ok {
			v.DestinationID.Set(fz)
		} else if v.DestinationID.Valid {
			errs = append(errs, NewInvalidFarezoneError("destination_id", v.DestinationID.Val))
		}
		if fz, ok := e.zones[v.ContainsID.Val]; ok {
			v.ContainsID.Set(fz)
		} else if v.ContainsID.Valid {
			errs = append(errs, NewInvalidFarezoneError("contains_id", v.ContainsID.Val))
		}
	}
	return errs
}

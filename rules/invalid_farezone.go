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

// ValidFarezoneCheck checks for InvalidFarezoneErrors.
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
		if fz, ok := e.zones[v.OriginID.Val]; ok {
			v.OriginID = tl.NewKey(fz)
		} else if v.OriginID.Val != "" {
			errs = append(errs, NewInvalidFarezoneError("origin_id", v.OriginID.Val))
		}
		if fz, ok := e.zones[v.DestinationID.Val]; ok {
			v.DestinationID = tl.NewKey(fz)
		} else if v.DestinationID.Val != "" {
			errs = append(errs, NewInvalidFarezoneError("destination_id", v.DestinationID.Val))
		}
		if fz, ok := e.zones[v.ContainsID.Val]; ok {
			v.ContainsID = tl.NewKey(fz)
		} else if v.ContainsID.Val != "" {
			errs = append(errs, NewInvalidFarezoneError("contains_id", v.ContainsID.Val))
		}
	}
	return errs
}

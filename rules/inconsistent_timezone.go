package rules

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// InconsistentTimezoneError reports when agency.txt has more than 1 unique timezone present.
type InconsistentTimezoneError struct {
	bc
}

// NewInconsistentTimezoneError returns a new InconsistentTimezoneError.
func NewInconsistentTimezoneError(value string) *InconsistentTimezoneError {
	return &InconsistentTimezoneError{bc: bc{Value: value}}
}

func (e *InconsistentTimezoneError) Error() string {
	return "file contains inconsistent timezones"
}

// InconsistentTimezoneCheck checks for InconsistentTimezoneErrors.
type InconsistentTimezoneCheck struct {
	firstTimeZone string
}

// Validate .
func (e *InconsistentTimezoneCheck) Validate(ent tt.Entity) []error {
	v, ok := ent.(*gtfs.Agency)
	if !ok {
		return nil
	}
	if e.firstTimeZone == "" {
		e.firstTimeZone = v.AgencyTimezone
	}
	if v.AgencyTimezone != e.firstTimeZone {
		return []error{NewInconsistentTimezoneError(v.AgencyTimezone)}
	}
	return nil
}

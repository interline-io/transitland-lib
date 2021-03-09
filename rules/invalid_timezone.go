package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// AgencyInvalidTimezoneCheck returns an error if an agency has an invalid timezone.
// This is handled as a Best Practices warning because it is so common.
type AgencyInvalidTimezoneCheck struct{}

// Validate .
func (e *AgencyInvalidTimezoneCheck) Validate(ent tl.Entity) []error {
	var errs []error
	if v, ok := ent.(*tl.Agency); ok {
		if !enum.IsValidTimezone(v.AgencyTimezone) {
			errs = append(errs, causes.NewInvalidTimezoneError(v.AgencyID, "agency_timezone", v.AgencyTimezone))
		}
	}
	if v, ok := ent.(*tl.Stop); ok {
		if !enum.IsValidTimezone(v.StopTimezone) {
			errs = append(errs, causes.NewInvalidTimezoneError(v.StopID, "stop_timezone", v.StopTimezone))
		}
	}
	return errs
}

package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

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

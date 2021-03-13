package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// AgencyIDConditionallyRequiredCheck checks if agency_id is missing when more than one agency is present.
// This check is for when agency_id is *required* - see AgencyIDRecommendedCheck for when it is recommended but not required.
type AgencyIDConditionallyRequiredCheck struct {
	agencyCount int
}

// Validate .
func (e *AgencyIDConditionallyRequiredCheck) Validate(ent tl.Entity) []error {
	var errs []error
	switch v := ent.(type) {
	case *tl.FareAttribute:
		if v.AgencyID.Key == "" && e.agencyCount > 1 {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *tl.Route:
		if v.AgencyID == "" && e.agencyCount > 1 {
			// routes.agency_id is required when more than one agency is present
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *tl.Agency:
		e.agencyCount++
	}
	return errs
}

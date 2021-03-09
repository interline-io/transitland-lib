package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// AgencyIDConditionallyRecommendedCheck checks if agency_id is missing when more than one agency is present.
type AgencyIDConditionallyRecommendedCheck struct {
	agencyCount int
}

// Validate .
func (e *AgencyIDConditionallyRecommendedCheck) Validate(ent tl.Entity) []error {
	var errs []error
	switch v := ent.(type) {
	case *tl.Route:
		if v.AgencyID == "" && e.agencyCount == 1 {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *tl.Agency:
		e.agencyCount++
	}
	return errs
}

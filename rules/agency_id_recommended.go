package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tl"
)

// AgencyIDRecommendedCheck checks if agency_id is missing when more than one agency is present.
type AgencyIDRecommendedCheck struct {
	agencyCount     int
	defaultAgencyID string
}

// Validate .
func (e *AgencyIDRecommendedCheck) Validate(ent tl.Entity) []error {
	var errs []error
	switch v := ent.(type) {
	case *tl.Route:
		// If there is EXACTLY ONE agency, then the field can be omitted but with a warning.
		if v.AgencyID == "" && e.agencyCount == 1 {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *tl.Agency:
		// Missing agency.agency_id always gets a warning.
		e.agencyCount++
		if v.AgencyID == "" {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	}
	return errs
}

package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// AgencyIDConditionallyRequiredCheck checks if agency_id is missing when more than one agency is present.
// This check is for when agency_id is *required* - see AgencyIDRecommendedCheck for when it is recommended but not required.
type AgencyIDConditionallyRequiredCheck struct {
	agencyCount int
}

// Validate .
func (e *AgencyIDConditionallyRequiredCheck) Validate(ent tt.Entity) []error {
	var errs []error
	switch v := ent.(type) {
	case *gtfs.FareAttribute:
		if v.AgencyID.Val == "" && e.agencyCount > 1 {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *gtfs.Route:
		if v.AgencyID == "" && e.agencyCount > 1 {
			// routes.agency_id is required when more than one agency is present
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *gtfs.Agency:
		e.agencyCount++
	}
	return errs
}

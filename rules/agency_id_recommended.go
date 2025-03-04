package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// AgencyIDRecommendedCheck checks if agency_id is missing when more than one agency is present.
type AgencyIDRecommendedCheck struct {
	agencyCount int
}

// Validate .
func (e *AgencyIDRecommendedCheck) Validate(ent tt.Entity) []error {
	var errs []error
	switch v := ent.(type) {
	case *gtfs.Route:
		// If there is EXACTLY ONE agency, then the field can be omitted but with a warning.
		if v.AgencyID.Val == "" && e.agencyCount == 1 {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	case *gtfs.Agency:
		// Missing agency.agency_id always gets a warning.
		e.agencyCount++
		if !v.AgencyID.Valid {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("agency_id"))
		}
	}
	return errs
}

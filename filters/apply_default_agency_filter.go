package filters

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// ApplDefaultAgencyFilter sets a default agency_id value in relevant fields when the value is empty, e.g. routes.txt agency_id.
// It will only set a default agency_id value when the feed contains a single agency.
type ApplyDefaultAgencyFilter struct {
	defaultAgencyId string
	agencyCount     int
}

func (e *ApplyDefaultAgencyFilter) Filter(ent tl.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		if e.defaultAgencyId == "" {
			e.defaultAgencyId = v.AgencyID
		}
		e.agencyCount += 1
	case *tl.Route:
		if v.AgencyID == "" && e.agencyCount == 1 {
			v.AgencyID = e.defaultAgencyId
		}
	}
	return nil
}

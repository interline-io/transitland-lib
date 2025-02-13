package filters

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// ApplDefaultAgencyFilter sets a default agency_id value in relevant fields when the value is empty, e.g. routes.txt agency_id.
// It will only set a default agency_id value when the feed contains a single agency.
type ApplyDefaultAgencyFilter struct {
	defaultAgencyId string
	agencyCount     int
}

func (e *ApplyDefaultAgencyFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		if e.defaultAgencyId == "" {
			e.defaultAgencyId = v.AgencyID.Val
		}
		e.agencyCount += 1
	case *gtfs.Route:
		if !v.AgencyID.Valid && e.agencyCount == 1 {
			v.AgencyID.Set(e.defaultAgencyId)
		}
	}
	return nil
}

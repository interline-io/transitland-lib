package filters

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tt"
)

// ApplyParentTimezoneFilter sets timezone based on the default agency timezone or parent stop timezone
// Can be used with NormalizeTimezoneFilter
type ApplyParentTimezoneFilter struct {
	defaultAgencyTimezone string
	parentStopTimezones   map[string]string
}

func (e *ApplyParentTimezoneFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	// Remember filter happens before UpdateKeys or final ID available
	switch v := ent.(type) {
	case *tl.Agency:
		if e.defaultAgencyTimezone == "" {
			e.defaultAgencyTimezone = v.AgencyTimezone
		}
	case *tl.Stop:
		if v.StopTimezone == "" {
			// Use default agency timezone, unless a parent station provided a timezone
			v.StopTimezone = e.defaultAgencyTimezone
			if ptz, ok := e.parentStopTimezones[v.ParentStation.Val]; ok {
				v.StopTimezone = ptz
			}
		}
		if v.LocationType == 1 {
			if e.parentStopTimezones == nil {
				e.parentStopTimezones = map[string]string{}
			}
			e.parentStopTimezones[v.StopID] = v.StopTimezone
		}
	}
	return nil
}

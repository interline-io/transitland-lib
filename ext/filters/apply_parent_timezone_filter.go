package filters

import (
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// ApplyParentTimezoneFilter sets timezone based on the default agency timezone or parent stop timezone
// Can be used with NormalizeTimezoneFilter
type ApplyParentTimezoneFilter struct {
	defaultAgencyTimezone string
	parentStopTimezones   map[string]string
}

func (e *ApplyParentTimezoneFilter) Filter(ent tt.Entity, _ *tt.EntityMap) error {
	// Remember filter happens before UpdateKeys or final ID available
	switch v := ent.(type) {
	case *gtfs.Agency:
		if e.defaultAgencyTimezone == "" {
			e.defaultAgencyTimezone = v.AgencyTimezone.Val
		}
	case *gtfs.Stop:
		if v.StopTimezone.Val == "" {
			// Use default agency timezone, unless a parent station provided a timezone
			v.StopTimezone.Set(e.defaultAgencyTimezone)
			if ptz, ok := e.parentStopTimezones[v.ParentStation.Val]; ok {
				v.StopTimezone.Set(ptz)
			}
		}
		if v.LocationType.Val == 1 {
			if e.parentStopTimezones == nil {
				e.parentStopTimezones = map[string]string{}
			}
			e.parentStopTimezones[v.StopID.Val] = v.StopTimezone.Val
		}
	}
	return nil
}

package filters

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// NormalizeTimezoneFilter changes a timezone alias to a normalized timezone, e.g. "US/Pacific" -> "America/Los_Angeles"
type NormalizeTimezoneFilter struct{}

// Validate .
func (e *NormalizeTimezoneFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		n, ok := tt.IsValidTimezone(v.AgencyTimezone.Val)
		if !ok {
			return causes.NewInvalidTimezoneError("agency_timezone", v.AgencyTimezone.Val)
		} else {
			v.AgencyTimezone.Set(n)
		}
	case *gtfs.Stop:
		n, ok := tt.IsValidTimezone(v.StopTimezone)
		if !ok {
			return causes.NewInvalidTimezoneError("stop_timezone", v.StopTimezone)
		} else {
			v.StopTimezone = n
		}
	}
	return nil
}

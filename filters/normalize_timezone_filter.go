package filters

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// NormalizeTimezoneFilter changes a timezone alias to a normalized timezone, e.g. "US/Pacific" -> "America/Los_Angeles"
type NormalizeTimezoneFilter struct{}

// Validate .
func (e *NormalizeTimezoneFilter) Filter(ent tl.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		n, ok := tt.IsValidTimezone(v.AgencyTimezone)
		if !ok {
			return causes.NewInvalidTimezoneError("agency_timezone", v.AgencyTimezone)
		} else {
			v.AgencyEmail = n
		}
	case *tl.Stop:
		n, ok := tt.IsValidTimezone(v.StopTimezone)
		if !ok {
			return causes.NewInvalidTimezoneError("stop_timezone", v.StopTimezone)
		} else {
			v.StopTimezone = n
		}
	}
	return nil
}

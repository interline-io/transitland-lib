package copier

import (
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// BasicRouteTypeFilter checks for extended route_type's and converts to basic route_types.
type BasicRouteTypeFilter struct{}

// Filter converts extended route_types to basic route_types.
func (e *BasicRouteTypeFilter) Filter(ent tl.Entity, emap *tl.EntityMap) error {
	// Filters can edit in place, add entity errors, etc.
	v, ok := ent.(*tl.Route)
	if !ok {
		return nil
	}
	if rt, ok := enum.GetBasicRouteType(v.RouteType); ok {
		v.RouteType = rt.Code
	} else {
		return causes.NewInvalidFieldError("route_type", strconv.Itoa(v.RouteType), fmt.Errorf("cannot convert route_type %d to basic route type", v.RouteType))
	}
	return nil
}

// NormalizeTimezoneFilter changes a timezone alias to a normalized timezone, e.g. "US/Pacific" -> "America/Los_Angeles"
type NormalizeTimezoneFilter struct{}

// Validate .
func (e *NormalizeTimezoneFilter) Filter(ent tl.Entity) error {
	switch v := ent.(type) {
	case *tl.Agency:
		if n, ok := enum.IsValidTimezone(v.AgencyTimezone.String()); ok {
			v.AgencyTimezone = enum.NewTimezone(n)
		}
	case *tl.Stop:
		n, ok := enum.IsValidTimezone(v.StopTimezone)
		if !ok {
			return enum.NewInvalidTimezoneError(v.StopTimezone)
		} else {
			v.StopTimezone = n
		}
	}
	return nil
}

// ApplyParentTimezoneFilter sets timezone based on the default agency timezone or parent stop timezone
// Can be used with NormalizeTimezoneFilter
type ApplyParentTimezoneFilter struct {
	defaultAgencyTimezone tl.Timezone
	parentStopTimezones   map[string]string
}

func (e *ApplyParentTimezoneFilter) Filter(ent tl.Entity) []error {
	// Remember filter happens before UpdateKeys or final ID available
	switch v := ent.(type) {
	case *tl.Agency:
		if e.defaultAgencyTimezone.IsValid() {
			e.defaultAgencyTimezone = v.AgencyTimezone
		}
	case *tl.Stop:
		if v.StopTimezone == "" {
			// Use default agency timezone, unless a parent station provided a timezone
			v.StopTimezone = e.defaultAgencyTimezone.String()
			if ptz, ok := e.parentStopTimezones[v.ParentStation.Key]; ok {
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

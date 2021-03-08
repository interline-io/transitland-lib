package rules

import (
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// DuplicateRouteNameCheck checks for routes of the same agency with identical route_long_names.
type DuplicateRouteNameCheck struct {
	names map[string]int
}

// Validate .
func (e *DuplicateRouteNameCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tl.Route)
	if !ok {
		return nil
	}
	if e.names == nil {
		e.names = map[string]int{}
	}
	key := v.AgencyID + ":" + strconv.Itoa(v.RouteType) + ":" + v.RouteLongName // todo: use a real separator
	if _, ok := e.names[key]; ok {
		return []error{causes.NewValidationWarning("route_long_name", "duplicate route_long_name in same agency_id,route_type")}
	}
	e.names[key]++
	return nil
}

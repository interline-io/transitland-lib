package rules

import (
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

// DuplicateRouteNameError reports when routes of the same agency have identical route_long_name values.
type DuplicateRouteNameError struct {
	RouteLongName string
	RouteType     int
	AgencyID      string
	bc
}

func (e *DuplicateRouteNameError) Error() string {
	return fmt.Sprintf(
		"route '%s' with route_type %d and agency_id '%s' has the same route_long_name as another route of the same type and agency",
		e.RouteLongName,
		e.RouteType,
		e.AgencyID,
	)
}

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
		return []error{&DuplicateRouteNameError{
			RouteLongName: v.RouteLongName,
			RouteType:     v.RouteType,
			AgencyID:      v.AgencyID,
		}}
	}
	e.names[key]++
	return nil
}

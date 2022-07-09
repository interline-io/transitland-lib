package rules

import (
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

// DuplicateRouteNameError reports when routes of the same agency have identical route_long_name values.
type DuplicateRouteNameError struct {
	RouteID       string
	RouteLongName string
	RouteType     int
	AgencyID      string
	OtherRouteID  string
	bc
}

func (e *DuplicateRouteNameError) Error() string {
	return fmt.Sprintf(
		"route '%s' with route_long_name '%s', route_type %d, and agency_id '%s' has the same route_long_name, route_type and agency_id as route '%s'",
		e.RouteID,
		e.RouteLongName,
		e.RouteType,
		e.AgencyID,
		e.OtherRouteID,
	)
}

// DuplicateRouteNameCheck checks for routes of the same agency with identical route_long_names.
type DuplicateRouteNameCheck struct {
	names map[string]string
}

// Validate .
func (e *DuplicateRouteNameCheck) Validate(ent tl.Entity) []error {
	v, ok := ent.(*tl.Route)
	if !ok || !v.RouteLongName.Present() {
		return nil
	}
	if e.names == nil {
		e.names = map[string]string{}
	}
	key := v.AgencyID + ":" + strconv.Itoa(v.RouteType) + ":" + v.RouteLongName.String // todo: use a real separator
	if hit, ok := e.names[key]; ok {
		return []error{&DuplicateRouteNameError{
			RouteID:       v.RouteID,
			RouteLongName: v.RouteLongName.String,
			RouteType:     v.RouteType,
			AgencyID:      v.AgencyID,
			OtherRouteID:  hit,
		}}
	}
	e.names[key] = v.RouteID
	return nil
}

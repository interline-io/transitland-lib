package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
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
func (e *DuplicateRouteNameCheck) Validate(ent tt.Entity) []error {
	v, ok := ent.(*gtfs.Route)
	if !ok || !v.RouteLongName.Valid {
		return nil
	}
	if e.names == nil {
		e.names = map[string]string{}
	}
	key := v.AgencyID.Val + ":" + v.RouteType.String() + ":" + v.RouteLongName.Val // todo: use a real separator
	if hit, ok := e.names[key]; ok {
		return []error{&DuplicateRouteNameError{
			RouteID:       v.RouteID.Val,
			RouteLongName: v.RouteLongName.Val,
			RouteType:     v.RouteType.Int(),
			AgencyID:      v.AgencyID.Val,
			OtherRouteID:  hit,
		}}
	}
	e.names[key] = v.RouteID.Val
	return nil
}

package filters

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// BasicRouteTypeFilter checks for extended route_type's and converts to basic route_types.
type BasicRouteTypeFilter struct{}

// Filter converts extended route_types to basic route_types.
func (e *BasicRouteTypeFilter) Filter(ent tt.Entity, emap *tt.EntityMap) error {
	// Filters can edit in place, add entity errors, etc.
	v, ok := ent.(*gtfs.Route)
	if !ok {
		return nil
	}
	if rt, ok := tt.GetBasicRouteType(v.RouteType.Int()); ok {
		v.RouteType.SetInt(rt.Code)
	} else {
		return causes.NewInvalidFieldError("route_type", tt.TryCsv(v.RouteType), fmt.Errorf("cannot convert route_type %d to basic route type", v.RouteType))
	}
	return nil
}

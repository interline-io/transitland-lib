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
		v.AddError(causes.NewInvalidFieldError("route_type", strconv.Itoa(v.RouteType), fmt.Errorf("cannot convert route_type %d to basic route type", v.RouteType)))
	}
	return nil
}

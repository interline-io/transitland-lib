package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// DescriptionEqualsName checks that route_desc does not duplicate route_short_name or route_long_name.
type DescriptionEqualsName struct{}

// Validate .
func (e *DescriptionEqualsName) Validate(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Route); ok {
		if v.RouteDesc.Present() && (v.RouteDesc.Val == v.RouteLongName.Val || v.RouteDesc.Val == v.RouteShortName.Val) {
			return []error{causes.NewValidationWarning("route_desc", "route_desc should not duplicate route_short_name or route_long_name")}
		}
	}
	if v, ok := ent.(*tl.Stop); ok {
		if v.StopDesc != "" && v.StopDesc == v.StopName {
			return []error{causes.NewValidationWarning("stop_name", "stop_desc should not duplicate stop_name")}
		}
	}

	return nil
}

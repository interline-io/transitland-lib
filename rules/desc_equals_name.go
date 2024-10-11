package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// DescriptionEqualsName checks that route_desc does not duplicate route_short_name or route_long_name.
type DescriptionEqualsName struct{}

// Validate .
func (e *DescriptionEqualsName) Validate(ent tt.Entity) []error {
	if v, ok := ent.(*gtfs.Route); ok {
		if v.RouteDesc != "" && (v.RouteDesc == v.RouteLongName || v.RouteDesc == v.RouteShortName) {
			return []error{causes.NewValidationWarning("route_desc", "route_desc should not duplicate route_short_name or route_long_name")}
		}
	}
	if v, ok := ent.(*gtfs.Stop); ok {
		if v.StopDesc != "" && v.StopDesc == v.StopName {
			return []error{causes.NewValidationWarning("stop_name", "stop_desc should not duplicate stop_name")}
		}
	}

	return nil
}

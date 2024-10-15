package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// RouteShortNameTooLongCheck checks if route_short_name is, well, too long.
type RouteShortNameTooLongCheck struct{}

// Validate .
func (e *RouteShortNameTooLongCheck) Validate(ent tt.Entity) []error {
	if v, ok := ent.(*gtfs.Route); ok {
		if len(v.RouteShortName.Val) > 12 {
			return []error{causes.NewValidationWarning("route_short_name", "route_short_name should be no more than 12 characters")}
		}
	}
	return nil
}

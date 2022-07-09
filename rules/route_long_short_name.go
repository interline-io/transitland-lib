package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
)

// RouteShortNameTooLongCheck checks if route_short_name is, well, too long.
type RouteShortNameTooLongCheck struct{}

// Validate .
func (e *RouteShortNameTooLongCheck) Validate(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Route); ok {
		if len(v.RouteShortName.String) > 12 {
			return []error{causes.NewValidationWarning("route_short_name", "route_short_name should be no more than 12 characters")}
		}
	}
	return nil
}

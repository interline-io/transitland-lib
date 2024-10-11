package rules

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// RouteExtendedTypesCheck reports a Best Practices warning when extended route_type values are used.
// These are not well supported.
type RouteExtendedTypesCheck struct{}

// Validate .
func (e *RouteExtendedTypesCheck) Validate(ent tt.Entity) []error {
	if v, ok := ent.(*tl.Route); ok {
		if v.RouteType > 12 {
			return []error{causes.NewValidationWarning("route_type", "extended route_types not universally supported")}
		}
	}
	return nil
}

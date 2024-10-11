package rules

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tl"
)

// TODO: calculate actual contrast.

// InsufficientColorContrastCheck checks that when route_color and route_text_color are specified, sufficient contrast exists to be legible.
type InsufficientColorContrastCheck struct{}

// Validate .
func (e *InsufficientColorContrastCheck) Validate(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Route); ok {
		if v.RouteColor != "" && v.RouteColor == v.RouteTextColor {
			return []error{causes.NewValidationWarning("route_text_color", "route_text_color should provide contrast with route_color")}
		}
	}
	return nil
}

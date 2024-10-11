package rules

import (
	"strings"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteNamesPrefixError struct {
	bc
}

type RouteNamesPrefixCheck struct {
}

func (e *RouteNamesPrefixCheck) Validate(ent tt.Entity) []error {
	if v, ok := ent.(*tl.Route); ok {
		if v.RouteShortName != "" && v.RouteLongName != "" && strings.HasPrefix(v.RouteLongName, v.RouteShortName) {
			return []error{&RouteNamesPrefixError{}}
		}
	}
	return nil
}

package rules

import (
	"strings"

	"github.com/interline-io/transitland-lib/tl"
)

type RouteNamesPrefixError struct {
	bc
}

type RouteNamesPrefixCheck struct {
}

func (e *RouteNamesPrefixCheck) Validate(ent tl.Entity) []error {
	if v, ok := ent.(*tl.Route); ok {
		if len(v.RouteShortName) > 0 && len(v.RouteLongName) > 0 && strings.HasPrefix(v.RouteLongName, v.RouteShortName) {
			err := &RouteNamesPrefixError{}
			return []error{err}
		}
	}
	return nil
}

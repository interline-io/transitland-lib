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
		if v.RouteShortName != "" && v.RouteLongName != "" && strings.HasPrefix(v.RouteLongName, v.RouteShortName) {
			return []error{&RouteNamesPrefixError{}}
		}
	}
	return nil
}

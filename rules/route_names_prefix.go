package rules

import (
	"strings"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteNamesPrefixError struct {
	bc
}

type RouteNamesPrefixCheck struct {
}

func (e *RouteNamesPrefixCheck) Validate(ent tt.Entity) []error {
	if v, ok := ent.(*gtfs.Route); ok {
		if v.RouteShortName.Valid && v.RouteLongName.Valid && strings.HasPrefix(v.RouteLongName.Val, v.RouteShortName.Val) {
			return []error{&RouteNamesPrefixError{}}
		}
	}
	return nil
}

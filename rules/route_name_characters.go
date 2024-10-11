package rules

import (
	"regexp"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteNamesCharactersError struct {
	bc
}

type RouteNamesCharactersCheck struct{}

func (e *RouteNamesCharactersCheck) Validate(ent tt.Entity) []error {
	var errs []error
	if v, ok := ent.(*gtfs.Route); ok {
		if !routeNameCheckAllowedChars(v.RouteShortName) {
			err := RouteNamesCharactersError{}
			err.Field = "route_short_name"
			err.Value = v.RouteShortName
			errs = append(errs, &err)
		}
		if !routeNameCheckAllowedChars(v.RouteLongName) {
			err := RouteNamesCharactersError{}
			err.Field = "route_long_name"
			err.Value = v.RouteLongName
			errs = append(errs, &err)
		}
	}
	return errs
}

var routeNameallowedChars = regexp.MustCompile(`^[\.0-9\s\p{L}\(\)-/\&<>"']+$`)

func routeNameCheckAllowedChars(s string) bool {
	if s == "" {
		return true
	}
	return routeNameallowedChars.MatchString(s)
}

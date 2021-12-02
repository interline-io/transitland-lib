package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

type RouteNamesCharactersError struct {
	bc
}

type RouteNamesCharactersCheck struct{}

func (e *RouteNamesCharactersCheck) Validate(ent tl.Entity) []error {
	var errs []error
	if v, ok := ent.(*tl.Route); ok {
		if !checkAllowedChars(v.RouteShortName) {
			err := RouteNamesCharactersError{}
			err.Field = "route_short_name"
			err.Value = v.RouteShortName
			fmt.Println(err.Error())
			errs = append(errs, &err)
		}
		if !checkAllowedChars(v.RouteLongName) {
			err := RouteNamesCharactersError{}
			err.Field = "route_long_name"
			err.Value = v.RouteLongName
			fmt.Println(err.Error())
			errs = append(errs, &err)
		}
	}
	return errs
}

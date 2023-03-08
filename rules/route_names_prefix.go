package rules

import (
	"fmt"
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
		fmt.Println("checking:", v.RouteShortName, ":", v.RouteLongName)
		if v.RouteShortName != "" && v.RouteLongName != "" && strings.HasPrefix(v.RouteLongName, v.RouteShortName) {
			fmt.Println("prefixed")
			return []error{&RouteNamesPrefixError{}}
		}
	}
	return nil
}
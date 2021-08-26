package builders

import (
	"fmt"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
)

type OnestopIDBuilder struct {
	agencyRoutes map[string][]string
	routeStops   map[string]map[string]bool
}

func NewOnestopIDBuilder() *OnestopIDBuilder {
	return &OnestopIDBuilder{
		agencyRoutes: map[string][]string{},
		routeStops:   map[string]map[string]bool{},
	}
}

func (pp *OnestopIDBuilder) AfterValidator(ent tl.Entity) error {
	switch v := ent.(type) {
	case *tl.Route:
		pp.agencyRoutes[v.AgencyID] = append(pp.agencyRoutes[v.AgencyID], v.RouteID)
	case *tl.Trip:
		rid := v.RouteID
		rs, ok := pp.routeStops[rid]
		if !ok {
			rs = map[string]bool{}
			pp.routeStops[rid] = rs
		}
		for _, st := range v.StopTimes {
			rs[st.StopID] = true
		}
	}
	return nil
}

func (pp *OnestopIDBuilder) Copy(copier *copier.Copier) error {
	fmt.Println("OnestopIDBuilder Copy", pp.agencyRoutes, pp.routeStops)
	return nil
}

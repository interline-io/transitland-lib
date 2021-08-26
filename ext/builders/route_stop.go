package builders

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
)

type RouteStop struct {
	RouteID  string
	AgencyID string
	StopID   string
	tl.BaseEntity
}

func (rs *RouteStop) TableName() string {
	return "tl_route_stops"
}

func (rs *RouteStop) Filename() string {
	return "tl_route_stops.txt"
}

type RouteStopBuilder struct {
	routeAgencies map[string]string
	routeStops    map[string]map[string]bool
}

func NewRouteStopBuilder() *RouteStopBuilder {
	return &RouteStopBuilder{
		routeAgencies: map[string]string{},
		routeStops:    map[string]map[string]bool{},
	}
}

func (pp *RouteStopBuilder) AfterValidator(ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Route:
		pp.routeAgencies[v.RouteID] = v.AgencyID
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

func (pp *RouteStopBuilder) Copy(copier *copier.Copier) error {
	emap := copier.EntityMap
	bt := []tl.Entity{}
	for k, v := range pp.routeStops {
		rid, ok := emap.Get("routes.txt", k)
		if !ok {
			continue
		}
		aid, ok := emap.Get("agency.txt", pp.routeAgencies[k])
		if !ok {
			continue
		}
		for stopid := range v {
			if sid, ok := emap.Get("stops.txt", stopid); ok {
				bt = append(bt, &RouteStop{RouteID: rid, StopID: sid, AgencyID: aid})
			}
		}
	}
	if _, err := copier.Writer.AddEntities(bt); err != nil {
		return err
	}
	return nil
}

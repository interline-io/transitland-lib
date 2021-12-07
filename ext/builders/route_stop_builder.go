package builders

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
)

type RouteStop struct {
	RouteID  string
	AgencyID string
	StopID   string
	tl.MinEntity
	tl.FeedVersionEntity
}

func (rs *RouteStop) TableName() string {
	return "tl_route_stops"
}

func (rs *RouteStop) Filename() string {
	return "tl_route_stops.txt"
}

////////

type RouteStopBuilder struct {
	routeAgencies map[string]string
	tripRoutes    map[string]string
	routeStops    map[string]map[string]bool
}

func NewRouteStopBuilder() *RouteStopBuilder {
	return &RouteStopBuilder{
		routeAgencies: map[string]string{},
		tripRoutes:    map[string]string{},
		routeStops:    map[string]map[string]bool{},
	}
}

func (pp *RouteStopBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Route:
		pp.routeAgencies[eid] = v.AgencyID
	case *tl.Trip:
		pp.tripRoutes[eid] = v.RouteID
	case *tl.StopTime:
		rid := pp.tripRoutes[v.TripID]
		rs, ok := pp.routeStops[rid]
		if !ok {
			rs = map[string]bool{}
			pp.routeStops[rid] = rs
		}
		rs[v.StopID] = true
	}
	return nil
}

func (pp *RouteStopBuilder) Copy(copier *copier.Copier) error {
	bt := []tl.Entity{}
	for rid, v := range pp.routeStops {
		aid, ok := pp.routeAgencies[rid]
		if !ok {
			continue
		}
		for stopid := range v {
			bt = append(bt, &RouteStop{RouteID: rid, StopID: stopid, AgencyID: aid})
		}
	}
	if _, err := copier.Writer.AddEntities(bt); err != nil {
		return err
	}
	return nil
}

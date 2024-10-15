package builders

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

type RouteStop struct {
	RouteID  string
	AgencyID string
	StopID   string
	tt.MinEntity
	tt.FeedVersionEntity
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

func (pp *RouteStopBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Route:
		pp.routeAgencies[eid] = v.AgencyID
	case *gtfs.Trip:
		pp.tripRoutes[eid] = v.RouteID.Val
	case *gtfs.StopTime:
		rid := pp.tripRoutes[v.TripID.Val]
		rs, ok := pp.routeStops[rid]
		if !ok {
			rs = map[string]bool{}
			pp.routeStops[rid] = rs
		}
		rs[v.StopID.Val] = true
	}
	return nil
}

func (pp *RouteStopBuilder) Copy(copier *copier.Copier) error {
	bt := []tt.Entity{}
	for rid, v := range pp.routeStops {
		aid, ok := pp.routeAgencies[rid]
		if !ok {
			continue
		}
		for stopid := range v {
			bt = append(bt, &RouteStop{RouteID: rid, StopID: stopid, AgencyID: aid})
		}
	}
	if err := copier.CopyEntities(bt); err != nil {
		return err
	}
	return nil
}

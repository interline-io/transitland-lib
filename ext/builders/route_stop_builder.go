package builders

import (
	"github.com/interline-io/transitland-lib/adapters"
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
	routeStops    map[string]map[string]bool
}

func NewRouteStopBuilder() *RouteStopBuilder {
	return &RouteStopBuilder{
		routeAgencies: map[string]string{},
		routeStops:    map[string]map[string]bool{},
	}
}

func (pp *RouteStopBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Route:
		pp.routeAgencies[eid] = v.AgencyID.Val
	case *gtfs.Trip:
		// The trip carries its stop_times, so record the route's stops here from
		// v.RouteID rather than keeping a trip->route map until each StopTime arrives.
		// The attached stop_times are raw, so resolve stop ids through the EntityMap.
		rid := v.RouteID.Val
		var rs map[string]bool // bound on the first kept stop; nil until then avoids an empty route entry
		for _, st := range v.StopTimes {
			if !st.StopID.Valid {
				continue
			}
			stopId, ok := emap.Get("stops.txt", st.StopID.Val)
			if !ok {
				continue
			}
			if rs == nil {
				if rs = pp.routeStops[rid]; rs == nil {
					rs = map[string]bool{}
					pp.routeStops[rid] = rs
				}
			}
			rs[stopId] = true
		}
	}
	return nil
}

func (pp *RouteStopBuilder) Copy(copier adapters.EntityCopier) error {
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

package builders

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/xy"
	"maps"
	"slices"
)

//////////

type AgencyGeometry struct {
	AgencyID tt.Key
	Geometry tt.Polygon
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *AgencyGeometry) Filename() string {
	return "tl_agency_geometries.txt"
}

func (ent *AgencyGeometry) TableName() string {
	return "tl_agency_geometries"
}

//////////

// ConvexHullBuilder builds each agency's convex hull, from the stops its routes'
// trips visit.
type ConvexHullBuilder struct {
	stops          map[string]*stopGeom
	routeStopGeoms map[string]*routeStopGeoms
}

// NewConvexHullBuilder returns a ConvexHullBuilder.
func NewConvexHullBuilder() *ConvexHullBuilder {
	return &ConvexHullBuilder{
		stops:          map[string]*stopGeom{},
		routeStopGeoms: map[string]*routeStopGeoms{},
	}
}

// AfterWrite keeps track of which routes/agencies visit which stops
func (pp *ConvexHullBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Stop:
		pp.stops[eid] = &stopGeom{
			lon: v.Geometry.X(),
			lat: v.Geometry.Y(),
		}
	case *gtfs.Route:
		pp.routeStopGeoms[eid] = &routeStopGeoms{
			agency:    v.AgencyID.Val,
			stopGeoms: map[string]*stopGeom{},
		}
	case *gtfs.Trip:
		// Record the route's visited stop geometries from the trip's own stop_times
		// (see RouteStopBuilder); resolve stop ids through the EntityMap.
		r, ok := pp.routeStopGeoms[v.RouteID.Val]
		if !ok {
			return nil
		}
		for _, st := range v.StopTimes {
			if !st.StopID.Valid {
				continue
			}
			stopId, ok := emap.Get("stops.txt", st.StopID.Val)
			if !ok {
				continue
			}
			s, ok := pp.stops[stopId]
			if !ok {
				continue
			}
			r.stopGeoms[stopId] = s
		}
	}
	return nil
}

func (pp *ConvexHullBuilder) Copy(copier adapters.EntityCopier) error {
	agencyStops := map[string]map[string]*stopGeom{}
	for _, rsg := range pp.routeStopGeoms {
		r, ok := agencyStops[rsg.agency]
		if !ok {
			r = map[string]*stopGeom{}
			agencyStops[rsg.agency] = r
		}
		for stopid, sg := range rsg.stopGeoms {
			r[stopid] = sg
		}
	}
	// In agency id order, so the output doesn't depend on map order.
	for _, aid := range slices.Sorted(maps.Keys(agencyStops)) {
		coords := []float64{}
		for _, sg := range agencyStops[aid] {
			coords = append(coords, sg.lon, sg.lat)
		}
		ch := xy.ConvexHullFlat(geom.XY, coords)
		v, ok := ch.(*geom.Polygon)
		if !ok {
			// log.For(ctx).Debug().Msgf("agency convex hull is not polygon:", aid)
			continue
		}
		ent := AgencyGeometry{
			AgencyID: tt.NewKey(aid),
			Geometry: tt.NewPolygon(v),
		}
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

package builders

import (
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/xy"
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

type FeedVersionGeometry struct {
	Geometry tt.Polygon
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *FeedVersionGeometry) Filename() string {
	return "tl_feed_version_geometries.txt"
}

func (ent *FeedVersionGeometry) TableName() string {
	return "tl_feed_version_geometries"
}

//////////

type ConvexHullBuilder struct {
	stops              map[string]*stopGeom
	routeStopGeoms     map[string]*routeStopGeoms
	locationGeometries []tt.Geometry
}

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
	case *gtfs.Location:
		// Include location geometries in feed version convex hull
		pp.locationGeometries = append(pp.locationGeometries, v.Geometry)
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
	// build feed version convex hulls
	coords := []float64{}
	for _, v := range pp.stops {
		coords = append(coords, v.lon, v.lat)
	}
	for _, v := range pp.locationGeometries {
		coords = append(coords, v.FlatCoords()...)
	}
	ch := xy.ConvexHullFlat(geom.XY, coords)
	if v, ok := ch.(*geom.Polygon); ok {
		ent := FeedVersionGeometry{
			Geometry: tt.NewPolygon(v),
		}
		if err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}

	// now build agency convex hulls
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
	for aid, v := range agencyStops {
		coords := []float64{}
		for _, sg := range v {
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

package builders

import (
	"github.com/interline-io/transitland-lib/copier"
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
	stops          map[string]*stopGeom
	tripRoutes     map[string]string
	routeStopGeoms map[string]*routeStopGeoms
}

func NewConvexHullBuilder() *ConvexHullBuilder {
	return &ConvexHullBuilder{
		stops:          map[string]*stopGeom{},
		tripRoutes:     map[string]string{},
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
		pp.tripRoutes[eid] = v.RouteID.Val
	case *gtfs.StopTime:
		r, ok := pp.routeStopGeoms[pp.tripRoutes[v.TripID.Val]]
		if !ok {
			// log.Debugf("no route:", v.TripID, pp.tripRoutes[v.TripID])
			return nil
		}
		s, ok := pp.stops[v.StopID.Val]
		if !ok {
			// log.Debugf("no stop:", v.StopID)
			return nil
		}
		r.stopGeoms[v.StopID.Val] = s
	}
	return nil
}

func (pp *ConvexHullBuilder) Copy(copier *copier.Copier) error {
	// build feed version convex hulls
	fvStops := map[int][]*stopGeom{}
	for _, sg := range pp.stops {
		fvStops[sg.fvid] = append(fvStops[sg.fvid], sg)
	}
	for fvid, v := range fvStops {
		_ = fvid
		coords := []float64{}
		for _, coord := range v {
			coords = append(coords, coord.lon, coord.lat)
		}
		ch := xy.ConvexHullFlat(geom.XY, coords)
		v, ok := ch.(*geom.Polygon)
		if !ok {
			// log.Debugf("feed version convex hull is not polygon:", fvid)
			continue
		}
		ent := FeedVersionGeometry{
			Geometry: tt.NewPolygon(v),
		}
		if _, err := copier.CopyEntity(&ent); err != nil {
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
			// log.Debugf("agency convex hull is not polygon:", aid)
			continue
		}
		ent := AgencyGeometry{
			AgencyID: tt.NewKey(aid),
			Geometry: tt.NewPolygon(v),
		}
		if _, err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

package builders

import (
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/twpayne/go-geom"
	geomxy "github.com/twpayne/go-geom/xy"
)

//////////

type AgencyGeometry struct {
	AgencyID tl.OKey
	Geometry tl.Polygon
	tl.MinEntity
	tl.FeedVersionEntity
}

func (ent *AgencyGeometry) Filename() string {
	return "tl_agency_geometries.txt"
}

func (ent *AgencyGeometry) TableName() string {
	return "tl_agency_geometries"
}

//////////

type FeedVersionGeometry struct {
	Geometry tl.Polygon
	tl.MinEntity
	tl.FeedVersionEntity
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
func (pp *ConvexHullBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Stop:
		pp.stops[eid] = &stopGeom{
			lon: v.Geometry.X(),
			lat: v.Geometry.Y(),
		}
	case *tl.Route:
		pp.routeStopGeoms[eid] = &routeStopGeoms{
			agency:    v.AgencyID,
			stopGeoms: map[string]*stopGeom{},
		}
	case *tl.Trip:
		pp.tripRoutes[eid] = v.RouteID
	case *tl.StopTime:
		r, ok := pp.routeStopGeoms[pp.tripRoutes[v.TripID]]
		if !ok {
			log.Debug("no route:", v.TripID, pp.tripRoutes[v.TripID])
			return nil
		}
		s, ok := pp.stops[v.StopID]
		if !ok {
			log.Debug("no stop:", v.StopID)
			return nil
		}
		r.stopGeoms[v.StopID] = s
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
		coords := []float64{}
		for _, coord := range v {
			coords = append(coords, coord.lon, coord.lat)
		}
		ch := geomxy.ConvexHullFlat(geom.XY, coords)
		v, ok := ch.(*geom.Polygon)
		if !ok {
			log.Debug("feed version convex hull is not polygon:", fvid)
			continue
		}
		ent := FeedVersionGeometry{
			Geometry: tl.Polygon{Valid: true, Polygon: *v},
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
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
		ch := geomxy.ConvexHullFlat(geom.XY, coords)
		v, ok := ch.(*geom.Polygon)
		if !ok {
			log.Debug("agency convex hull is not polygon:", aid)
			continue
		}
		ent := AgencyGeometry{
			AgencyID: tl.NewOKey(aid),
			Geometry: tl.Polygon{Valid: true, Polygon: *v},
		}
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

package builders

import (
	"fmt"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/twpayne/go-geom"
	geomxy "github.com/twpayne/go-geom/xy"
)

//////////
// these structs also used by OnestopIDBuilder

type stopGeom struct {
	fvid int
	lat  float64
	lon  float64
}

type routeStopGeoms struct {
	agency    string
	stopGeoms map[string]*stopGeom
}

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

//////////

type FeedVersionGeometry struct {
	Geometry tl.Polygon
	tl.MinEntity
	tl.FeedVersionEntity
}

func (ent *FeedVersionGeometry) Filename() string {
	return "tl_feed_version_geometries.txt"
}

//////////

type ConvexHullBuilder struct {
	stops          map[string]*stopGeom
	routeStopGeoms map[string]*routeStopGeoms
}

func NewConvexHullBuilder() *ConvexHullBuilder {
	return &ConvexHullBuilder{
		stops:          map[string]*stopGeom{},
		routeStopGeoms: map[string]*routeStopGeoms{},
	}
}

// AfterValidator keeps track of which routes/agencies visit which stops
func (pp *ConvexHullBuilder) AfterValidator(ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Stop:
		pp.stops[v.StopID] = &stopGeom{
			lat:  v.Geometry.X(),
			lon:  v.Geometry.Y(),
			fvid: v.FeedVersionID,
		}
	case *tl.Route:
		pp.routeStopGeoms[v.RouteID] = &routeStopGeoms{
			agency:    v.AgencyID,
			stopGeoms: map[string]*stopGeom{},
		}
	case *tl.Trip:
		r, ok := pp.routeStopGeoms[v.RouteID]
		if !ok {
			fmt.Println("no route:", v.RouteID)
			return nil
		}
		for _, st := range v.StopTimes {
			s, ok := pp.stops[st.StopID]
			if !ok {
				fmt.Println("no stop:", st.StopID)
				return nil
			}
			r.stopGeoms[st.StopID] = s
		}
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
		gl := geom.NewLineStringFlat(geom.XY, coords)
		ch := geomxy.ConvexHull(gl)
		v, ok := ch.(*geom.Polygon)
		if !ok {
			fmt.Println("feed version convex hull is not polygon:", fvid)
			continue
		}
		ent := FeedVersionGeometry{
			Geometry: tl.Polygon{Valid: true, Polygon: *v},
		}
		ent.FeedVersionID = fvid
		fmt.Println(ent.FeedVersionID, ent.Geometry.String())
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			return err
		}
	}
	// now build agency convex hulls
	agencyStops := map[string][]*stopGeom{}
	for _, rsg := range pp.routeStopGeoms {
		for _, sg := range rsg.stopGeoms {
			agencyStops[rsg.agency] = append(agencyStops[rsg.agency], sg)
		}
	}
	for aid, v := range agencyStops {
		coords := []float64{}
		for _, sg := range v {
			coords = append(coords, sg.lon, sg.lat)
		}
		gl := geom.NewLineStringFlat(geom.XY, coords)
		ch := geomxy.ConvexHull(gl)
		v, ok := ch.(*geom.Polygon)
		if !ok {
			fmt.Println("agency convex hull is not polygon:", aid)
			continue
		}
		ent := AgencyGeometry{
			AgencyID: tl.NewOKey(aid),
			Geometry: tl.Polygon{Valid: true, Polygon: *v},
		}
		fmt.Println(ent.Geometry.String())
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

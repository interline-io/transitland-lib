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
	stops          map[string]*stopGeom
	locations      map[string]*locationGeom
	tripRoutes     map[string]string
	routeStopGeoms map[string]*routeStopGeoms
}

func NewConvexHullBuilder() *ConvexHullBuilder {
	return &ConvexHullBuilder{
		stops:          map[string]*stopGeom{},
		locations:      map[string]*locationGeom{},
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
	case *gtfs.Location:
		pp.locations[eid] = &locationGeom{
			fvid:   v.FeedVersionID,
			coords: v.Geometry.FlatCoords(),
			stride: v.Geometry.Stride(),
		}
	case *gtfs.Route:
		pp.routeStopGeoms[eid] = &routeStopGeoms{
			agency:    v.AgencyID.Val,
			stopGeoms: map[string]*stopGeom{},
		}
	case *gtfs.Trip:
		pp.tripRoutes[eid] = v.RouteID.Val
	case *gtfs.StopTime:
		if !v.StopID.Valid {
			return nil
		}
		r, ok := pp.routeStopGeoms[pp.tripRoutes[v.TripID.Val]]
		if !ok {
			// log.For(ctx).Debug().Msgf("no route:", v.TripID, pp.tripRoutes[v.TripID])
			return nil
		}
		s, ok := pp.stops[v.StopID.Val]
		if !ok {
			// log.For(ctx).Debug().Msgf("no stop:", v.StopID)
			return nil
		}
		r.stopGeoms[v.StopID.Val] = s
	}
	return nil
}

func (pp *ConvexHullBuilder) Copy(copier adapters.EntityCopier) error {
	// build feed version convex hulls
	fvStops := map[int][]*stopGeom{}
	for _, sg := range pp.stops {
		fvStops[sg.fvid] = append(fvStops[sg.fvid], sg)
	}
	fvLocations := map[int][]*locationGeom{}
	for _, lg := range pp.locations {
		fvLocations[lg.fvid] = append(fvLocations[lg.fvid], lg)
	}

	allFvids := map[int]struct{}{}
	for k := range fvStops {
		allFvids[k] = struct{}{}
	}
	for k := range fvLocations {
		allFvids[k] = struct{}{}
	}

	for fvid := range allFvids {
		coords := []float64{}
		if stops, ok := fvStops[fvid]; ok {
			for _, coord := range stops {
				coords = append(coords, coord.lon, coord.lat)
			}
		}
		if locs, ok := fvLocations[fvid]; ok {
			for _, loc := range locs {
				if loc.stride == 2 {
					coords = append(coords, loc.coords...)
				} else {
					for i := 0; i < len(loc.coords); i += loc.stride {
						coords = append(coords, loc.coords[i], loc.coords[i+1])
					}
				}
			}
		}

		ch := xy.ConvexHullFlat(geom.XY, coords)
		v, ok := ch.(*geom.Polygon)
		if !ok {
			// log.For(ctx).Debug().Msgf("feed version convex hull is not polygon:", fvid)
			continue
		}
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

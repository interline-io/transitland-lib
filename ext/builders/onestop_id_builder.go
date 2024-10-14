package builders

import (
	"fmt"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/mmcloughlin/geohash"
)

type StopOnestopID struct {
	StopID    string
	OnestopID string
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *StopOnestopID) Filename() string {
	return "tl_stop_onestop_ids.txt"
}

func (ent *StopOnestopID) TableName() string {
	return "tl_stop_onestop_ids"
}

type RouteOnestopID struct {
	RouteID   string
	OnestopID string
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *RouteOnestopID) Filename() string {
	return "tl_route_onestop_ids.txt"
}
func (ent *RouteOnestopID) TableName() string {
	return "tl_route_onestop_ids"
}

type AgencyOnestopID struct {
	AgencyID  string
	OnestopID string
	tt.MinEntity
	tt.FeedVersionEntity
}

func (ent *AgencyOnestopID) Filename() string {
	return "tl_agency_onestop_ids.txt"
}

func (ent *AgencyOnestopID) TableName() string {
	return "tl_agency_onestop_ids"
}

// OnestopID Builder

type OnestopIDBuilder struct {
	agencyNames    map[string]string
	stops          map[string]*stopGeom
	tripRoutes     map[string]string
	routeStopGeoms map[string]*routeStopGeoms
}

func NewOnestopIDBuilder() *OnestopIDBuilder {
	return &OnestopIDBuilder{
		agencyNames:    map[string]string{},
		stops:          map[string]*stopGeom{},
		tripRoutes:     map[string]string{},
		routeStopGeoms: map[string]*routeStopGeoms{},
	}
}

func (pp *OnestopIDBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		pp.agencyNames[eid] = v.AgencyName
	case *gtfs.Stop:
		pp.stops[eid] = &stopGeom{
			name: v.StopName,
			lon:  v.Geometry.X(),
			lat:  v.Geometry.Y(),
		}
	case *gtfs.Route:
		name := v.RouteShortName
		if name == "" {
			name = v.RouteLongName
		}
		pp.routeStopGeoms[eid] = &routeStopGeoms{
			name:      name,
			agency:    v.AgencyID,
			stopGeoms: map[string]*stopGeom{},
		}
	case *gtfs.Trip:
		pp.tripRoutes[eid] = v.RouteID
	case *gtfs.StopTime:
		r, ok := pp.routeStopGeoms[pp.tripRoutes[v.TripID.Val]]
		if !ok {
			// log.Debugf("OnestopIDBuilder no route:", v.TripID, pp.tripRoutes[v.TripID])
			return nil
		}
		s, ok := pp.stops[v.StopID.Val]
		if !ok {
			// log.Debugf("OnestopIDBuilder no stop:", v.StopID)
			return nil
		}
		r.stopGeoms[v.StopID.Val] = s
	}
	return nil
}

func (pp *OnestopIDBuilder) AgencyOnestopIDs() []AgencyOnestopID {
	// group stops by agency
	var ret []AgencyOnestopID
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
	// generate agency onestop id's
	for aid, sgs := range agencyStops {
		name := pp.agencyNames[aid]
		if name == "" {
			name = aid
		}
		pts := []point{}
		for _, sg := range sgs {
			pts = append(pts, point{Lon: sg.lon, Lat: sg.lat})
		}
		if gh := pointsGeohash(pts, 1, 6); len(gh) > 0 {
			ent := AgencyOnestopID{
				AgencyID:  aid,
				OnestopID: fmt.Sprintf("o-%s-%s", gh, filterName(name)),
			}
			ret = append(ret, ent)
		}
	}
	return ret
}

func (pp *OnestopIDBuilder) StopOnestopIDs() []StopOnestopID {
	// generate stop onestop id's
	var ret []StopOnestopID
	for stopid, sg := range pp.stops {
		if gh := geohash.EncodeWithPrecision(sg.lat, sg.lon, 10); len(gh) > 0 {
			ent := StopOnestopID{
				StopID:    stopid,
				OnestopID: fmt.Sprintf("s-%s-%s", gh, filterName(sg.name)),
			}
			ret = append(ret, ent)
		}
	}
	return ret
}

func (pp *OnestopIDBuilder) RouteOnestopIDs() []RouteOnestopID {
	var ret []RouteOnestopID
	for rid, rsg := range pp.routeStopGeoms {
		pts := []point{}
		for _, sg := range rsg.stopGeoms {
			pts = append(pts, point{Lon: sg.lon, Lat: sg.lat})
		}
		if gh := pointsGeohash(pts, 1, 6); len(gh) > 0 {
			ent := RouteOnestopID{
				RouteID:   rid,
				OnestopID: fmt.Sprintf("r-%s-%s", gh, filterName(rsg.name)),
			}
			ret = append(ret, ent)
		}
	}
	return ret
}

func (pp *OnestopIDBuilder) Copy(copier *copier.Copier) error {
	for _, ent := range pp.AgencyOnestopIDs() {
		ent := ent
		if _, err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	for _, ent := range pp.RouteOnestopIDs() {
		ent := ent
		if _, err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	for _, ent := range pp.StopOnestopIDs() {
		ent := ent
		if _, err := copier.CopyEntity(&ent); err != nil {
			return err
		}
	}
	return nil
}

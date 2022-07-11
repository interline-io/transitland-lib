package builders

import (
	"fmt"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/mmcloughlin/geohash"
)

type StopOnestopID struct {
	StopID    string
	OnestopID string
	tl.MinEntity
	tl.FeedVersionEntity
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
	tl.MinEntity
	tl.FeedVersionEntity
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
	tl.MinEntity
	tl.FeedVersionEntity
}

func (ent *AgencyOnestopID) Filename() string {
	return "tl_agency_onestop_ids.txt"
}

func (ent *AgencyOnestopID) TableName() string {
	return "tl_agency_onestop_ids"
}

//////////

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

func (pp *OnestopIDBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		pp.agencyNames[eid] = v.AgencyName
	case *tl.Stop:
		pp.stops[eid] = &stopGeom{
			name: v.StopName,
			lon:  v.Geometry.X(),
			lat:  v.Geometry.Y(),
		}
	case *tl.Route:
		name := v.RouteShortName.Val
		if name == "" {
			name = v.RouteLongName.Val
		}
		pp.routeStopGeoms[eid] = &routeStopGeoms{
			name:      name,
			agency:    v.AgencyID,
			stopGeoms: map[string]*stopGeom{},
		}
	case *tl.Trip:
		pp.tripRoutes[eid] = v.RouteID
	case *tl.StopTime:
		r, ok := pp.routeStopGeoms[pp.tripRoutes[v.TripID]]
		if !ok {
			// log.Debugf("OnestopIDBuilder no route:", v.TripID, pp.tripRoutes[v.TripID])
			return nil
		}
		s, ok := pp.stops[v.StopID]
		if !ok {
			// log.Debugf("OnestopIDBuilder no stop:", v.StopID)
			return nil
		}
		r.stopGeoms[v.StopID] = s
	}
	return nil
}

func (pp *OnestopIDBuilder) Copy(copier *copier.Copier) error {
	// generate stop onestop id's
	for stopid, sg := range pp.stops {
		if gh := geohash.EncodeWithPrecision(sg.lat, sg.lon, 10); len(gh) > 0 {
			ent := StopOnestopID{
				StopID:    stopid,
				OnestopID: fmt.Sprintf("s-%s-%s", gh, filterName(sg.name)),
			}
			if _, _, err := copier.CopyEntity(&ent); err != nil {
				return err
			}
		}
	}
	// generate route onstop id's
	for rid, rsg := range pp.routeStopGeoms {
		pts := []point{}
		for _, sg := range rsg.stopGeoms {
			pts = append(pts, point{lon: sg.lon, lat: sg.lat})
		}
		if gh := pointsGeohash(pts, 1, 6); len(gh) > 0 {
			ent := RouteOnestopID{
				RouteID:   rid,
				OnestopID: fmt.Sprintf("r-%s-%s", gh, filterName(rsg.name)),
			}
			if _, _, err := copier.CopyEntity(&ent); err != nil {
				return err
			}
		}
	}
	// group stops by agency
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
			pts = append(pts, point{lon: sg.lon, lat: sg.lat})
		}
		if gh := pointsGeohash(pts, 1, 6); len(gh) > 0 {
			ent := AgencyOnestopID{
				AgencyID:  aid,
				OnestopID: fmt.Sprintf("o-%s-%s", gh, filterName(name)),
			}
			if _, _, err := copier.CopyEntity(&ent); err != nil {
				return err
			}
		}
	}
	return nil
}

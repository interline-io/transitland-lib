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

//////////

type RouteOnestopID struct {
	RouteID   string
	OnestopID string
	tl.MinEntity
	tl.FeedVersionEntity
}

func (ent *RouteOnestopID) Filename() string {
	return "tl_route_onestop_ids.txt"
}

//////////

type AgencyOnestopID struct {
	AgencyID  string
	OnestopID string
	tl.MinEntity
	tl.FeedVersionEntity
}

func (ent *AgencyOnestopID) Filename() string {
	return "tl_agency_onestop_ids.txt"
}

//////////

type OnestopIDBuilder struct {
	stops          map[string]*stopGeom
	routeStopGeoms map[string]*routeStopGeoms
}

func NewOnestopIDBuilder() *OnestopIDBuilder {
	return &OnestopIDBuilder{
		stops:          map[string]*stopGeom{},
		routeStopGeoms: map[string]*routeStopGeoms{},
	}
}

func (pp *OnestopIDBuilder) AfterValidator(ent tl.Entity, emap *tl.EntityMap) error {
	// same as ConvexHullBuilder
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

func (pp *OnestopIDBuilder) Copy(copier *copier.Copier) error {
	// generate stop onestop id's
	stoposids := map[string]string{}
	for stopid, sg := range pp.stops {
		ent := StopOnestopID{
			StopID:    stopid,
			OnestopID: geohash.EncodeWithPrecision(sg.lat, sg.lon, 10),
		}
		stoposids[stopid] = ent.OnestopID
		if _, err := copier.Writer.AddEntity(&ent); err != nil {
			return err
		}
	}
	// generate route onstop id's
	return nil
}

package builders

import (
	"database/sql"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/mmcloughlin/geohash"
)

type AgencyPlace struct {
	AgencyID string
	Name     tl.String
	Adm1name tl.String
	Adm0name tl.String
	Count    int
	Rank     float64
	tl.MinEntity
	tl.FeedVersionEntity
}

func (rs *AgencyPlace) TableName() string {
	return "tl_agency_places"
}

func (rs *AgencyPlace) Filename() string {
	return "tl_agency_places.txt"
}

////////

type AgencyPlaceBuilder struct {
	stops       map[string]string // store just geohash
	routeAgency map[string]string
	tripAgency  map[string]string
	agencyStops map[string]map[string]int
}

func NewAgencyPlaceBuilder() *AgencyPlaceBuilder {
	return &AgencyPlaceBuilder{
		stops:       map[string]string{},
		routeAgency: map[string]string{},
		tripAgency:  map[string]string{},
		agencyStops: map[string]map[string]int{},
	}
}

func (pp *AgencyPlaceBuilder) AfterWrite(eid string, ent tl.Entity, emap *tl.EntityMap) error {
	switch v := ent.(type) {
	case *tl.Agency:
		pp.agencyStops[eid] = map[string]int{}
	case *tl.Stop:
		spt := v.ToPoint()
		pp.stops[eid] = geohash.EncodeWithPrecision(spt.Lat, spt.Lon, 6) // Note reversed coords
	case *tl.Route:
		pp.routeAgency[eid] = v.AgencyID
	case *tl.Trip:
		pp.tripAgency[eid] = pp.routeAgency[v.RouteID]
	case *tl.StopTime:
		aid := pp.tripAgency[v.TripID]
		if sg, ok := pp.stops[v.StopID]; ok {
			if v, ok := pp.agencyStops[aid]; ok {
				v[sg] += 1
			}
		}
	}
	return nil
}

var agencyPlaceQuery = `
select 
	ne.name, 
	coalesce(neadmin.name, ne.adm1name) as adm1name,
	coalesce(neadmin.admin, ne.adm0name) as adm0name,
	ST_Distance(ne.geometry, ST_MakePoint(?, ?)::geography) as distance 
from ne_10m_populated_places ne 
left join ne_10m_admin_1_states_provinces neadmin on ST_Intersects(ne.geometry, neadmin.geometry)
where st_dwithin(ne.geometry, ST_MakePoint(?, ?)::geography, 40000) 
order by distance asc
limit 1
`

var agencyAdminQuery = `
select 
	name adm1name,
	ne.admin adm0name
from ne_10m_admin_1_states_provinces ne
where st_intersects(ne.geometry, ST_MakePoint(?, ?));
`

func (pp *AgencyPlaceBuilder) Copy(copier *copier.Copier) error {
	// get places for each point
	ghPoints := map[string][]string{}
	for stopId, ghPoint := range pp.stops {
		ghPoints[ghPoint] = append(ghPoints[ghPoint], stopId)
	}
	dbWriter, ok := copier.Writer.(*tldb.Writer)
	if !ok {
		// log.Traceln("writer is not dbwriter")
		return nil
	}
	db := dbWriter.Adapter
	if _, ok := db.(*tldb.PostgresAdapter); !ok {
		// log.Traceln("only postgres is supported")
		return nil
	}
	// For each geohash, check nearby populated places and inside admin boundaries
	type foundPlace struct {
		Name     tl.String
		Adm1name tl.String
		Adm0name tl.String
	}
	pointPlaces := map[string]foundPlace{}
	pointAdmins := map[string]foundPlace{}
	for ghPoint := range ghPoints {
		gLat, gLon := geohash.Decode(ghPoint)
		r := []foundPlace{}
		if err := db.Select(&r, agencyPlaceQuery, gLon, gLat, gLon, gLat); err == sql.ErrNoRows {
			// ok
		} else if err != nil {
			return nil
		}
		if len(r) > 0 {
			pointPlaces[ghPoint] = r[0]
		}
	}
	for ghPoint := range ghPoints {
		gLat, gLon := geohash.Decode(ghPoint)
		r := []foundPlace{}
		if err := db.Select(&r, agencyAdminQuery, gLon, gLat); err == sql.ErrNoRows {
			// ok
		} else if err != nil {
			return nil
		}
		if len(r) > 0 {
			pointAdmins[ghPoint] = r[0]
		}
	}
	for aid, agencyPoints := range pp.agencyStops {
		// log.Traceln("agency stops:", agencyPoints)
		placeWeights := map[foundPlace]int{}
		agencyTotalWeight := 0
		for ghPoint, count := range agencyPoints {
			agencyTotalWeight += count
			if place, ok := pointAdmins[ghPoint]; ok {
				placeWeights[place] += count
			}
			if place, ok := pointPlaces[ghPoint]; ok {
				// include if we have a match for state/country, or no state/country matches
				checkPlace := foundPlace{
					Adm1name: place.Adm1name,
					Adm0name: place.Adm0name,
				}
				if _, ok2 := placeWeights[checkPlace]; ok2 || len(pointAdmins) == 0 {
					placeWeights[place] += count
				}
			}
		}
		// log.Traceln("aid:", aid, "total weight:", agencyTotalWeight)
		for k, v := range placeWeights {
			score := float64(v) / float64(agencyTotalWeight)
			if score > 0.05 {
				// log.Traceln("\tplace:", k.Name.String, "/", k.Adm1name.String, "/", k.Adm0name.String, "weight:", v, "score:", score)
				ap := AgencyPlace{}
				ap.AgencyID = aid
				ap.Name = k.Name
				ap.Adm0name = k.Adm0name
				ap.Adm1name = k.Adm1name
				ap.Count = v
				ap.Rank = score
				if _, err := copier.CopyEntity(&ap); err != nil {
					return err
				}

			}
		}
	}
	////////

	return nil
}

package builders

import (
	"context"
	"database/sql"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tt"
	"github.com/mmcloughlin/geohash"
)

type AgencyPlace struct {
	AgencyID string
	Name     tt.String
	Adm1name tt.String
	Adm0name tt.String
	Count    int
	Rank     float64
	tt.MinEntity
	tt.FeedVersionEntity
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

func (pp *AgencyPlaceBuilder) AfterWrite(eid string, ent tt.Entity, emap *tt.EntityMap) error {
	switch v := ent.(type) {
	case *gtfs.Agency:
		pp.agencyStops[eid] = map[string]int{}
	case *gtfs.Stop:
		spt := v.ToPoint()
		pp.stops[eid] = geohash.EncodeWithPrecision(spt.Lat, spt.Lon, 6) // Note reversed coords
	case *gtfs.Route:
		pp.routeAgency[eid] = v.AgencyID.Val
	case *gtfs.Trip:
		pp.tripAgency[eid] = pp.routeAgency[v.RouteID.Val]
	case *gtfs.StopTime:
		aid := pp.tripAgency[v.TripID.Val]
		if sg, ok := pp.stops[v.StopID.Val]; ok {
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

func (pp *AgencyPlaceBuilder) Copy(copier adapters.EntityCopier) error {
	ctx := context.TODO()
	// get places for each point
	ghPoints := map[string][]string{}
	for stopId, ghPoint := range pp.stops {
		ghPoints[ghPoint] = append(ghPoints[ghPoint], stopId)
	}
	dbWriter, ok := copier.Writer().(*tldb.Writer)
	if !ok {
		log.For(ctx).Trace().Msg("AgencyPlaceBuilder: skipping, writer is not dbwriter")
		return nil
	}
	db := dbWriter.Adapter
	if _, ok := db.(*postgres.PostgresAdapter); !ok {
		log.For(ctx).Trace().Msg("AgencyPlaceBuilder: skipping, only postgres is supported")
		return nil
	}
	// For each geohash, check nearby populated places and inside admin boundaries
	type foundPlace struct {
		Name     tt.String
		Adm1name tt.String
		Adm0name tt.String
	}
	pointPlaces := map[string]foundPlace{}
	pointAdmins := map[string]foundPlace{}
	for ghPoint := range ghPoints {
		gLat, gLon := geohash.Decode(ghPoint)
		r := []foundPlace{}
		if err := db.Select(ctx, &r, agencyPlaceQuery, gLon, gLat, gLon, gLat); err == sql.ErrNoRows {
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
		if err := db.Select(ctx, &r, agencyAdminQuery, gLon, gLat); err == sql.ErrNoRows {
			// ok
		} else if err != nil {
			return nil
		}
		if len(r) > 0 {
			pointAdmins[ghPoint] = r[0]
		}
	}
	var ents []tt.Entity
	for aid, agencyPoints := range pp.agencyStops {
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
		for k, v := range placeWeights {
			score := float64(v) / float64(agencyTotalWeight)
			if score > 0.05 {
				ap := AgencyPlace{}
				ap.AgencyID = aid
				ap.Name = k.Name
				ap.Adm0name = k.Adm0name
				ap.Adm1name = k.Adm1name
				ap.Count = v
				ap.Rank = score
				ents = append(ents, &ap)
			}
		}
	}
	return copier.CopyEntities(ents)
}

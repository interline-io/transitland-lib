package builders

import (
	"cmp"
	"context"
	"slices"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/interline-io/transitland-lib/tlxy"
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

// Place is a populated place or a first-level admin area, with the admin area
// and country it lies in. An admin area has no Name.
type Place struct {
	Name     tt.String
	Adm1name tt.String
	Adm0name tt.String
}

// PlaceLookup finds the places around a point.
type PlaceLookup interface {
	// NearestPlace returns the populated place nearest pt, within maxMeters.
	NearestPlace(ctx context.Context, pt tlxy.Point, maxMeters float64) (Place, bool, error)
	// Admin returns the first-level admin area holding pt.
	Admin(ctx context.Context, pt tlxy.Point) (Place, bool, error)
}

// placeRadius is how far from a stop its populated place can be, in meters.
const placeRadius = 40000.0

////////

type AgencyPlaceBuilder struct {
	places      PlaceLookup
	stops       map[string]string // store just geohash
	routeAgency map[string]string
	agencyStops map[string]map[string]int
}

// NewAgencyPlaceBuilder returns a builder that finds each agency's places
// through places. With places nil, it reads a Postgres writer's Natural Earth
// tables, and skips any other writer.
func NewAgencyPlaceBuilder(places PlaceLookup) *AgencyPlaceBuilder {
	return &AgencyPlaceBuilder{
		places:      places,
		stops:       map[string]string{},
		routeAgency: map[string]string{},
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
		// Tally the agency's stop geohashes from the trip's own stop_times rather than
		// keeping a trip->agency map for every trip. With journey-pattern dedup this also
		// counts deduplicated trips, weighting by service frequency; stop ids resolve
		// through the EntityMap.
		agencyStops, ok := pp.agencyStops[pp.routeAgency[v.RouteID.Val]]
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
			if sg, ok := pp.stops[stopId]; ok {
				agencyStops[sg] += 1
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
where st_dwithin(ne.geometry, ST_MakePoint(?, ?)::geography, ?)
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

// postgresPlaces looks places up in a database's Natural Earth tables.
type postgresPlaces struct {
	db tldb.Adapter
}

func (p postgresPlaces) NearestPlace(ctx context.Context, pt tlxy.Point, maxMeters float64) (Place, bool, error) {
	var r []Place
	if err := p.db.Select(ctx, &r, agencyPlaceQuery, pt.Lon, pt.Lat, pt.Lon, pt.Lat, maxMeters); err != nil {
		return Place{}, false, err
	}
	if len(r) == 0 {
		return Place{}, false, nil
	}
	return r[0], true, nil
}

func (p postgresPlaces) Admin(ctx context.Context, pt tlxy.Point) (Place, bool, error) {
	var r []Place
	if err := p.db.Select(ctx, &r, agencyAdminQuery, pt.Lon, pt.Lat); err != nil {
		return Place{}, false, err
	}
	if len(r) == 0 {
		return Place{}, false, nil
	}
	return r[0], true, nil
}

func (pp *AgencyPlaceBuilder) Copy(copier adapters.EntityCopier) error {
	ctx := context.TODO()
	places := pp.places
	if places == nil {
		dbWriter, ok := copier.Writer().(*tldb.Writer)
		if !ok {
			log.For(ctx).Trace().Msg("AgencyPlaceBuilder: skipping, no place lookup and writer is not dbwriter")
			return nil
		}
		if _, ok := dbWriter.Adapter.(*postgres.PostgresAdapter); !ok {
			log.For(ctx).Trace().Msg("AgencyPlaceBuilder: skipping, only postgres is supported")
			return nil
		}
		places = postgresPlaces{db: dbWriter.Adapter}
	}
	// For each geohash, find the nearby populated place and the admin area holding it
	cells := map[string]bool{}
	for _, ghPoint := range pp.stops {
		cells[ghPoint] = true
	}
	pointPlaces := map[string]Place{}
	pointAdmins := map[string]Place{}
	for ghPoint := range cells {
		gLat, gLon := geohash.Decode(ghPoint)
		pt := tlxy.Point{Lon: gLon, Lat: gLat}
		place, ok, err := places.NearestPlace(ctx, pt, placeRadius)
		if err != nil {
			return err
		}
		if ok {
			pointPlaces[ghPoint] = place
		}
		admin, ok, err := places.Admin(ctx, pt)
		if err != nil {
			return err
		}
		if ok {
			pointAdmins[ghPoint] = admin
		}
	}
	var aps []*AgencyPlace
	for aid, agencyPoints := range pp.agencyStops {
		placeWeights := map[Place]int{}
		agencyTotalWeight := 0
		for ghPoint, count := range agencyPoints {
			agencyTotalWeight += count
			if admin, ok := pointAdmins[ghPoint]; ok {
				placeWeights[admin] += count
			}
		}
		// Count a populated place only in an admin area the agency's stops reach,
		// or anywhere when no stop lies in any admin area. The admin areas are
		// tallied first, so the answer doesn't depend on map order.
		for ghPoint, count := range agencyPoints {
			place, ok := pointPlaces[ghPoint]
			if !ok {
				continue
			}
			admin := Place{Adm1name: place.Adm1name, Adm0name: place.Adm0name}
			if _, ok := placeWeights[admin]; ok || len(pointAdmins) == 0 {
				placeWeights[place] += count
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
				aps = append(aps, &ap)
			}
		}
	}
	// Write in a fixed order, since readers order places by row id.
	slices.SortFunc(aps, func(a, b *AgencyPlace) int {
		return cmp.Or(
			cmp.Compare(a.AgencyID, b.AgencyID),
			cmp.Compare(b.Rank, a.Rank),
			cmp.Compare(a.Name.Val, b.Name.Val),
			cmp.Compare(a.Adm1name.Val, b.Adm1name.Val),
			cmp.Compare(a.Adm0name.Val, b.Adm0name.Val),
		)
	})
	var ents []tt.Entity
	for _, ap := range aps {
		ents = append(ents, ap)
	}
	return copier.CopyEntities(ents)
}

package builders

import (
	"cmp"
	"slices"

	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/schema/ne"
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

// placeRadius is how far from a stop its populated place can be, in meters.
const placeRadius = 40000.0

////////

type AgencyPlaceBuilder struct {
	stops       map[string]string // store just geohash
	routeAgency map[string]string
	agencyStops map[string]map[string]int
}

// NewAgencyPlaceBuilder returns a builder that finds each agency's places in
// the embedded Natural Earth data.
func NewAgencyPlaceBuilder() *AgencyPlaceBuilder {
	return &AgencyPlaceBuilder{
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

func (pp *AgencyPlaceBuilder) Copy(copier adapters.EntityCopier) error {
	places, err := ne.Places()
	if err != nil {
		return err
	}
	// For each geohash, find the nearby populated place and the admin area holding it
	cells := map[string]bool{}
	for _, ghPoint := range pp.stops {
		cells[ghPoint] = true
	}
	pointPlaces := map[string]ne.Place{}
	pointAdmins := map[string]ne.Place{}
	for ghPoint := range cells {
		gLat, gLon := geohash.Decode(ghPoint)
		pt := tlxy.Point{Lon: gLon, Lat: gLat}
		if place, ok := places.NearestPlace(pt, placeRadius); ok {
			pointPlaces[ghPoint] = place
		}
		if admin, ok := places.Admin(pt); ok {
			pointAdmins[ghPoint] = admin
		}
	}
	var aps []*AgencyPlace
	for aid, agencyPoints := range pp.agencyStops {
		placeWeights := map[ne.Place]int{}
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
			admin := ne.Place{Adm1name: place.Adm1name, Adm0name: place.Adm0name}
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

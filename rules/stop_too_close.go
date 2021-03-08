package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/mmcloughlin/geohash"
)

///////////////

// StopTooCloseError .
type StopTooCloseError struct {
	StopID      string
	OtherStopID string
	Distance    float64
	bc
}

func (e *StopTooCloseError) Error() string {
	return fmt.Sprintf(
		"stop '%s' is too close to another stop '%s' at %0.2fm",
		e.StopID,
		e.OtherStopID,
		e.Distance,
	)
}

type stopPoint struct {
	id  string
	lat float64
	lon float64
}

// StopTooCloseCheck checks if two stops are within 1m
type StopTooCloseCheck struct {
	geoms   map[string][]*stopPoint
	maxdist float64
}

// Validate .
func (e *StopTooCloseCheck) Validate(ent tl.Entity) []error {
	e.maxdist = 1.0
	if e.geoms == nil {
		e.geoms = map[string][]*stopPoint{}
	}
	v, ok := ent.(*tl.Stop)
	// This only checks location_type == 0 and no parent
	if !ok || v.ParentStation.Key != "" || v.LocationType != 0 || !v.Geometry.Valid {
		return nil
	}
	// Use geohash for fast neighbor search; precision = 9 is approx 5m x 5m at the equator.
	coords := v.Geometry.Coords()
	if len(coords) < 2 {
		return nil
	}
	var errs []error
	gh := geohash.EncodeWithPrecision(coords[0], coords[1], 9)
	neighbors := geohash.Neighbors(gh)
	neighbors = append(neighbors, gh)
	g := stopPoint{id: v.StopID, lat: coords[0], lon: coords[1]}
	for _, neighbor := range neighbors {
		if hits, ok := e.geoms[neighbor]; ok {
			for _, hit := range hits {
				d := xy.DistanceHaversine(g.lon, g.lat, hit.lon, hit.lat)
				if d < e.maxdist {
					errs = append(errs, &StopTooCloseError{
						StopID:      v.StopID,
						OtherStopID: hit.id,
						Distance:    d,
					})
				}
			}
		}
	}
	// add to index
	e.geoms[gh] = append(e.geoms[gh], &g)
	return errs
}

package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/mmcloughlin/geohash"
)

///////////////

// StopTooCloseError reports when two stops of location_type = 0 that have no parent are within 1m of each other.
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
	id string
	pt tlxy.Point
}

// StopTooCloseCheck checks for StopTooCloseErrors.
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
	if !ok || v.ParentStation.Val != "" || v.LocationType != 0 || !v.Geometry.Valid {
		return nil
	}
	// Use geohash for fast neighbor search; precision = 9 is approx 5m x 5m at the equator.
	spt := v.ToPoint()
	if spt.Lon == 0 && spt.Lat == 0 {
		return nil // 0,0 is handled elsewhere
	}
	var errs []error
	gh := geohash.EncodeWithPrecision(spt.Lat, spt.Lon, 9) // Note reversed order
	neighbors := geohash.Neighbors(gh)
	neighbors = append(neighbors, gh)
	g := stopPoint{
		id: v.StopID,
		pt: spt,
	}
	for _, neighbor := range neighbors {
		if hits, ok := e.geoms[neighbor]; ok {
			for _, hit := range hits {
				d := tlxy.DistanceHaversine(g.pt, hit.pt)
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

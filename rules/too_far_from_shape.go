package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

// StopTooFarFromShapeError reports when a stop is too far from a shape.
type StopTooFarFromShapeError struct {
	TripID   string
	StopID   string
	ShapeID  string
	Distance float64
	bc
}

func (e *StopTooFarFromShapeError) Error() string {
	return fmt.Sprintf("trip '%s' has stop '%s' that is too far from shape '%s' at %0.2fm", e.TripID, e.StopID, e.ShapeID, e.Distance)
}

// StopTooFarFromShapeCheck checks for StopTooFarFromShapeErrors.
type StopTooFarFromShapeCheck struct {
	maxdist   float64
	geomCache tlxy.GeomCache // share stop/shape geometry cache with copier
	checked   map[string]map[string]bool
}

// SetGeomCache sets a shared geometry cache.
func (e *StopTooFarFromShapeCheck) SetGeomCache(g tlxy.GeomCache) {
	e.geomCache = g
}

// Validate .
func (e *StopTooFarFromShapeCheck) Validate(ent tt.Entity) []error {
	// An initial approach used geohashes to check shape <-> stop as an initial filter, but it turns
	// out in practice that just checking directly is almost exactly the same speed.
	// Even the largest feeds are only a few tens of thousands of comparisons. Just keep track
	// of comparisons that have already been made and it's fine.
	e.maxdist = 100.0
	v, ok := ent.(*gtfs.Trip)
	if !ok {
		return nil
	}
	if e.checked == nil {
		e.checked = map[string]map[string]bool{}
	}
	shapeid := v.ShapeID.Val
	if shapeid == "" || len(v.StopTimes) == 0 {
		return nil
	}
	if e.checked[shapeid] == nil {
		e.checked[shapeid] = map[string]bool{}
	}
	var errs []error
	for _, st := range v.StopTimes {
		// Check the cache
		if e.checked[shapeid][st.StopID.Val] {
			continue
		}
		e.checked[shapeid][st.StopID.Val] = true
		g := e.geomCache.GetStop(st.StopID.Val)
		sgeom := e.geomCache.GetShape(shapeid)
		nearest, _, _ := tlxy.LineClosestPoint(sgeom, g)
		distance := tlxy.DistanceHaversine(g, nearest)
		if distance > e.maxdist {
			errs = append(errs, &StopTooFarFromShapeError{
				TripID:   v.TripID,
				StopID:   st.StopID.Val,
				ShapeID:  shapeid,
				Distance: distance,
			})
		}
	}
	return errs
}

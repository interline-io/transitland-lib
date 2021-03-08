package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
)

// StopTooFarFromShapeError reports when a stop is too far from a shape.
type StopTooFarFromShapeError struct {
	StopID   string
	ShapeID  string
	Distance float64
	bc
}

// NewStopTooFarFromShapeError .
func NewStopTooFarFromShapeError(stopid string, shapeid string, distance float64) *StopTooFarFromShapeError {
	return &StopTooFarFromShapeError{
		StopID:   stopid,
		ShapeID:  shapeid,
		Distance: distance,
	}
}

func (e *StopTooFarFromShapeError) Error() string {
	return fmt.Sprintf("stop '%s' is too far from shape '%s' at %0.2fm", e.StopID, e.ShapeID, e.Distance)
}

// StopTooFarFromShapeCheck checks if a stop is more than 100m from an associated shape.
type StopTooFarFromShapeCheck struct {
	maxdist   float64
	geomCache *xy.GeomCache // share stop/shape geometry cache with copier
	checked   map[string]map[string]bool
}

// SetGeomCache sets a shared geometry cache.
func (e *StopTooFarFromShapeCheck) SetGeomCache(g *xy.GeomCache) {
	e.geomCache = g
}

// Validate .
func (e *StopTooFarFromShapeCheck) Validate(ent tl.Entity) []error {
	// An initial approach used geohashes to check shape <-> stop as an initial filter, but it turns
	// out in practice that just checking directly is almost exactly the same speed.
	// Even the largest feeds are only a few tens of thousands of comparisons. Just keep track
	// of comparisons that have already been made and it's fine.
	e.maxdist = 100.0
	v, ok := ent.(*tl.Trip)
	if !ok {
		return nil
	}
	if e.checked == nil {
		e.checked = map[string]map[string]bool{}
	}
	shapeid := v.ShapeID.Key
	if shapeid == "" || len(v.StopTimes) == 0 {
		return nil
	}
	if e.checked[shapeid] == nil {
		e.checked[shapeid] = map[string]bool{}
	}
	var errs []error
	for _, st := range v.StopTimes {
		// Check the cache
		if e.checked[shapeid][st.StopID] {
			continue
		}
		e.checked[shapeid][st.StopID] = true
		g := e.geomCache.GetStop(st.StopID)
		sgeom := e.geomCache.GetShape(shapeid)
		nearest, _ := xy.LineClosestPoint(sgeom, g)
		distance := xy.DistanceHaversine(g[0], g[1], nearest[0], nearest[1])
		if distance > e.maxdist {
			errs = append(errs, NewStopTooFarFromShapeError(st.StopID, shapeid, distance))
		}
	}
	return errs
}

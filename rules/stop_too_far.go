package rules

import (
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
)

// StopTooFarError .
type StopTooFarError struct{ bc }

// NewStopTooFarError .
func NewStopTooFarError() *StopTooFarError {
	return &StopTooFarError{}
}

func (e *StopTooFarError) Error() string {
	return "stop too far from parent"
}

// StopTooFarCheck checks if two related stops are >1km away.
type StopTooFarCheck struct {
	geoms   map[string]*tl.Point // regularize and use copier geomCache?
	maxdist float64
}

// Validate .
func (e *StopTooFarCheck) Validate(ent tl.Entity) []error {
	e.maxdist = 1000.0
	if e.geoms == nil {
		e.geoms = map[string]*tl.Point{}
	}
	v, ok := ent.(*tl.Stop)
	if !ok {
		return nil
	}
	var errs []error
	coords := v.Geometry.Coords()
	newp := tl.NewPoint(coords[0], coords[1]) // copy
	e.geoms[v.StopID] = &newp
	if v.ParentStation.Key == "" {
		return nil
	}
	// Check if parent stop is >1km
	if pgeom, ok := e.geoms[v.ParentStation.Key]; ok {
		// if not ok, then it's a parent error and out of scope for this check
		d := xy.DistanceHaversinePoint(coords, pgeom.Coords())
		if d > e.maxdist {
			errs = append(errs, NewStopTooFarError())
		}
	}
	return errs
}

package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/tl"
)

// StopTooFarError reports when two related stops are >1km away.
type StopTooFarError struct {
	StopID        string
	ParentStation string
	Distance      float64
	bc
}

func (e *StopTooFarError) Error() string {
	return fmt.Sprintf(
		"stop '%s' is too far from parent stop '%s' at %0.2fm",
		e.StopID,
		e.ParentStation,
		e.Distance,
	)
}

// StopTooFarCheck checks for StopTooFarErrors.
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
	if coords[0] == 0 && coords[1] == 0 {
		return nil // 0,0 handled elsewhere
	}
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
			errs = append(errs, &StopTooFarError{
				StopID:        v.StopID,
				ParentStation: v.ParentStation.Key,
				Distance:      d,
			})
		}
	}
	return errs
}

package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
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
	geoms   map[string]tlxy.Point // use shared geom cache?
	maxdist float64
}

// Validate .
func (e *StopTooFarCheck) Validate(ent tt.Entity) []error {
	e.maxdist = 1000.0
	if e.geoms == nil {
		e.geoms = map[string]tlxy.Point{}
	}
	v, ok := ent.(*tl.Stop)
	if !ok {
		return nil
	}
	var errs []error
	spoint := v.ToPoint()
	if spoint.Lon == 0 && spoint.Lat == 0 {
		return nil // 0,0 handled elsewhere
	}
	e.geoms[v.StopID] = spoint
	if v.ParentStation.Val == "" {
		return nil
	}
	// Check if parent stop is >1km
	if pgeom, ok := e.geoms[v.ParentStation.Val]; ok {
		// if not ok, then it's a parent error and out of scope for this check
		d := tlxy.DistanceHaversine(spoint, pgeom)
		if d > e.maxdist {
			errs = append(errs, &StopTooFarError{
				StopID:        v.StopID,
				ParentStation: v.ParentStation.Val,
				Distance:      d,
			})
		}
	}
	return errs
}

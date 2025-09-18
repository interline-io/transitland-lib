package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

type ShapeMaxSegmentLengthError struct {
	MaxAllowedDistance float64
	MaxDistance        float64
	bc
}

func (e ShapeMaxSegmentLengthError) Error() string {
	return fmt.Sprintf("shape segment length exceeds maximum allowed distance: %f > %f", e.MaxDistance, e.MaxAllowedDistance)
}

type ShapeMaxSegmentLengthCheck struct {
	MaxAllowedDistance float64
}

func (e *ShapeMaxSegmentLengthCheck) Validate(ent tt.Entity) []error {
	if e.MaxAllowedDistance <= 0 {
		return nil
	}
	v, ok := ent.(*service.ShapeLine)
	if !ok {
		return nil
	}
	var errs []error
	maxLength := 0.0
	pts := v.Geometry.ToPoints()
	if len(pts) < 2 {
		return nil
	}
	lastPt := pts[0]
	for _, pt := range pts {
		d := tlxy.DistanceHaversine(lastPt, pt)
		if d > maxLength {
			maxLength = d
		}
		lastPt = pt
	}
	if maxLength > e.MaxAllowedDistance {
		errs = append(errs, ShapeMaxSegmentLengthError{
			MaxAllowedDistance: e.MaxAllowedDistance,
			MaxDistance:        maxLength,
		})
	}
	return errs
}

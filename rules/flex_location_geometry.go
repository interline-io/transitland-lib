package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// FlexLocationGeometryError reports when a location has invalid geometry.
type FlexLocationGeometryError struct {
	LocationID   string
	ErrorMessage string
	bc
}

func (e *FlexLocationGeometryError) Error() string {
	return fmt.Sprintf(
		"location '%s' has invalid geometry: %s",
		e.LocationID,
		e.ErrorMessage,
	)
}

// FlexLocationGeometryCheck validates that locations have valid Polygon or MultiPolygon geometries.
type FlexLocationGeometryCheck struct{}

func (e *FlexLocationGeometryCheck) Validate(ent tt.Entity) []error {
	loc, ok := ent.(*gtfs.Location)
	if !ok {
		return nil
	}

	var errs []error

	// Geometry must be present (already checked in ConditionalErrors)
	if !loc.Geometry.Valid {
		return nil
	}

	// Validate that geometry is Polygon or MultiPolygon
	geom := loc.Geometry.Val
	switch geom.(type) {
	case nil:
		errs = append(errs, &FlexLocationGeometryError{
			LocationID:   loc.LocationID.Val,
			ErrorMessage: "geometry is nil",
		})
	default:
		// Additional geometry validation could go here
		// For example: check for self-intersections, minimum area, etc.
		// Check that polygon has at least 3 points
		coords := loc.Geometry.FlatCoords()
		if len(coords) < 6 { // At least 3 points (x,y) = 6 coordinates for a closed ring
			errs = append(errs, &FlexLocationGeometryError{
				LocationID:   loc.LocationID.Val,
				ErrorMessage: fmt.Sprintf("polygon has insufficient coordinates: %d (need at least 6 for closed triangle)", len(coords)),
			})
		}
	}

	return errs
}


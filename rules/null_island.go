package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// ZeroCoordinateError reports when a required geometry has a (0,0) coordinate.
type ZeroCoordinateError struct{ bc }

func (e *ZeroCoordinateError) Error() string {
	return fmt.Sprintf("entity '%s' has coordinates that include (0,0)", e.EntityID)
}

// NullIslandCheck checks for ZeroCoordinateError.
type NullIslandCheck struct{}

// Validate .
func (e *NullIslandCheck) Validate(ent tt.Entity) []error {
	switch v := ent.(type) {
	case *gtfs.Stop:
		if v.LocationType.Val == 3 || v.LocationType.Val == 4 {
			return nil // allowed
		}
		coords := v.Coordinates()
		if coords[0] == 0 && coords[1] == 0 {
			return []error{&ZeroCoordinateError{bc: bc{Field: "stop_lat", EntityID: v.StopID.Val, Message: "stop has (0,0) coordinates"}}}
		}
	case *gtfs.Shape:
		for _, coords := range v.Geometry.Val.Coords() {
			if coords[0] == 0 && coords[1] == 0 {
				return []error{&ZeroCoordinateError{bc: bc{Field: "shape_pt_lon", EntityID: v.ShapeID.Val, Message: "shape has (0,0) coordinates"}}}
			}
		}
	}
	return nil
}

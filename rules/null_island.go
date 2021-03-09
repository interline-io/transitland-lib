package rules

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl"
)

// ZeroCoordinateError reports when a required geometry has a (0,0) coordinate.
type ZeroCoordinateError struct{ bc }

func (e *ZeroCoordinateError) Error() string {
	return fmt.Sprintf("invalid geometry")
}

// NullIslandCheck checks for ZeroCoordinateError.
type NullIslandCheck struct{}

// Validate .
func (e *NullIslandCheck) Validate(ent tl.Entity) []error {
	switch v := ent.(type) {
	case *tl.Stop:
		if v.LocationType == 3 {
			return nil // allowed
		}
		coords := v.Coordinates()
		if coords[0] == 0 && coords[1] == 0 {
			return []error{&ZeroCoordinateError{bc: bc{Field: "stop_lat", Message: "stop has (0,0) coordinates"}}}
		}
	case *tl.Shape:
		for _, coords := range v.Geometry.Coords() {
			if coords[0] == 0 && coords[1] == 0 {
				return []error{&ZeroCoordinateError{bc: bc{Field: "shape_pt_lon", Message: "shape has (0,0) coordinates"}}}
			}
		}
	}
	return nil
}

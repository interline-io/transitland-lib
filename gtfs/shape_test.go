package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestShape_Errors(t *testing.T) {
	newShape := func(fn func(*Shape)) *Shape {
		shape := &Shape{
			ShapeID:           tt.NewString("ok"),
			ShapePtLat:        tt.NewFloat(36.641496),
			ShapePtLon:        tt.NewFloat(-116.40094),
			ShapePtSequence:   tt.NewInt(1),
			ShapeDistTraveled: tt.NewFloat(1.0),
		}
		if fn != nil {
			fn(shape)
		}
		return shape
	}

	testcases := []struct {
		name           string
		entity         *Shape
		expectedErrors []testutil.ExpectError
	}{
		{
			name:           "Valid shape",
			entity:         newShape(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing shape_pt_lat",
			entity: newShape(func(s *Shape) {
				s.ShapePtLat = tt.Float{}
			}),
			expectedErrors: PE("RequiredFieldError:shape_pt_lat"),
		},
		{
			name: "Missing shape_pt_lon",
			entity: newShape(func(s *Shape) {
				s.ShapePtLon = tt.Float{}
			}),
			expectedErrors: PE("RequiredFieldError:shape_pt_lon"),
		},
		{
			name: "Missing shape_pt_sequence",
			entity: newShape(func(s *Shape) {
				s.ShapePtSequence = tt.Int{}
			}),
			expectedErrors: PE("RequiredFieldError:shape_pt_sequence"),
		},
		{
			name: "Invalid shape_pt_lat (too low)",
			entity: newShape(func(s *Shape) {
				s.ShapePtLat = tt.NewFloat(-91.0)
			}),
			expectedErrors: PE("InvalidFieldError:shape_pt_lat"),
		},
		{
			name: "Invalid shape_pt_lat (too high)",
			entity: newShape(func(s *Shape) {
				s.ShapePtLat = tt.NewFloat(91.0)
			}),
			expectedErrors: PE("InvalidFieldError:shape_pt_lat"),
		},
		{
			name: "Invalid shape_pt_lon (too low)",
			entity: newShape(func(s *Shape) {
				s.ShapePtLon = tt.NewFloat(-181.0)
			}),
			expectedErrors: PE("InvalidFieldError:shape_pt_lon"),
		},
		{
			name: "Invalid shape_pt_lon (too high)",
			entity: newShape(func(s *Shape) {
				s.ShapePtLon = tt.NewFloat(181.0)
			}),
			expectedErrors: PE("InvalidFieldError:shape_pt_lon"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestArea_Errors(t *testing.T) {
	newArea := func(fn func(*Area)) *Area {
		area := &Area{
			AreaID:   tt.NewString("ok"),
			AreaName: tt.NewString("Valid Area"),
		}
		if fn != nil {
			fn(area)
		}
		return area
	}

	tests := []struct {
		name           string
		area           *Area
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid area",
			area:           newArea(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing area_id (required field)",
			area: newArea(func(a *Area) {
				a.AreaID = tt.String{}
			}),
			expectedErrors: PE("RequiredFieldError:area_id"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.area)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestStopArea_Errors(t *testing.T) {
	newStopArea := func(fn func(*StopArea)) *StopArea {
		stopArea := &StopArea{
			AreaID: tt.NewKey("area1"),
			StopID: tt.NewKey("stop1"),
		}
		if fn != nil {
			fn(stopArea)
		}
		return stopArea
	}

	testcases := []struct {
		name           string
		entity         *StopArea
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid stop_area",
			entity:         newStopArea(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing area_id",
			entity: newStopArea(func(sa *StopArea) {
				sa.AreaID = tt.Key{}
			}),
			expectedErrors: PE("RequiredFieldError:area_id"),
		},
		{
			name: "Missing stop_id",
			entity: newStopArea(func(sa *StopArea) {
				sa.StopID = tt.Key{}
			}),
			expectedErrors: PE("RequiredFieldError:stop_id"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

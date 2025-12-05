package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestTrip_Errors(t *testing.T) {
	newTrip := func(fn func(*Trip)) *Trip {
		trip := &Trip{
			TripID:               tt.NewString("ok"),
			RouteID:              tt.NewKey("ok"),
			ServiceID:            tt.NewKey("ok"),
			TripShortName:        tt.NewString("valid"),
			TripHeadsign:         tt.NewString("valid"),
			DirectionID:          tt.NewInt(0),
			BlockID:              tt.NewString("0"),
			ShapeID:              tt.NewKey("ok"),
			WheelchairAccessible: tt.NewInt(1),
			BikesAllowed:         tt.NewInt(1),
		}
		if fn != nil {
			fn(trip)
		}
		return trip
	}

	tests := []struct {
		name           string
		trip           *Trip
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid trip",
			trip:           newTrip(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing trip_id (required field)",
			trip: newTrip(func(t *Trip) {
				t.TripID = tt.String{}
			}),
			expectedErrors: PE("RequiredFieldError:trip_id"),
		},
		{
			name: "Missing route_id (required field)",
			trip: newTrip(func(t *Trip) {
				t.RouteID = tt.Key{}
			}),
			expectedErrors: PE("RequiredFieldError:route_id"),
		},
		{
			name: "Missing service_id (required field)",
			trip: newTrip(func(t *Trip) {
				t.ServiceID = tt.Key{}
			}),
			expectedErrors: PE("RequiredFieldError:service_id"),
		},
		{
			name: "Invalid direction_id",
			trip: newTrip(func(t *Trip) {
				t.DirectionID = tt.NewInt(100)
			}),
			expectedErrors: PE("InvalidFieldError:direction_id"),
		},
		{
			name: "Invalid wheelchair_accessible",
			trip: newTrip(func(t *Trip) {
				t.WheelchairAccessible = tt.NewInt(100)
			}),
			expectedErrors: PE("InvalidFieldError:wheelchair_accessible"),
		},
		{
			name: "Invalid bikes_allowed",
			trip: newTrip(func(t *Trip) {
				t.BikesAllowed = tt.NewInt(100)
			}),
			expectedErrors: PE("InvalidFieldError:bikes_allowed"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.trip)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

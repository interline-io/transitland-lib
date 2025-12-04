package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestStop_Errors(t *testing.T) {
	newStop := func(fn func(*Stop)) *Stop {
		stop := &Stop{
			StopID:             tt.NewString("ok_stop"),
			StopName:           tt.NewString("Valid Stop"),
			StopDesc:           tt.NewString("A good stop"),
			StopLat:            tt.NewFloat(36.641496),
			StopLon:            tt.NewFloat(-116.40094),
			ZoneID:             tt.NewString("ok_zone"),
			LocationType:       tt.NewInt(0),
			WheelchairBoarding: tt.NewInt(0),
		}
		if fn != nil {
			fn(stop)
		}
		return stop
	}

	tests := []struct {
		name           string
		stop           *Stop
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid stop",
			stop:           newStop(nil),
			expectedErrors: nil,
		},
		{
			name: "Valid station (location_type=1)",
			stop: newStop(func(s *Stop) {
				s.StopID = tt.NewString("ok_station")
				s.StopName = tt.NewString("Station")
				s.StopDesc = tt.NewString("A good station")
				s.LocationType = tt.NewInt(1)
			}),
			expectedErrors: nil,
		},
		{
			name: "Missing stop_id (required field)",
			stop: newStop(func(s *Stop) {
				s.StopID = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:stop_id"),
		},
		{
			name: "Invalid stop_url",
			stop: newStop(func(s *Stop) {
				s.StopURL = tt.NewUrl("abcxyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:stop_url"),
		},
		{
			name: "Missing stop_name for stop/platform (location_type=0)",
			stop: newStop(func(s *Stop) {
				s.StopName = tt.String{}
				s.LocationType = tt.NewInt(0)
			}),
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:stop_name"),
		},
		{
			name: "Valid: no stop_name for node (location_type=3)",
			stop: newStop(func(s *Stop) {
				s.StopName = tt.String{}
				s.LocationType = tt.NewInt(3)
				s.ParentStation = tt.NewKey("ok_station")
			}),
			expectedErrors: nil,
		},
		{
			name: "Valid: no stop_name for boarding area (location_type=4)",
			stop: newStop(func(s *Stop) {
				s.StopName = tt.String{}
				s.LocationType = tt.NewInt(4)
				s.ParentStation = tt.NewKey("ok_stop")
			}),
			expectedErrors: nil,
		},
		{
			name: "Invalid stop_lat < -90",
			stop: newStop(func(s *Stop) {
				s.StopLat = tt.NewFloat(-91.0)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:stop_lat"),
		},
		{
			name: "Invalid stop_lon < -180",
			stop: newStop(func(s *Stop) {
				s.StopLon = tt.NewFloat(-181.0)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:stop_lon"),
		},
		{
			name: "Invalid stop_lat > 90",
			stop: newStop(func(s *Stop) {
				s.StopLat = tt.NewFloat(91.0)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:stop_lat"),
		},
		{
			name: "Invalid stop_lon > 180",
			stop: newStop(func(s *Stop) {
				s.StopLon = tt.NewFloat(181.0)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:stop_lon"),
		},
		{
			name: "Valid: stop_lat = 0 for node (location_type=3)",
			stop: newStop(func(s *Stop) {
				s.StopLat = tt.NewFloat(0.0)
				s.LocationType = tt.NewInt(3)
				s.ParentStation = tt.NewKey("ok_station")
			}),
			expectedErrors: nil,
		},
		{
			name: "Valid: stop_lon = 0 for node (location_type=3)",
			stop: newStop(func(s *Stop) {
				s.StopLon = tt.NewFloat(0.0)
				s.LocationType = tt.NewInt(3)
				s.ParentStation = tt.NewKey("ok_station")
			}),
			expectedErrors: nil,
		},
		{
			name: "Valid: missing stop_lat for boarding area (location_type=4)",
			stop: newStop(func(s *Stop) {
				s.StopLat = tt.Float{}
				s.LocationType = tt.NewInt(4)
				s.ParentStation = tt.NewKey("ok_stop")
			}),
			expectedErrors: nil,
		},
		{
			name: "Valid: missing stop_lon for boarding area (location_type=4)",
			stop: newStop(func(s *Stop) {
				s.StopLon = tt.Float{}
				s.LocationType = tt.NewInt(4)
				s.ParentStation = tt.NewKey("ok_stop")
			}),
			expectedErrors: nil,
		},
		{
			name: "Invalid location_type = 6",
			stop: newStop(func(s *Stop) {
				s.LocationType = tt.NewInt(6)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:location_type"),
		},
		{
			name: "Invalid wheelchair_boarding > 2",
			stop: newStop(func(s *Stop) {
				s.WheelchairBoarding = tt.NewInt(3)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:wheelchair_boarding"),
		},
		{
			name: "Missing parent_station for entrance (location_type=2)",
			stop: newStop(func(s *Stop) {
				s.LocationType = tt.NewInt(2)
				s.ParentStation = tt.Key{}
			}),
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:parent_station"),
		},
		{
			name: "Station (location_type=1) with parent_station",
			stop: newStop(func(s *Stop) {
				s.LocationType = tt.NewInt(1)
				s.ParentStation = tt.NewKey("station")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:parent_station"),
		},
		{
			name: "Missing stop_lat for stop (location_type=0) - NaN equivalent",
			stop: newStop(func(s *Stop) {
				s.StopLat = tt.Float{}
				s.LocationType = tt.NewInt(0)
			}),
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:stop_lat"),
		},
		{
			name: "Missing stop_lon for stop (location_type=0) - NaN equivalent",
			stop: newStop(func(s *Stop) {
				s.StopLon = tt.Float{}
				s.LocationType = tt.NewInt(0)
			}),
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:stop_lon"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.stop)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

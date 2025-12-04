package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestRoute_Errors(t *testing.T) {
	newRoute := func(fn func(*Route)) *Route {
		route := &Route{
			RouteID:           tt.NewString("ok"),
			AgencyID:          tt.NewKey("ok"),
			RouteDesc:         tt.NewString("valid route"),
			RouteLongName:     tt.NewString("valid route 1"),
			RouteShortName:    tt.NewString("ok"),
			RouteType:         tt.NewInt(3),
			RouteURL:          tt.NewUrl("http://example.com"),
			RouteColor:        tt.NewColor("#ff0000"),
			RouteTextColor:    tt.NewColor("#00ff00"),
			RouteSortOrder:    tt.NewInt(0),
			NetworkID:         tt.NewString("ok"),
			ContinuousPickup:  tt.NewInt(0),
			ContinuousDropOff: tt.NewInt(0),
		}
		if fn != nil {
			fn(route)
		}
		return route
	}

	tests := []struct {
		name           string
		route          *Route
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid route",
			route:          newRoute(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing route_id (required field)",
			route: newRoute(func(r *Route) {
				r.RouteID = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:route_id"),
		},
		{
			name: "Missing route_short_name and route_long_name",
			route: newRoute(func(r *Route) {
				r.RouteShortName = tt.String{}
				r.RouteLongName = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:route_short_name"),
		},
		{
			name: "Invalid route_type (negative)",
			route: newRoute(func(r *Route) {
				r.RouteType = tt.NewInt(-1)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:route_type"),
		},
		{
			name: "Invalid route_type (too large)",
			route: newRoute(func(r *Route) {
				r.RouteType = tt.NewInt(1234567)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:route_type"),
		},
		{
			name: "Invalid route_url",
			route: newRoute(func(r *Route) {
				r.RouteURL = tt.NewUrl("abcxyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:route_url"),
		},
		{
			name: "Invalid route_color",
			route: newRoute(func(r *Route) {
				r.RouteColor = tt.NewColor("xyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:route_color"),
		},
		{
			name: "Invalid route_text_color",
			route: newRoute(func(r *Route) {
				r.RouteTextColor = tt.NewColor("xyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:route_text_color"),
		},
		{
			name: "Invalid continuous_pickup",
			route: newRoute(func(r *Route) {
				r.ContinuousPickup = tt.NewInt(100)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:continuous_pickup"),
		},
		{
			name: "Invalid continuous_drop_off",
			route: newRoute(func(r *Route) {
				r.ContinuousDropOff = tt.NewInt(100)
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:continuous_drop_off"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.route)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

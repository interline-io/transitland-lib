package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestAttribution_Errors(t *testing.T) {
	newAttribution := func(fn func(*Attribution)) *Attribution {
		attribution := &Attribution{
			AttributionID:    tt.NewString("ok"),
			OrganizationName: tt.NewString("test organization"),
			IsProducer:       tt.NewInt(1),
			IsOperator:       tt.NewInt(0),
			IsAuthority:      tt.NewInt(0),
			AttributionURL:   tt.NewUrl("http://interline.io"),
			AttributionEmail: tt.NewEmail("info@interline.io"),
			AttributionPhone: tt.NewString("510-555-5555"),
		}
		if fn != nil {
			fn(attribution)
		}
		return attribution
	}

	testcases := []struct {
		name           string
		entity         *Attribution
		expectedErrors []testutil.ExpectError
	}{
		{
			name:           "Valid attribution",
			entity:         newAttribution(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing organization_name",
			entity: newAttribution(func(a *Attribution) {
				a.OrganizationName = tt.String{}
			}),
			expectedErrors: PE("RequiredFieldError:organization_name"),
		},
		{
			name: "Missing is_producer (no role specified)",
			entity: newAttribution(func(a *Attribution) {
				a.IsProducer = tt.NewInt(0)
				a.IsOperator = tt.NewInt(0)
				a.IsAuthority = tt.NewInt(0)
			}),
			expectedErrors: PE("ConditionallyRequiredFieldError:is_producer"),
		},
		{
			name: "Invalid is_producer value",
			entity: newAttribution(func(a *Attribution) {
				a.IsProducer = tt.NewInt(100)
			}),
			expectedErrors: PE("InvalidFieldError:is_producer"),
		},
		{
			name: "Invalid is_operator value",
			entity: newAttribution(func(a *Attribution) {
				a.IsOperator = tt.NewInt(100)
			}),
			expectedErrors: PE("InvalidFieldError:is_operator"),
		},
		{
			name: "Invalid is_authority value",
			entity: newAttribution(func(a *Attribution) {
				a.IsAuthority = tt.NewInt(100)
			}),
			expectedErrors: PE("InvalidFieldError:is_authority"),
		},
		{
			name: "Invalid attribution_email",
			entity: newAttribution(func(a *Attribution) {
				a.AttributionEmail = tt.NewEmail("xyz")
			}),
			expectedErrors: PE("InvalidFieldError:attribution_email"),
		},
		{
			name: "Invalid attribution_url",
			entity: newAttribution(func(a *Attribution) {
				a.AttributionURL = tt.NewUrl("xyz")
			}),
			expectedErrors: PE("InvalidFieldError:attribution_url"),
		},
		{
			name: "agency_id conflicts with route_id",
			entity: newAttribution(func(a *Attribution) {
				a.AgencyID = tt.NewKey("agency")
				a.RouteID = tt.NewKey("route")
			}),
			expectedErrors: PE("ConditionallyForbiddenFieldError:route_id"),
		},
		{
			name: "agency_id conflicts with trip_id",
			entity: newAttribution(func(a *Attribution) {
				a.AgencyID = tt.NewKey("agency")
				a.TripID = tt.NewKey("trip")
			}),
			expectedErrors: PE("ConditionallyForbiddenFieldError:trip_id"),
		},
		{
			name: "route_id conflicts with trip_id",
			entity: newAttribution(func(a *Attribution) {
				a.RouteID = tt.NewKey("route")
				a.TripID = tt.NewKey("trip")
			}),
			expectedErrors: PE("ConditionallyForbiddenFieldError:trip_id"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

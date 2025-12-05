package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestFareRule_Errors(t *testing.T) {
	tests := []struct {
		name           string
		fareRule       *FareRule
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: basic fare rule",
			fareRule: &FareRule{
				FareID:        tt.NewString("fare1"),
				RouteID:       tt.NewKey("route1"),
				OriginID:      tt.NewString("zone1"),
				DestinationID: tt.NewString("zone2"),
				ContainsID:    tt.NewString("zone3"),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing fare_id",
			fareRule: &FareRule{
				RouteID: tt.NewKey("route1"),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:fare_id"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.fareRule)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

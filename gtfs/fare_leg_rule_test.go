package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestFareLegRule_Errors(t *testing.T) {
	tests := []struct {
		name           string
		fareLegRule    *FareLegRule
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: basic fare leg rule",
			fareLegRule: &FareLegRule{
				LegGroupID:    tt.NewString("lg1"),
				FromAreaID:    tt.NewString("area1"),
				ToAreaID:      tt.NewString("area2"),
				NetworkID:     tt.NewString("net1"),
				FareProductID: tt.NewString("product1"),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing fare_product_id",
			fareLegRule: &FareLegRule{
				LegGroupID: tt.NewString("lg1"),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:fare_product_id"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.fareLegRule)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

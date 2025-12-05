package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestFareTransferRule_Errors(t *testing.T) {
	tests := []struct {
		name             string
		fareTransferRule *FareTransferRule
		expectedErrors   []ExpectError
	}{
		{
			name: "Valid: basic fare transfer rule",
			fareTransferRule: &FareTransferRule{
				FareTransferType: tt.NewInt(0),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing fare_transfer_type",
			fareTransferRule: &FareTransferRule{
				FareProductID: tt.NewString("product1"),
			},
			expectedErrors: PE("RequiredFieldError:fare_transfer_type"),
		},
		{
			name: "Invalid: transfer_count required if from_leg_group_id == to_leg_group_id",
			fareTransferRule: &FareTransferRule{
				FromLegGroupID:   tt.NewString("lg1"),
				ToLegGroupID:     tt.NewString("lg1"),
				FareProductID:    tt.NewString("test"),
				FareTransferType: tt.NewInt(1),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:transfer_count"),
		},
		{
			name: "Invalid: transfer_count forbidden if from_leg_group_id != to_leg_group_id (implicit)",
			// Note: The code says: if ent.TransferCount.Valid { if !legGroupsValidEqual { ... ConditionallyForbiddenFieldError } }
			// legGroupsValidEqual is true only if both valid and equal.
			// If one is missing, legGroupsValidEqual is false.
			// So transfer_count is forbidden if leg groups are not (valid AND equal).
			fareTransferRule: &FareTransferRule{
				FareProductID:    tt.NewString("test"),
				FareTransferType: tt.NewInt(1),
				TransferCount:    tt.NewInt(1),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:transfer_count"),
		},
		{
			name: "Invalid: duration_limit required (actually duration_limit_type missing)",
			// Code: else if ent.DurationLimit.Valid { errs = append(errs, causes.NewConditionallyRequiredFieldError("duration_limit")) }
			fareTransferRule: &FareTransferRule{
				FareProductID:    tt.NewString("test"),
				FareTransferType: tt.NewInt(1),
				DurationLimit:    tt.NewInt(3600),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:duration_limit"),
		},
		{
			name: "Invalid: duration_limit forbidden (actually duration_limit missing but type present)",
			// Code: if ent.DurationLimitType.Valid { if !ent.DurationLimit.Valid { ... ConditionallyForbiddenFieldError("duration_limit") } }
			fareTransferRule: &FareTransferRule{
				FareProductID:     tt.NewString("test"),
				FareTransferType:  tt.NewInt(1),
				DurationLimitType: tt.NewInt(1),
			},
			expectedErrors: PE("ConditionallyForbiddenFieldError:duration_limit"),
		},
		{
			name: "Invalid: invalid fare_transfer_type (-1)",
			fareTransferRule: &FareTransferRule{
				FareTransferType: tt.NewInt(-1),
			},
			expectedErrors: PE("InvalidFieldError:fare_transfer_type"),
		},
		{
			name: "Invalid: invalid fare_transfer_type (3)",
			fareTransferRule: &FareTransferRule{
				FareTransferType: tt.NewInt(3), // Enum 0,1,2
			},
			expectedErrors: PE("InvalidFieldError:fare_transfer_type"),
		},
		{
			name: "Invalid: invalid transfer_count (-2)",
			fareTransferRule: &FareTransferRule{
				FromLegGroupID:   tt.NewString("lg1"),
				ToLegGroupID:     tt.NewString("lg1"),
				FareTransferType: tt.NewInt(1),
				TransferCount:    tt.NewInt(-2), // Range -1,
			},
			expectedErrors: PE("InvalidFieldError:transfer_count"),
		},
		{
			name: "Invalid: invalid duration_limit (-1)",
			fareTransferRule: &FareTransferRule{
				FareProductID:     tt.NewString("1"),
				FareTransferType:  tt.NewInt(1),
				DurationLimit:     tt.NewInt(-1),
				DurationLimitType: tt.NewInt(1),
			},
			expectedErrors: PE("InvalidFieldError:duration_limit"),
		},
		{
			name: "Invalid: invalid duration_limit_type (-1)",
			fareTransferRule: &FareTransferRule{
				FareProductID:     tt.NewString("1"),
				FareTransferType:  tt.NewInt(1),
				DurationLimit:     tt.NewInt(3600),
				DurationLimitType: tt.NewInt(-1),
			},
			expectedErrors: PE("InvalidFieldError:duration_limit_type"),
		},
		{
			name: "Invalid: invalid duration_limit_type (4)",
			fareTransferRule: &FareTransferRule{
				FareProductID:     tt.NewString("1"),
				FareTransferType:  tt.NewInt(1),
				DurationLimit:     tt.NewInt(3600),
				DurationLimitType: tt.NewInt(4), // Enum 0,1,2,3
			},
			expectedErrors: PE("InvalidFieldError:duration_limit_type"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.fareTransferRule)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestFareLegJoinRule_Errors(t *testing.T) {
	tests := []struct {
		name            string
		fareLegJoinRule *FareLegJoinRule
		expectedErrors  []ExpectError
	}{
		{
			name: "Valid: network to network",
			fareLegJoinRule: &FareLegJoinRule{
				FromNetworkID: tt.NewString("net1"),
				ToNetworkID:   tt.NewString("net2"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: with stops",
			fareLegJoinRule: &FareLegJoinRule{
				FromNetworkID: tt.NewString("net1"),
				ToNetworkID:   tt.NewString("net2"),
				FromStopID:    tt.NewString("s1"),
				ToStopID:      tt.NewString("s2"),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing from_network_id",
			fareLegJoinRule: &FareLegJoinRule{
				ToNetworkID: tt.NewString("net2"),
			},
			expectedErrors: PE("RequiredFieldError:from_network_id"),
		},
		{
			name: "Invalid: from_stop_id without to_stop_id",
			fareLegJoinRule: &FareLegJoinRule{
				FromNetworkID: tt.NewString("net1"),
				ToNetworkID:   tt.NewString("net2"),
				FromStopID:    tt.NewString("s1"),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:to_stop_id"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.fareLegJoinRule)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

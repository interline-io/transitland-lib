package rules

import (
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

func TestTransferStopLocationTypeCheck(t *testing.T) {
	checker := TransferStopLocationTypeCheck{}

	// Setup stops with different location types
	stops := []gtfs.Stop{
		{StopID: tt.NewString("stop0"), LocationType: tt.NewInt(0)}, // valid - stop/platform
		{StopID: tt.NewString("stop1"), LocationType: tt.NewInt(1)}, // valid - station
		{StopID: tt.NewString("stop2"), LocationType: tt.NewInt(2)}, // invalid - entrance
		{StopID: tt.NewString("stop3"), LocationType: tt.NewInt(3)}, // invalid - generic node
		{StopID: tt.NewString("stop4"), LocationType: tt.NewInt(4)}, // invalid - boarding area
	}

	// Add stops to checker
	for _, stop := range stops {
		checker.AfterWrite(stop.StopID.Val, &stop, nil)
	}

	testcases := []struct {
		name          string
		transfer      gtfs.Transfer
		wantErrors    int      // expected number of errors
		errorContains []string // expected substrings in error messages
	}{
		{
			name: "stop_to_stop_transfer",
			transfer: gtfs.Transfer{
				FromStopID:   tt.NewKey("stop0"),
				ToStopID:     tt.NewKey("stop0"),
				TransferType: tt.NewInt(2),
			},
			wantErrors: 0,
		},
		{
			name: "stop_to_station_transfer",
			transfer: gtfs.Transfer{
				FromStopID:   tt.NewKey("stop0"),
				ToStopID:     tt.NewKey("stop1"),
				TransferType: tt.NewInt(2),
			},
			wantErrors: 0,
		},
		{
			name: "station_to_station_transfer",
			transfer: gtfs.Transfer{
				FromStopID:   tt.NewKey("stop1"),
				ToStopID:     tt.NewKey("stop1"),
				TransferType: tt.NewInt(2),
			},
			wantErrors: 0,
		},
		{
			name: "entrance_to_generic_node_transfer",
			transfer: gtfs.Transfer{
				FromStopID:   tt.NewKey("stop2"),
				ToStopID:     tt.NewKey("stop3"),
				TransferType: tt.NewInt(2),
			},
			wantErrors: 2,
			errorContains: []string{
				"transfer field 'from_stop_id' references stop 'stop2'",
				"transfer field 'to_stop_id' references stop 'stop3'",
			},
		},
		{
			name: "stop_to_boarding_area_transfer",
			transfer: gtfs.Transfer{
				FromStopID:   tt.NewKey("stop0"),
				ToStopID:     tt.NewKey("stop4"),
				TransferType: tt.NewInt(2),
			},
			wantErrors: 1,
			errorContains: []string{
				"transfer field 'to_stop_id' references stop 'stop4'",
			},
		},
		{
			name: "transfer_type_no_stops",
			transfer: gtfs.Transfer{
				FromTripID:   tt.NewKey("trip1"),
				ToTripID:     tt.NewKey("trip2"),
				TransferType: tt.NewInt(4),
			},
			wantErrors: 0,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := checker.Validate(&tc.transfer)

			// Check error count
			if got := len(errs); got != tc.wantErrors {
				t.Errorf("got %d errors, want %d", got, tc.wantErrors)
			}

			// Check error messages if specified
			if len(tc.errorContains) > 0 {
				if len(errs) != len(tc.errorContains) {
					t.Errorf("got %d errors, want %d", len(errs), len(tc.errorContains))
				}
				for i, want := range tc.errorContains {
					if i >= len(errs) {
						break
					}
					if !strings.Contains(errs[i].Error(), want) {
						t.Errorf("error %d: got %q, want it to contain %q", i, errs[i].Error(), want)
					}
				}
			}
		})
	}
}

package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestTransfer_Errors(t *testing.T) {
	newTransfer := func(fn func(*Transfer)) *Transfer {
		transfer := &Transfer{
			FromStopID:      tt.NewKey("ok1"),
			ToStopID:        tt.NewKey("ok2"),
			TransferType:    tt.NewInt(0),
			MinTransferTime: tt.NewInt(600),
		}
		if fn != nil {
			fn(transfer)
		}
		return transfer
	}

	tests := []struct {
		name           string
		transfer       *Transfer
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid transfer",
			transfer:       newTransfer(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing from_stop_id for transfer_type=1",
			transfer: newTransfer(func(t *Transfer) {
				t.FromStopID = tt.Key{}
				t.ToTripID = tt.NewKey("ok")
				t.TransferType = tt.NewInt(1)
			}),
			expectedErrors: PE("ConditionallyRequiredFieldError:from_stop_id"),
		},
		{
			name: "Missing to_stop_id for transfer_type=1",
			transfer: newTransfer(func(t *Transfer) {
				t.ToStopID = tt.Key{}
				t.FromTripID = tt.NewKey("ok")
				t.TransferType = tt.NewInt(1)
			}),
			expectedErrors: PE("ConditionallyRequiredFieldError:to_stop_id"),
		},
		{
			name: "Missing from_trip_id for transfer_type=4",
			transfer: newTransfer(func(t *Transfer) {
				t.ToTripID = tt.NewKey("ok")
				t.TransferType = tt.NewInt(4)
			}),
			expectedErrors: PE("ConditionallyRequiredFieldError:from_trip_id"),
		},
		{
			name: "Missing to_trip_id for transfer_type=4",
			transfer: newTransfer(func(t *Transfer) {
				t.FromTripID = tt.NewKey("ok")
				t.TransferType = tt.NewInt(4)
			}),
			expectedErrors: PE("ConditionallyRequiredFieldError:to_trip_id"),
		},
		{
			name: "Missing from_trip_id for transfer_type=5",
			transfer: newTransfer(func(t *Transfer) {
				t.ToTripID = tt.NewKey("ok")
				t.TransferType = tt.NewInt(5)
			}),
			expectedErrors: PE("ConditionallyRequiredFieldError:from_trip_id"),
		},
		{
			name: "Missing to_trip_id for transfer_type=5",
			transfer: newTransfer(func(t *Transfer) {
				t.FromTripID = tt.NewKey("ok")
				t.TransferType = tt.NewInt(5)
			}),
			expectedErrors: PE("ConditionallyRequiredFieldError:to_trip_id"),
		},
		{
			name: "Invalid transfer_type (negative)",
			transfer: newTransfer(func(t *Transfer) {
				t.TransferType = tt.NewInt(-1)
			}),
			expectedErrors: PE("InvalidFieldError:transfer_type"),
		},
		{
			name: "Invalid transfer_type (too large)",
			transfer: newTransfer(func(t *Transfer) {
				t.TransferType = tt.NewInt(6)
			}),
			expectedErrors: PE("InvalidFieldError:transfer_type"),
		},
		{
			name: "Invalid min_transfer_time (negative)",
			transfer: newTransfer(func(t *Transfer) {
				t.TransferType = tt.NewInt(2)
				t.MinTransferTime = tt.NewInt(-1)
			}),
			expectedErrors: PE("InvalidFieldError:min_transfer_time"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.transfer)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

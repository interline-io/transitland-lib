package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestFareAttribute_Errors(t *testing.T) {
	tests := []struct {
		name           string
		fareAttribute  *FareAttribute
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: basic fare attribute",
			fareAttribute: &FareAttribute{
				FareID:           tt.NewString("fare1"),
				Price:            tt.NewFloat(2.0),
				CurrencyType:     tt.NewCurrency("USD"),
				PaymentMethod:    tt.NewInt(1),
				Transfers:        tt.NewInt(1),
				AgencyID:         tt.NewKey("agency1"),
				TransferDuration: tt.NewInt(600),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: special case empty transfers",
			fareAttribute: &FareAttribute{
				FareID:        tt.NewString("fare2"),
				Price:         tt.NewFloat(2.0),
				CurrencyType:  tt.NewCurrency("USD"),
				PaymentMethod: tt.NewInt(1),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing fare_id",
			fareAttribute: &FareAttribute{
				Price:         tt.NewFloat(2.0),
				CurrencyType:  tt.NewCurrency("USD"),
				PaymentMethod: tt.NewInt(1),
				Transfers:     tt.NewInt(1),
			},
			expectedErrors: PE("RequiredFieldError:fare_id"),
		},
		{
			name: "Invalid: missing price",
			fareAttribute: &FareAttribute{
				FareID:        tt.NewString("fare3"),
				CurrencyType:  tt.NewCurrency("USD"),
				PaymentMethod: tt.NewInt(1),
				Transfers:     tt.NewInt(1),
			},
			expectedErrors: PE("RequiredFieldError:price"),
		},
		{
			name: "Invalid: missing currency_type",
			fareAttribute: &FareAttribute{
				FareID:        tt.NewString("fare4"),
				Price:         tt.NewFloat(2.0),
				PaymentMethod: tt.NewInt(1),
				Transfers:     tt.NewInt(1),
			},
			expectedErrors: PE("RequiredFieldError:currency_type"),
		},
		{
			name: "Invalid: missing payment_method",
			fareAttribute: &FareAttribute{
				FareID:       tt.NewString("fare5"),
				Price:        tt.NewFloat(2.0),
				CurrencyType: tt.NewCurrency("USD"),
				Transfers:    tt.NewInt(1),
			},
			expectedErrors: PE("RequiredFieldError:payment_method"),
		},
		{
			name: "Invalid: invalid price (< 0)",
			fareAttribute: &FareAttribute{
				FareID:        tt.NewString("fare6"),
				Price:         tt.NewFloat(-1.0),
				CurrencyType:  tt.NewCurrency("USD"),
				PaymentMethod: tt.NewInt(1),
				Transfers:     tt.NewInt(1),
			},
			expectedErrors: PE("InvalidFieldError:price"),
		},
		{
			name: "Invalid: invalid payment_method (2)",
			fareAttribute: &FareAttribute{
				FareID:        tt.NewString("fare7"),
				Price:         tt.NewFloat(2.0),
				CurrencyType:  tt.NewCurrency("USD"),
				PaymentMethod: tt.NewInt(2),
				Transfers:     tt.NewInt(1),
			},
			expectedErrors: PE("InvalidFieldError:payment_method"),
		},
		{
			name: "Invalid: invalid transfers (3)",
			fareAttribute: &FareAttribute{
				FareID:        tt.NewString("fare8"),
				Price:         tt.NewFloat(2.0),
				CurrencyType:  tt.NewCurrency("USD"),
				PaymentMethod: tt.NewInt(1),
				Transfers:     tt.NewInt(3),
			},
			expectedErrors: PE("InvalidFieldError:transfers"),
		},
		{
			name: "Invalid: invalid transfer_duration (< 0)",
			fareAttribute: &FareAttribute{
				FareID:           tt.NewString("fare9"),
				Price:            tt.NewFloat(2.0),
				CurrencyType:     tt.NewCurrency("USD"),
				PaymentMethod:    tt.NewInt(1),
				Transfers:        tt.NewInt(1),
				TransferDuration: tt.NewInt(-1),
			},
			expectedErrors: PE("InvalidFieldError:transfer_duration"),
		},
		// Note: Currency validation might depend on external libraries or strictness settings.
		// The bad-entities test expects InvalidFieldError:currency_type for "xyz".
		// Assuming tt.Currency validation handles this if implemented, or it might pass if it just checks string presence.
		// Let's include it and see if it fails.
		{
			name: "Invalid: invalid currency_type",
			fareAttribute: &FareAttribute{
				FareID:        tt.NewString("fare10"),
				Price:         tt.NewFloat(2.0),
				CurrencyType:  tt.NewCurrency("xyz"),
				PaymentMethod: tt.NewInt(1),
				Transfers:     tt.NewInt(1),
			},
			expectedErrors: PE("InvalidFieldError:currency_type"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.fareAttribute)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}

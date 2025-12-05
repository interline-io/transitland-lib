package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestFareProduct_Errors(t *testing.T) {
	tests := []struct {
		name           string
		fareProduct    *FareProduct
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: basic fare product",
			fareProduct: &FareProduct{
				FareProductID:   tt.NewString("product1"),
				FareProductName: tt.NewString("Product 1"),
				Amount:          tt.NewCurrencyAmount(2.50),
				Currency:        tt.NewCurrency("USD"),
			},
			expectedErrors: nil,
		},
		{
			name: "Valid: fare product with duration",
			fareProduct: &FareProduct{
				FareProductID:  tt.NewString("product2"),
				Amount:         tt.NewCurrencyAmount(0),
				Currency:       tt.NewCurrency("USD"),
				DurationStart:  tt.NewInt(1),
				DurationAmount: tt.NewFloat(30),
				DurationUnit:   tt.NewInt(1), // Minute
				DurationType:   tt.NewInt(1), // Fixed
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing fare_product_id",
			fareProduct: &FareProduct{
				Amount:   tt.NewCurrencyAmount(0),
				Currency: tt.NewCurrency("USD"),
			},
			expectedErrors: PE("RequiredFieldError:fare_product_id"),
		},
		{
			name: "Invalid: missing amount",
			fareProduct: &FareProduct{
				FareProductID: tt.NewString("product3"),
				Currency:      tt.NewCurrency("USD"),
			},
			expectedErrors: PE("RequiredFieldError:amount"),
		},
		{
			name: "Invalid: missing currency",
			fareProduct: &FareProduct{
				FareProductID: tt.NewString("product4"),
				Amount:        tt.NewCurrencyAmount(0),
			},
			expectedErrors: PE("RequiredFieldError:currency"),
		},
		{
			name: "Invalid: invalid duration_start",
			fareProduct: &FareProduct{
				FareProductID: tt.NewString("product5"),
				Amount:        tt.NewCurrencyAmount(0),
				Currency:      tt.NewCurrency("USD"),
				DurationStart: tt.NewInt(-1),
			},
			expectedErrors: PE("InvalidFieldError:duration_start"),
		},
		{
			name: "Invalid: invalid duration_amount (< 0)",
			fareProduct: &FareProduct{
				FareProductID:  tt.NewString("product6"),
				Amount:         tt.NewCurrencyAmount(0),
				Currency:       tt.NewCurrency("USD"),
				DurationAmount: tt.NewFloat(-1),
				DurationType:   tt.NewInt(1),
			},
			expectedErrors: PE("InvalidFieldError:duration_amount"),
		},
		{
			name: "Invalid: invalid duration_unit",
			fareProduct: &FareProduct{
				FareProductID: tt.NewString("product7"),
				Amount:        tt.NewCurrencyAmount(0),
				Currency:      tt.NewCurrency("USD"),
				DurationUnit:  tt.NewInt(-1),
			},
			expectedErrors: PE("InvalidFieldError:duration_unit"),
		},
		{
			name: "Invalid: invalid duration_type",
			fareProduct: &FareProduct{
				FareProductID:  tt.NewString("product8"),
				Amount:         tt.NewCurrencyAmount(0),
				Currency:       tt.NewCurrency("USD"),
				DurationAmount: tt.NewFloat(30),
				DurationType:   tt.NewInt(0),
			},
			expectedErrors: PE("InvalidFieldError:duration_type"),
		},
		{
			name: "Invalid: duration_type required if duration_amount present",
			fareProduct: &FareProduct{
				FareProductID:  tt.NewString("product9"),
				Amount:         tt.NewCurrencyAmount(0),
				Currency:       tt.NewCurrency("USD"),
				DurationAmount: tt.NewFloat(30),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:duration_type"),
		},
		{
			name: "Invalid: duration_amount required if duration_type present",
			fareProduct: &FareProduct{
				FareProductID: tt.NewString("product10"),
				Amount:        tt.NewCurrencyAmount(0),
				Currency:      tt.NewCurrency("USD"),
				DurationType:  tt.NewInt(1),
			},
			expectedErrors: PE("ConditionallyRequiredFieldError:duration_amount"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.fareProduct)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
